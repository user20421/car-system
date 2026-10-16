// Package metrics Prometheus 指标注册与采集：HTTP 请求、MQ 吞吐、WS 连接数、队列积压
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/ws"
)

var (
	// HTTPRequestsTotal HTTP 请求总数
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "car", Subsystem: "http", Name: "requests_total",
		Help: "HTTP 请求总数",
	}, []string{"method", "path", "status"})

	// HTTPRequestDuration HTTP 请求耗时分布
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "car", Subsystem: "http", Name: "request_duration_seconds",
		Help:    "HTTP 请求耗时分布",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	}, []string{"method", "path"})

	// MQPublishedTotal MQ 发布总数
	MQPublishedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "car", Subsystem: "mq", Name: "published_total",
		Help: "RabbitMQ 发布消息总数",
	})

	// MQPersistedTotal 遥测落库总数
	MQPersistedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "car", Subsystem: "mq", Name: "persisted_total",
		Help: "遥测消息消费落库总数",
	})

	// DeadLettersTotal 死信总数
	DeadLettersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "car", Subsystem: "mq", Name: "dead_letters_total",
		Help: "消费重试耗尽转入死信的消息总数",
	})

	// WSClients WS 在线连接数（抓取时实时读取）
	WSClients = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "car", Subsystem: "ws", Name: "clients",
		Help: "WebSocket 在线连接数",
	}, func() float64 {
		return float64(ws.GlobalHub.ClientCount())
	})

	// MQQueueDepth 遥测落库队列积压（抓取时实时查询）
	MQQueueDepth = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "car", Subsystem: "mq", Name: "queue_depth",
		Help: "遥测落库队列积压消息数（-1 表示未连接）",
	}, func() float64 {
		if mq.Client == nil {
			return -1
		}
		return float64(mq.Client.QueueDepth())
	})

	// OutboxPending Outbox 待投递消息数（抓取时实时查询）
	OutboxPending = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "car", Subsystem: "outbox", Name: "pending",
		Help: "Outbox 发件箱待投递消息数",
	}, func() float64 {
		if database.DB == nil {
			return 0
		}
		var count int64
		if err := database.DB.Model(&model.OutboxMessage{}).
			Where("status = ?", "pending").Count(&count).Error; err != nil {
			return 0
		}
		return float64(count)
	})
)
