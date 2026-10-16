package mq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"backend-go/internal/config"
	"backend-go/internal/tracing"
	"backend-go/pkg/logger"
)

// ErrNotConnected 当前无可用 RabbitMQ 连接（重连进行中）
var ErrNotConnected = errors.New("RabbitMQ 连接不可用（重连中）")

// OnDeadLetter 消息重试耗尽、进入死信队列前的回调（由应用层注入，用于落死信表）
// 参数：来源队列、原交换机、原 routing key、消息体、最终失败原因
var OnDeadLetter func(queue, exchange, routingKey, body, reason string)

const (
	dlxExchange     = "vehicle.dlx" // 死信交换机（topic）
	telemetryDLQ    = "telemetry.dlq"
	controlDLQ      = "control.dlq"
	controlExchange = "vehicle.control"
	controlQueue    = "control.dispatch"
)

// MQ 封装 RabbitMQ 连接管理：自动重连 + channel 池 + 消费者自动恢复 + 死信兜底
type MQ struct {
	url           string
	exchange      string
	queue         string
	poolSize      int
	maxRetries    int
	reconnectMax  time.Duration

	mu      sync.RWMutex
	conn    *amqp.Connection
	readyCh chan struct{} // 连接状态变化时关闭并重建，用于唤醒等待者

	chPool chan *amqp.Channel // 发布 channel 池

	closing int32
	stopCh  chan struct{}
	done    chan struct{} // manage goroutine 退出后关闭
}

var Client *MQ

// Init 建立初始连接（启动期快速失败），随后由后台 goroutine 维护连接生命周期
func Init(cfg config.RabbitMQConfig) error {
	m := &MQ{
		url:          cfg.URL,
		exchange:     cfg.TelemetryExchange,
		queue:        cfg.TelemetryQueue,
		poolSize:     cfg.ChannelPoolSize,
		maxRetries:   cfg.MaxRetries,
		reconnectMax: cfg.ReconnectMaxDelay(),
		readyCh:      make(chan struct{}),
		stopCh:       make(chan struct{}),
		done:         make(chan struct{}),
	}
	if m.poolSize <= 0 {
		m.poolSize = 8
	}
	if m.maxRetries <= 0 {
		m.maxRetries = 3
	}
	m.chPool = make(chan *amqp.Channel, m.poolSize)

	// 启动阶段：有限次重试，连不上直接失败（保持快速反馈）
	var conn *amqp.Connection
	var err error
	for i := 1; i <= 5; i++ {
		conn, err = amqp.Dial(cfg.URL)
		if err == nil {
			if err = m.declareTopology(conn); err == nil {
				break
			}
			_ = conn.Close()
		}
		logger.L.Warn("RabbitMQ 初始连接失败，2s 后重试",
			zap.Int("attempt", i), zap.Error(err))
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	Client = m
	m.setConn(conn) // 同步就绪，避免 Init 返回后立即发布遇到竞态
	go m.manage(conn)
	return nil
}

// declareTopology 声明交换机/队列/绑定（含死信拓扑），每次（重）建连后都会执行
func (m *MQ) declareTopology(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("创建 channel 失败: %w", err)
	}
	defer ch.Close()

	// 死信交换机
	if err := ch.ExchangeDeclare(dlxExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明死信交换机失败: %w", err)
	}

	// 队列声明放独立 channel 执行：旧版队列参数不一致会触发 406 并关闭 channel，
	// ensureQueue 内部负责删除重建（本地开发环境的拓扑自动迁移）
	for _, q := range []struct {
		name   string
		withDLX bool
	}{
		{m.queue, true},
		{telemetryDLQ, false},
		{controlQueue, true},
		{controlDLQ, false},
	} {
		if err := ensureQueue(conn, q.name, q.withDLX); err != nil {
			return fmt.Errorf("声明队列 %s 失败: %w", q.name, err)
		}
	}

	// 以下绑定均在主 channel 上执行（交换机和队列此时均已就绪）
	// 遥测：主交换机 + 落库队列 + 死信队列
	if err := ch.ExchangeDeclare(m.exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明交换机失败: %w", err)
	}
	if err := ch.QueueBind(m.queue, "vehicle.telemetry.#", m.exchange, false, nil); err != nil {
		return fmt.Errorf("绑定队列失败: %w", err)
	}
	if err := ch.QueueBind(telemetryDLQ, "vehicle.telemetry.#", dlxExchange, false, nil); err != nil {
		return fmt.Errorf("绑定遥测死信队列失败: %w", err)
	}

	// 控车：交换机 + 分发队列 + 死信队列
	if err := ch.ExchangeDeclare(controlExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明控车交换机失败: %w", err)
	}
	if err := ch.QueueBind(controlQueue, "vehicle.control.#", controlExchange, false, nil); err != nil {
		return fmt.Errorf("绑定控车分发队列失败: %w", err)
	}
	if err := ch.QueueBind(controlDLQ, "vehicle.control.#", dlxExchange, false, nil); err != nil {
		return fmt.Errorf("绑定控车死信队列失败: %w", err)
	}
	return nil
}

