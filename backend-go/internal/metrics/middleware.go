package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GinMiddleware 采集每个 HTTP 请求的次数与耗时分布。
// 注意：path 使用路由模板（FullPath）而非原始 URL，避免 :vin 等参数打爆 label 基数。
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, strconv.Itoa(c.Writer.Status())).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
