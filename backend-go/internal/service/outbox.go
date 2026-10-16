package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/tracing"
	"backend-go/pkg/logger"
)

// Outbox relay：扫描 outbox 表中未投递的消息并发布到 MQ。
// 触发方式：业务写入后 NotifyOutbox 立即触发 + 定时兜底轮询。
// MQ 断连期间消息保持 pending，恢复后自动补投，保证指令零丢失。

const (
	outboxPollInterval = 2 * time.Second // 兜底轮询周期
	outboxBatchSize    = 100             // 单轮最大投递条数
	outboxMaxRetries   = 20              // 超过则标记 failed（等待人工介入）
	outboxRetention    = 7 * 24 * time.Hour // 已投递记录保留时长
)

var outboxNotify = make(chan struct{}, 1)

// silentSession 静默 GORM session：用于高频轮询查询，避免 debug 模式刷 SQL 日志
var silentSession = &gorm.Session{Logger: gormlogger.Discard}

// NotifyOutbox 通知 relay 立即投递（非阻塞，已有通知则合并）
func NotifyOutbox() {
	select {
	case outboxNotify <- struct{}{}:
	default:
	}
}

// StartOutboxRelay 启动发件箱投递器，随 ctx 取消而退出
func StartOutboxRelay(ctx context.Context) {
	logger.L.Info("Outbox 投递器已启动", zap.Duration("poll_interval", outboxPollInterval))
	tick := time.NewTicker(outboxPollInterval)
	cleanupTick := time.NewTicker(time.Hour)
	defer tick.Stop()
	defer cleanupTick.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.L.Info("Outbox 投递器已停止")
			return
		case <-outboxNotify:
		case <-tick.C:
		case <-cleanupTick.C:
			cleanupOutbox()
			continue
		}
		drainOutbox()
	}
}

// cleanupOutbox 定期清理已投递的历史记录，防止表无限膨胀（failed 保留供人工排查）
func cleanupOutbox() {
	res := database.DB.Where("status = ? AND created_at < ?", "sent", time.Now().Add(-outboxRetention)).
		Delete(&model.OutboxMessage{})
	if res.Error != nil {
		logger.L.Warn("Outbox 清理失败", zap.Error(res.Error))
		return
	}
	if res.RowsAffected > 0 {
		logger.L.Info("Outbox 历史记录已清理", zap.Int64("deleted", res.RowsAffected))
	}
}

// drainOutbox 批量投递 pending 消息：成功标记 sent，失败累加重试次数
func drainOutbox() {
	var msgs []model.OutboxMessage
	// 轮询是高频空查询，用静默 session 避免刷 GORM SQL 日志
	if err := database.DB.Session(silentSession).Where("status = ?", "pending").
		Order("id").Limit(outboxBatchSize).Find(&msgs).Error; err != nil {
		logger.L.Warn("Outbox 扫描失败", zap.Error(err))
		return
	}
	for i := range msgs {
		msg := &msgs[i]
		// 从持久化的 traceparent 恢复链路，让“HTTP 下单 → Outbox 投递 → 消费分发”连成一条 trace
		pubCtx, cancel := context.WithTimeout(tracing.ContextWithTraceParent(context.Background(), msg.TraceParent), 3*time.Second)
		err := mq.Client.PublishRaw(pubCtx, msg.Exchange, msg.RoutingKey, []byte(msg.Body))
		cancel()

		if err != nil {
			msg.Retries++
			if msg.Retries >= outboxMaxRetries {
				msg.Status = "failed"
				logger.L.Error("Outbox 消息投递多次失败，标记 failed 等待人工介入",
					zap.Uint("id", msg.ID), zap.Int("retries", msg.Retries))
			}
			if dbErr := database.DB.Save(msg).Error; dbErr != nil {
				logger.L.Warn("Outbox 状态更新失败", zap.Uint("id", msg.ID), zap.Error(dbErr))
			}
			// MQ 不可用时后续消息大概率同样失败，留待下轮重试
			if err == mq.ErrNotConnected {
				return
			}
			continue
		}

		now := time.Now()
		msg.Status = "sent"
		msg.SentAt = &now
		if err := database.DB.Save(msg).Error; err != nil {
			logger.L.Warn("Outbox 状态更新失败", zap.Uint("id", msg.ID), zap.Error(err))
		}
	}
}
