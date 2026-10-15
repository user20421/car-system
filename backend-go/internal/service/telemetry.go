package service

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"backend-go/internal/cache"
	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/ws"
)

// TelemetryInput 统一的车况输入（HTTP 上报与虚拟车队共用）
type TelemetryInput struct {
	VIN       string  `json:"vin"`
	Speed     float64 `json:"speed"`
	RPM       int     `json:"rpm"`
	Battery   float64 `json:"battery"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	FaultCode string  `json:"fault_code"`
}

var publishedTotal int64 // MQ 发布总数
var persistedTotal int64 // MQ 消费落库总数

// Ingest 车况数据统一入口：校验车辆 → Redis 实时缓存 → RabbitMQ 异步落库 → WebSocket 广播
func Ingest(in TelemetryInput) error {
	// 校验车辆存在
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", in.VIN).First(&vehicle).Error; err != nil {
		return err
	}

	body, err := json.Marshal(in)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Redis 实时状态（TTL 5 分钟）
	if err := cache.RDB.Set(ctx, "vehicle:latest:"+in.VIN, body, 5*time.Minute).Err(); err != nil {
		return err
	}
	// 2. RabbitMQ 异步落库
	if err := mq.Client.PublishTelemetry(ctx, in.VIN, body); err != nil {
		return err
	}
	// 3. WebSocket 广播
	ws.GlobalHub.Broadcast(body)

	atomic.AddInt64(&publishedTotal, 1)
	return nil
}

// MarkPersisted MQ 消费者落库成功后调用
func MarkPersisted() { atomic.AddInt64(&persistedTotal, 1) }

// Stats 返回 (发布总数, 落库总数)
func Stats() (int64, int64) {
	return atomic.LoadInt64(&publishedTotal), atomic.LoadInt64(&persistedTotal)
}
