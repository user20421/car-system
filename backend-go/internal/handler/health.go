package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/cache"
	"backend-go/internal/database"
	"backend-go/internal/mq"
)

// HealthCheck 健康检查：返回各中间件连通状态
// @Summary 健康检查
// @Tags 系统
// @Produce json
// @Success 200 {object} map[string]any "全部组件正常"
// @Failure 503 {object} map[string]any "有组件异常"
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	result := gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"components": gin.H{
			"mysql":    "up",
			"redis":    "up",
			"rabbitmq": "up",
		},
	}
	comps := result["components"].(gin.H)

	// 检查 MySQL
	if sqlDB, err := database.DB.DB(); err != nil || sqlDB.Ping() != nil {
		comps["mysql"] = "down"
		result["status"] = "degraded"
	}
	// 检查 Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := cache.RDB.Ping(ctx).Err(); err != nil {
		comps["redis"] = "down"
		result["status"] = "degraded"
	}
	// 检查 RabbitMQ（真实连接状态，重连中视为 down）
	if mq.Client == nil || !mq.Client.IsConnected() {
		comps["rabbitmq"] = "down"
		result["status"] = "degraded"
	}

	code := http.StatusOK
	if result["status"] == "degraded" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, result)
}
