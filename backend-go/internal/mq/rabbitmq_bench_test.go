package mq_test

// Channel 池化性能对比基准测试（需要本地 RabbitMQ 运行，否则自动跳过）：
//
//	cd backend-go
//	go test ./internal/mq/ -bench . -benchtime 2s -run '^$'
//
// 对比"每消息新建 channel"（优化前实现）与"池化复用 channel"（优化后实现）的发布吞吐。

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"backend-go/internal/config"
	"backend-go/internal/mq"
	"backend-go/pkg/logger"
)

func benchURL() string {
	if u := os.Getenv("CAR_RABBITMQ_URL"); u != "" {
		return u
	}
	return "amqp://car-system:car123456@127.0.0.1:5672/"
}

var benchBody = []byte(`{"vin":"BENCHVIN000000001","speed":66.6,"rpm":3000,"battery":88.8}`)

// BenchmarkPublishPerMessageChannel 优化前：每发一条消息新建并关闭一个 channel
func BenchmarkPublishPerMessageChannel(b *testing.B) {
	conn, err := amqp.Dial(benchURL())
	if err != nil {
		b.Skipf("RabbitMQ 不可用，跳过基准测试: %v", err)
	}
	defer conn.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := conn.Channel()
		if err != nil {
			b.Fatal(err)
		}
		// 发到无绑定的 routing key：只测量发布路径，不污染遥测消费链路
		err = ch.PublishWithContext(context.Background(),
			"vehicle.data", "bench.noop",
			false, false,
			amqp.Publishing{DeliveryMode: amqp.Persistent, Body: benchBody, Timestamp: time.Now()})
		_ = ch.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPublishPooled 优化后：channel 池化复用
func BenchmarkPublishPooled(b *testing.B) {
	logger.Init("release")
	cfg := config.RabbitMQConfig{
		URL:               benchURL(),
		TelemetryExchange: "vehicle.data",
		TelemetryQueue:    "telemetry.persist",
	}
	if err := mq.Init(cfg); err != nil {
		b.Skipf("RabbitMQ 不可用，跳过基准测试: %v", err)
	}
	defer mq.Client.Close()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 同样发到无绑定的 routing key，与未池化基准保持口径一致
		if err := mq.Client.PublishRaw(ctx, "vehicle.data", "bench.noop", benchBody); err != nil {
			b.Fatal(err)
		}
	}
}
