// Package tracing OpenTelemetry 链路追踪初始化与 MQ 消息传播的 trace context 注入/提取
package tracing

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Init 初始化全局 TracerProvider（OTLP gRPC → Jaeger）。
// Jaeger 未启动时 exporter 静默重试，不影响服务启动。
// 返回 shutdown 函数供优雅退出时调用。
func Init(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP exporter 失败: %w", err)
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("创建 resource 失败: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 本地演示全量采样
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}

// Tracer 获取指定名称的 Tracer
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// amqpHeaderCarrier 让 AMQP 消息头实现 OpenTelemetry TextMapCarrier，跨进程传递 trace context
type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpHeaderCarrier) Set(key, val string) { c[key] = val }

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// Inject 将当前 ctx 的 trace context 注入 AMQP 消息头
func Inject(ctx context.Context, headers amqp.Table) {
	if headers == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, amqpHeaderCarrier(headers))
}

// Extract 从 AMQP 消息头提取 trace context
func Extract(ctx context.Context, headers amqp.Table) context.Context {
	if headers == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, amqpHeaderCarrier(headers))
}

// TraceParentFrom 提取当前 ctx 的 traceparent 字符串（用于 Outbox 行持久化）
func TraceParentFrom(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ContextWithTraceParent 从持久化的 traceparent 恢复 trace context（relay 投递时续链）
func ContextWithTraceParent(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": traceparent})
}
