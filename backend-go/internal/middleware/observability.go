package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend-go/pkg/logger"
)

// RequestIDHeader 请求追踪 ID 的 HTTP 头名
const RequestIDHeader = "X-Request-ID"

// RequestID 请求追踪中间件：优先沿用调用方传入的 ID，否则生成新 ID。
// 贯穿访问日志与响应头，用于串联同一请求的全部日志。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// newRequestID 生成 16 字节随机十六进制 ID
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// AccessLog 结构化访问日志中间件：记录方法/路径/状态码/耗时/客户端/request-id
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.L.Error("HTTP 请求", fields...)
		case status >= 400:
			logger.L.Warn("HTTP 请求", fields...)
		default:
			logger.L.Info("HTTP 请求", fields...)
		}
	}
}
