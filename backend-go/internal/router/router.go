package router

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
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

	// CORS：本地开发允许所有 localhost 来源（Vue dev server / Flutter web 端口不固定）
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
			protected.GET("/my/vehicles", handler.MyVehicles) // 车主：我的车辆

			// 车辆
			vehicle := protected.Group("/vehicle")
			{
				vehicle.POST("/telemetry", handler.ReportTelemetry)          // 车机上报
				vehicle.GET("/:vin/telemetry/latest", handler.GetLatestTelemetry) // 查询最新状态
				vehicle.GET("", middleware.RequireRole("admin"), handler.ListVehicles) // 管理员列表
				vehicle.POST("/:vin/command", handler.SendVehicleCommand)     // 远程控车
				vehicle.POST("/:vin/command/:id/ack", handler.AckVehicleCommand) // 车机执行回执
				vehicle.GET("/:vin/commands", handler.ListVehicleCommands)   // 指令历史
				vehicle.GET("/:vin/telemetry/history", handler.GetTelemetryHistory) // 车况历史曲线
				vehicle.GET("/:vin", handler.GetVehicle)                     // 车辆详情
			}

			// 管理员专属：模拟器控制 + 系统指标
			admin := protected.Group("")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.POST("/simulator/start", handler.StartSimulator)
				admin.POST("/simulator/stop", handler.StopSimulator)
				admin.GET("/simulator/status", handler.SimulatorStatus)
				admin.GET("/system/metrics", handler.GetSystemMetrics)
			}
		}
	}

	// WebSocket 实时通道
	r.GET("/ws", handler.WebSocket)

	return r
}