// ensureQueue 声明队列（可选挂死信）；若旧版同名队列参数不一致（406）则删除重建。
// 每次操作使用独立 channel，避免 406 错误污染调用方的 channel。
func ensureQueue(conn *amqp.Connection, name string, withDLX bool) error {
	var args amqp.Table
	if withDLX {
		args = amqp.Table{"x-dead-letter-exchange": dlxExchange}
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	_, declareErr := ch.QueueDeclare(name, true, false, false, false, args)
	_ = ch.Close()
	if declareErr == nil {
		return nil
	}
	if !isPreconditionFailed(declareErr) {
		return declareErr
	}

	// 旧版队列（无死信配置）：删除后重建
	logger.L.Warn("检测到旧版队列参数不一致，删除重建以挂载死信（本地拓扑迁移）",
		zap.String("queue", name))
	delCh, err := conn.Channel()
	if err != nil {
		return err
	}
	if _, err := delCh.QueueDelete(name, false, false, false); err != nil {
		_ = delCh.Close()
		return fmt.Errorf("删除旧队列失败: %w", err)
	}
	_, err = delCh.QueueDeclare(name, true, false, false, false, args)
	_ = delCh.Close()
	return err
}

func isPreconditionFailed(err error) bool {
	var amqpErr *amqp.Error
	return errors.As(err, &amqpErr) && amqpErr.Code == 406
}

// manage 连接生命周期管理：断线后指数退避自动重连，恢复后重建拓扑与消费者
func (m *MQ) manage(conn *amqp.Connection) {
	defer close(m.done)
	backoff := time.Second
	for {
		if conn == nil {
			if m.isClosing() {
				return
			}
			c, err := amqp.Dial(m.url)
			if err != nil {
				logger.L.Warn("RabbitMQ 重连失败，稍后重试",
					zap.Duration("backoff", backoff), zap.Error(err))
				if !sleepOrStop(backoff, m.stopCh) {
					return
				}
				if backoff < m.reconnectMax {
					backoff *= 2
					if backoff > m.reconnectMax {
						backoff = m.reconnectMax
					}
				}
				continue
			}
			if err := m.declareTopology(c); err != nil {
				logger.L.Warn("RabbitMQ 拓扑重建失败，稍后重试", zap.Error(err))
				_ = c.Close()
				if !sleepOrStop(backoff, m.stopCh) {
					return
				}
				continue
			}
			conn = c
		}

		backoff = time.Second
		m.setConn(conn)
		logger.L.Info("RabbitMQ 连接已建立")

		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		select {
		case e := <-closeCh:
			logger.L.Warn("RabbitMQ 连接断开，开始自动重连", zap.Any("reason", e))
			m.setConn(nil)
			m.drainPool()
			conn = nil
		case <-m.stopCh:
			m.setConn(nil)
			m.drainPool()
			_ = conn.Close()
			return
		}
	}
}

// setConn 更新当前连接并唤醒所有等待者（conn 为 nil 表示断开）
func (m *MQ) setConn(conn *amqp.Connection) {
	m.mu.Lock()
	m.conn = conn
	close(m.readyCh)
	m.readyCh = make(chan struct{})
	m.mu.Unlock()
}

// currentConn 获取当前可用连接（无则返回 nil）
func (m *MQ) currentConn() *amqp.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.conn != nil && !m.conn.IsClosed() {
		return m.conn
	}
	return nil
}

// waitConnected 阻塞等待可用连接；ctx 取消或关闭中返回 nil
func (m *MQ) waitConnected(ctx context.Context) *amqp.Connection {
	for {
		m.mu.RLock()
		conn, ready := m.conn, m.readyCh
		m.mu.RUnlock()
		if conn != nil && !conn.IsClosed() {
			return conn
		}
		select {
		case <-ready:
		case <-ctx.Done():
			return nil
		case <-m.stopCh:
			return nil
		}
	}
}

// ---------------- 发布（channel 池化） ----------------

// getChannel 优先从池中取用，否则基于当前连接新建。
// 所有 channel 统一开启 Publisher Confirms 模式：
// 快路径（遥测）不等回执，可靠路径（Outbox/死信重投）等待 broker ack。
func (m *MQ) getChannel() (*amqp.Channel, error) {
	select {
	case ch := <-m.chPool:
		if !ch.IsClosed() {
			return ch, nil
		}
	default:
	}
	conn := m.currentConn()
	if conn == nil {
		return nil, ErrNotConnected
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("开启发布确认模式失败: %w", err)
	}
	return ch, nil
}

