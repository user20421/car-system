package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"backend-go/internal/cache"
	"backend-go/internal/database"
	"backend-go/internal/metrics"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/ws"
	"backend-go/pkg/logger"
)

// TelemetryInput 统一的车况输入（HTTP 上报与虚拟车队共用）
type TelemetryInput struct {
	VIN       string  `json:"vin"`
	MessageID string  `json:"message_id"` // 消息唯一标识：生产者生成，消费端幂等去重
	Speed     float64 `json:"speed"`
	RPM       int     `json:"rpm"`
	Battery   float64 `json:"battery"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	FaultCode string  `json:"fault_code"`
}

var publishedTotal int64 // MQ 发布总数
var persistedTotal int64 // MQ 消费落库总数
var ingestSeq int64      // message_id 生成序号

// nextMessageID 生成消息唯一标识：VIN + 纳秒时间戳 + 进程内递增序号
func nextMessageID(vin string) string {
	return fmt.Sprintf("%s-%d-%d", vin, time.Now().UnixNano(), atomic.AddInt64(&ingestSeq, 1))
}

// Ingest 车况数据统一入口：校验车辆 → Redis 实时缓存 → RabbitMQ 异步落库 → WebSocket 广播 → 告警评估
func Ingest(in TelemetryInput) error {
	// 校验车辆存在
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", in.VIN).First(&vehicle).Error; err != nil {
		return err
	}

	// 生成消息唯一标识（幂等键），随后随消息体一路流转到落库
	if in.MessageID == "" {
		in.MessageID = nextMessageID(in.VIN)
	}

	// 告警引擎：规则阈值判定 + 围栏进出判定（内存缓存规则，开销极小）
	EvaluateAlarms(in, vehicle)

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
	metrics.MQPublishedTotal.Inc()
	return nil
}

// MarkPersisted MQ 消费者落库成功后调用
func MarkPersisted() {
	atomic.AddInt64(&persistedTotal, 1)
	metrics.MQPersistedTotal.Inc()
}

// Stats 返回 (发布总数, 落库总数)
func Stats() (int64, int64) {
	return atomic.LoadInt64(&publishedTotal), atomic.LoadInt64(&persistedTotal)
}

// PersistTelemetry MQ 消费者回调：将车况数据落库
// 幂等：message_id 唯一约束 + 冲突忽略，同一消息重复投递只落库一次
func PersistTelemetry(body []byte) error {
	var req TelemetryInput
	if err := json.Unmarshal(body, &req); err != nil {
		return err
	}
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", req.VIN).First(&vehicle).Error; err != nil {
		return err
	}
	t := model.Telemetry{
		VehicleID: vehicle.ID, MessageID: req.MessageID, Speed: req.Speed, RPM: req.RPM,
		Battery: req.Battery, Lat: req.Lat, Lng: req.Lng, FaultCode: req.FaultCode,
	}
	res := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&t)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		logger.L.Debug("重复消息，幂等忽略", zap.String("message_id", req.MessageID))
		return nil
	}
	MarkPersisted()
	return nil
}
