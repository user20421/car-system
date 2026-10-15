package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/database"
	"backend-go/internal/mq"
	"backend-go/internal/service"
	"backend-go/internal/simulator"
)

var startedAt = time.Now()

// GetSystemMetrics 系统运行时指标（可观测性）
// @Summary 后端实时性能指标
// @Tags 系统
// @Security BearerAuth
// @Success 200 {object} map[string]any "goroutines/内存/连接池/消息计数/队列深度"
// @Router /api/v1/system/metrics [get]
func GetSystemMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// MySQL 连接池
	var dbStats gin.H
	if sqlDB, err := database.DB.DB(); err == nil {
		s := sqlDB.Stats()
		dbStats = gin.H{"open": s.OpenConnections, "in_use": s.InUse, "idle": s.Idle}
	}

	// RabbitMQ 队列深度
	queueReady := mq.Client.QueueDepth()

	published, persisted := service.Stats()
	simRunning, simParams := simulator.Global.Status()

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"goroutines":      runtime.NumGoroutine(),
		"heap_alloc_mb":   float64(m.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":     float64(m.HeapSys) / 1024 / 1024,
		"gc_count":        m.NumGC,
		"mysql":           dbStats,
		"mq_published":    published,
		"mq_persisted":    persisted,
		"mq_queue_ready":  queueReady,
		"uptime_seconds":  int(time.Since(startedAt).Seconds()),
		"simulator":       gin.H{"running": simRunning, "params": simParams},
	}})
}
