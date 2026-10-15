package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"backend-go/internal/handler"
	"backend-go/internal/middleware"
)

// Setup 注册全部路由
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 公开接口
	r.GET("/health", handler.HealthCheck)

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		// 鉴权
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
		}

		// 受保护接口（需 JWT）
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth())
		{
			protected.GET("/profile", handler.Profile)

			// 车辆
			vehicle := protected.Group("/vehicle")
			{
				vehicle.POST("/telemetry", handler.ReportTelemetry)          // 车机上报
				vehicle.GET("/:vin/telemetry/latest", handler.GetLatestTelemetry) // 查询最新状态
				vehicle.GET("", middleware.RequireRole("admin"), handler.ListVehicles) // 管理员列表
			}
		}
	}

	// WebSocket 实时通道
	r.GET("/ws", handler.WebSocket)

	return r
}