// putChannel 归还 channel 到池；池满或已关闭则直接关闭
func (m *MQ) putChannel(ch *amqp.Channel) {
	if ch == nil || ch.IsClosed() {
		return
	}
	select {
	case m.chPool <- ch:
	default:
		_ = ch.Close()
	}
}

// drainPool 连接断开后清空池中失效 channel
func (m *MQ) drainPool() {
	for {
		select {
		case ch := <-m.chPool:
			_ = ch.Close()
		default:
			return
		}
	}
}

// publish 池化发布：失败丢弃 channel 并重试一次（应对连接刚断开的竞态）
// 同时产生 producer span 并注入 trace context，实现 HTTP → MQ → 消费 的全链路追踪。
//
// confirmed 区分两级可靠性语义：
//   - false（遥测快路径）：写完 socket 即返回，丢一帧可接受
//   - true（指令/死信重投）：阻塞等待 broker ack 回执，未确认视为失败由上游重试
func (m *MQ) publish(ctx context.Context, exchange, routingKey string, body []byte, confirmed bool) error {
	ctx, span := tracing.Tracer("mq").Start(ctx, "mq.publish "+routingKey,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", exchange),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		))
	defer span.End()

	headers := amqp.Table{}
	tracing.Inject(ctx, headers)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		ch, err := m.getChannel()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		dc, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey,
			false, false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Headers:      headers,
				Body:         body,
				Timestamp:    time.Now(),
			})
		if err == nil && confirmed {
			// 可靠路径：等待 broker ack（超时时间由调用方 ctx 控制）
			var acked bool
			acked, err = dc.WaitContext(ctx)
			if err == nil && !acked {
				err = fmt.Errorf("broker 返回 nack，消息未被确认")
			}
		}
		if err == nil {
			m.putChannel(ch)
			return nil
		}
		lastErr = err
		_ = ch.Close() // 坏 channel 丢弃，不归还
	}
	span.RecordError(lastErr)
	span.SetStatus(codes.Error, lastErr.Error())
	return fmt.Errorf("发布消息失败: %w", lastErr)
}

// PublishTelemetry 发布车况数据（routing key: vehicle.telemetry.<vin>）
// 快路径：不等 broker 回执，高吞吐，丢帧可容忍
func (m *MQ) PublishTelemetry(ctx context.Context, vin string, body []byte) error {
	return m.publish(ctx, m.exchange, "vehicle.telemetry."+vin, body, false)
}

// PublishRaw 按原始 exchange/routingKey 可靠发布（供 Outbox 投递器与死信重投使用）
// 可靠路径：阻塞等待 broker ack，未确认返回错误由上游重试
func (m *MQ) PublishRaw(ctx context.Context, exchange, routingKey string, body []byte) error {
	return m.publish(ctx, exchange, routingKey, body, true)
}

// ---------------- 消费（断线自动恢复 + 重试 + 死信） ----------------

// ConsumeTelemetry 消费遥测落库队列；连接断开后自动等待重连并恢复消费，直到 ctx 取消
func (m *MQ) ConsumeTelemetry(ctx context.Context, handler func(body []byte) error) error {
	return m.consumeLoop(ctx, "telemetry-consumer", func(conn *amqp.Connection) (*amqp.Channel, <-chan amqp.Delivery, error) {
		ch, err := conn.Channel()
		if err != nil {
			return nil, nil, err
		}
		// 消费者 QoS：一次最多预取 10 条，防止压垮消费者
		if err := ch.Qos(10, 0, false); err != nil {
			_ = ch.Close()
			return nil, nil, err
		}
		msgs, err := ch.Consume(m.queue, "telemetry-consumer", false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			return nil, nil, err
		}
		return ch, msgs, nil
	}, m.queue, handler)
}

// ConsumeControl 消费控车相关队列；queue 为空时创建临时排他队列并按 bindingKeys 逐个绑定
func (m *MQ) ConsumeControl(ctx context.Context, queue string, bindingKeys []string, consumerTag string, handler func(body []byte) error) error {
	return m.consumeLoop(ctx, consumerTag, func(conn *amqp.Connection) (*amqp.Channel, <-chan amqp.Delivery, error) {
		ch, err := conn.Channel()
		if err != nil {
			return nil, nil, err
		}
		q := queue
		if q == "" {
			// 临时排他队列（如模拟车队），随连接断开自动删除，重连后重新声明
			tmp, err := ch.QueueDeclare("", false, true, true, false, nil)
			if err != nil {
				_ = ch.Close()
				return nil, nil, err
			}
			q = tmp.Name
			for _, key := range bindingKeys {
				if err := ch.QueueBind(q, key, controlExchange, false, nil); err != nil {
					_ = ch.Close()
					return nil, nil, err
				}
			}
		}
		msgs, err := ch.Consume(q, consumerTag, false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			return nil, nil, err
		}
		return ch, msgs, nil
	}, queue, handler)
}

