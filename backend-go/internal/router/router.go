package router

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"backend-go/internal/handler"
	"backend-go/internal/metrics"
	"backend-go/internal/middleware"
)

// Setup 注册全部路由
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()
	// 中间件链：request-id → 访问日志 → panic 恢复 → 链路追踪 → 指标采集
	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(),
		gin.Recovery(),
		otelgin.Middleware("car-server"),
		metrics.GinMiddleware(),
	)

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
	r.GET("/metrics", gin.WrapH(promhttp.Handler())) // Prometheus 抓取端点

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

			// 车辆（带 :vin 的接口均做归属校验：车主只能操作自己的车，admin 不受限）
			own := middleware.RequireVehicleOwnership()
			vehicle := protected.Group("/vehicle")
			{
				vehicle.POST("/telemetry", handler.ReportTelemetry)          // 车机上报（设备通道，仅 JWT）
				vehicle.GET("/:vin/telemetry/latest", own, handler.GetLatestTelemetry) // 查询最新状态
				vehicle.GET("", middleware.RequireRole("admin"), handler.ListVehicles) // 管理员列表
				vehicle.POST("/:vin/command", own, handler.SendVehicleCommand)     // 远程控车
				vehicle.POST("/:vin/command/:id/ack", own, handler.AckVehicleCommand) // 车机执行回执
				vehicle.GET("/:vin/commands", own, handler.ListVehicleCommands)   // 指令历史
				vehicle.GET("/:vin/telemetry/history", own, handler.GetTelemetryHistory) // 车况历史曲线
				vehicle.GET("/:vin", own, handler.GetVehicle)                     // 车辆详情
			}

			// 管理员专属：模拟器控制 + 系统指标
			admin := protected.Group("")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.POST("/simulator/start", handler.StartSimulator)
				admin.POST("/simulator/stop", handler.StopSimulator)
				admin.GET("/simulator/status", handler.SimulatorStatus)
				admin.GET("/system/metrics", handler.GetSystemMetrics)
				admin.GET("/system/dead-letters", handler.ListDeadLetters)              // 死信列表
				admin.POST("/system/dead-letters/:id/requeue", handler.RequeueDeadLetter) // 死信人工重投
				admin.GET("/system/outbox", handler.ListOutbox)                          // Outbox 发件箱（排查 pending/failed）

				// 告警引擎：规则 CRUD + 告警记录 + 电子围栏
				admin.GET("/alarm/rules", handler.ListAlarmRules)
				admin.POST("/alarm/rules", handler.CreateAlarmRule)
				admin.PUT("/alarm/rules/:id", handler.UpdateAlarmRule)
				admin.DELETE("/alarm/rules/:id", handler.DeleteAlarmRule)
				admin.GET("/alarm/records", handler.ListAlarmRecords)
				admin.POST("/alarm/records/:id/resolve", handler.ResolveAlarmRecord)
				admin.GET("/fences", handler.ListFences)
				admin.POST("/fences", handler.CreateFence)
				admin.DELETE("/fences/:id", handler.DeleteFence)

				// 管理员专属：车辆档案管理 + 用户管理
				admin.POST("/vehicle", handler.CreateVehicle)         // 登记车辆
				admin.PUT("/vehicle/:vin", handler.UpdateVehicle)     // 更新车辆档案
				admin.DELETE("/vehicle/:vin", handler.DeleteVehicle)  // 删除车辆
				admin.GET("/users", handler.ListUsers)                // 用户列表
				admin.POST("/users", handler.CreateUser)              // 创建用户
			}
		}
	}

	// WebSocket 实时通道
	r.GET("/ws", handler.WebSocket)

	return r
}
