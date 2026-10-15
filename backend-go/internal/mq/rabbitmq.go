package mq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"backend-go/internal/config"
	"backend-go/pkg/logger"
)

// MQ 封装 RabbitMQ 连接，提供发布/消费能力
type MQ struct {
	conn     *amqp.Connection
	exchange string
	queue    string
}

var Client *MQ

// Init 建立连接并声明交换机与队列
func Init(cfg config.RabbitMQConfig) error {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	Client = &MQ{conn: conn, exchange: cfg.TelemetryExchange, queue: cfg.TelemetryQueue}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("创建 channel 失败: %w", err)
	}
	defer ch.Close()

	// topic 交换机：按 routing key 灵活路由
	if err := ch.ExchangeDeclare(cfg.TelemetryExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明交换机失败: %w", err)
	}
	// 遥测落库队列：持久化，绑定 vehicle.telemetry.*
	if _, err := ch.QueueDeclare(cfg.TelemetryQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明队列失败: %w", err)
	}
	if err := ch.QueueBind(cfg.TelemetryQueue, "vehicle.telemetry.#", cfg.TelemetryExchange, false, nil); err != nil {
		return fmt.Errorf("绑定队列失败: %w", err)
	}
	return nil
}

// PublishTelemetry 发布车况数据（routing key: vehicle.telemetry.<vin>）
func (m *MQ) PublishTelemetry(ctx context.Context, vin string, body []byte) error {
	ch, err := m.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return ch.PublishWithContext(ctx,
		m.exchange,
		"vehicle.telemetry."+vin,
		false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		})
}

// ConsumeTelemetry 启动消费者（带手动 ACK 与消息幂等）
func (m *MQ) ConsumeTelemetry(ctx context.Context, handler func(body []byte) error) error {
	ch, err := m.conn.Channel()
	if err != nil {
		return err
	}
	// 消费者 QoS：一次最多预取 10 条，防止压垮消费者
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	msgs, err := ch.Consume(m.queue, "telemetry-consumer", false, false, false, false, nil)
	if err != nil {
		return err
	}
	logger.L.Info("RabbitMQ 遥测消费者已启动", zap.String("queue", m.queue))
	for {
		select {
		case <-ctx.Done():
			_ = ch.Close()
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消息通道被关闭")
			}
			if err := handler(msg.Body); err != nil {
				// 处理失败：拒绝并重新入队（本地开发场景简化处理）
				logger.L.Warn("消息处理失败，重新入队", zap.Error(err))
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

// Close 关闭连接
func (m *MQ) Close() {
	if m.conn != nil {
		_ = m.conn.Close()
	}
}