// consumeLoop 消费者主循环：等待连接 → 建立消费 → 通道关闭后等待重连自动恢复
func (m *MQ) consumeLoop(ctx context.Context, name string,
	setup func(*amqp.Connection) (*amqp.Channel, <-chan amqp.Delivery, error),
	queueName string, handler func(body []byte) error) error {

	for {
		conn := m.waitConnected(ctx)
		if conn == nil {
			return nil // 关闭中或 ctx 取消
		}
		ch, msgs, err := setup(conn)
		if err != nil {
			logger.L.Warn("消费者建立失败，等待重试", zap.String("consumer", name), zap.Error(err))
			if !sleepOrStopCtx(ctx, 2*time.Second, m.stopCh) {
				return nil
			}
			continue
		}
		logger.L.Info("消费者已启动", zap.String("consumer", name), zap.String("queue", queueName))

		err = m.deliver(ctx, name, queueName, msgs, handler)
		_ = ch.Close()
		if ctx.Err() != nil || m.isClosing() {
			return nil
		}
		logger.L.Warn("消费者通道中断，等待重连后自动恢复", zap.String("consumer", name), zap.Error(err))
	}
}

// deliver 消息投递循环：逐条处理，带应用层重试与死信兜底
func (m *MQ) deliver(ctx context.Context, name, queueName string, msgs <-chan amqp.Delivery, handler func(body []byte) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopCh:
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消息通道被关闭")
			}
			m.processWithRetry(ctx, name, queueName, msg, handler)
		}
	}
}

// processWithRetry 应用层重试（指数退避，默认 3 次）；耗尽后记录死信并 Nack 入 DLQ。
// 从消息头提取上游 trace context，consumer span 接续 HTTP/MQ 全链路。
func (m *MQ) processWithRetry(ctx context.Context, name, queueName string, msg amqp.Delivery, handler func(body []byte) error) {
	traceCtx := tracing.Extract(context.Background(), msg.Headers)
	_, span := tracing.Tracer("mq").Start(traceCtx, "mq.consume "+queueName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", queueName),
			attribute.String("messaging.rabbitmq.routing_key", msg.RoutingKey),
		))
	defer span.End()

	var err error
	for attempt := 1; attempt <= m.maxRetries; attempt++ {
		if err = handler(msg.Body); err == nil {
			_ = msg.Ack(false)
			return
		}
		logger.L.Warn("消息处理失败，将重试",
			zap.String("consumer", name),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", m.maxRetries),
			zap.Error(err))
		if attempt < m.maxRetries {
			if !sleepOrStopCtx(ctx, time.Duration(attempt)*time.Second, m.stopCh) {
				return
			}
		}
	}

	// 重试耗尽：记录死信（应用层回调）+ Nack 不重回队列 → 进入 DLQ
	span.RecordError(err)
	span.SetStatus(codes.Error, "重试耗尽转入死信")
	logger.L.Error("消息重试耗尽，转入死信队列",
		zap.String("consumer", name),
		zap.String("queue", queueName),
		zap.String("routing_key", msg.RoutingKey),
		zap.Error(err))
	if OnDeadLetter != nil {
		OnDeadLetter(queueName, msg.Exchange, msg.RoutingKey, string(msg.Body), err.Error())
	}
	_ = msg.Nack(false, false)
}

// ---------------- 生命周期 ----------------

// QueueDepth 被动声明队列以查询积压消息数（未连接或失败返回 -1）
func (m *MQ) QueueDepth() int {
	conn := m.currentConn()
	if conn == nil {
		return -1
	}
	ch, err := conn.Channel()
	if err != nil {
		return -1
	}
	defer ch.Close()
	q, err := ch.QueueDeclarePassive(m.queue, true, false, false, false, nil)
	if err != nil {
		return -1
	}
	return q.Messages
}

// IsConnected 当前是否处于已连接状态（供健康检查使用）
func (m *MQ) IsConnected() bool {
	return m.currentConn() != nil
}

func (m *MQ) isClosing() bool {
	return atomic.LoadInt32(&m.closing) == 1
}

// Close 停止重连管理器并关闭连接（幂等）
func (m *MQ) Close() {
	if m == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&m.closing, 0, 1) {
		return
	}
	close(m.stopCh)
	<-m.done
}

// sleepOrStop 睡眠可被关闭信号打断；返回 false 表示应退出
func sleepOrStop(d time.Duration, stop <-chan struct{}) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-stop:
		return false
	}
}

// sleepOrStopCtx 同 sleepOrStop，同时响应 ctx 取消
func sleepOrStopCtx(ctx context.Context, d time.Duration, stop <-chan struct{}) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	case <-stop:
		return false
	}
}
