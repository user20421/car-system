package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"backend-go/internal/cache"
	"backend-go/internal/config"
	"backend-go/internal/database"
	"backend-go/internal/metrics"
	"backend-go/internal/middleware"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/router"
	"backend-go/internal/service"
	"backend-go/internal/simulator"
	"backend-go/internal/tracing"
	"backend-go/internal/ws"
	"backend-go/pkg/logger"

	"go.uber.org/zap"

	_ "backend-go/docs" // swagger 生成的文档
)

// @title Car System 车联网后端 API
// @version 1.0
// @description 车联网系统后端接口文档：鉴权 / 车辆遥测 / 实时推送
// @contact.name user20421
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 格式: Bearer <token>
func main() {
	// 1. 加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 2. 初始化日志
	logger.Init(cfg.Server.Mode)
	defer logger.Sync()
	logger.L.Info("car-system 后端启动中", zap.Int("port", cfg.Server.Port))

	// 2.5 初始化链路追踪（Jaeger 未启动时静默重试，不阻塞服务）
	traceShutdown, err := tracing.Init(context.Background(), "car-server", cfg.Telemetry.JaegerEndpoint)
	if err != nil {
		logger.L.Warn("链路追踪初始化失败，将继续运行（无追踪）", zap.Error(err))
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = traceShutdown(ctx)
		}()
		logger.L.Info("链路追踪已启用", zap.String("jaeger", cfg.Telemetry.JaegerEndpoint))
	}

	// 3. 初始化 JWT
	middleware.InitJWT(cfg.JWT.Secret, cfg.JWT.ExpireDuration())

	// 4. 初始化 MySQL（自动建库 + 迁移）
	if err := database.Init(cfg.MySQL, cfg.Server.Mode); err != nil {
		logger.L.Fatal("MySQL 初始化失败", zap.Error(err))
	}
	logger.L.Info("MySQL 就绪")

	// 5. 初始化 Redis
	if err := cache.Init(cfg.Redis); err != nil {
		logger.L.Fatal("Redis 初始化失败", zap.Error(err))
	}
	logger.L.Info("Redis 就绪")

	// 6. 初始化 RabbitMQ
	if err := mq.Init(cfg.RabbitMQ); err != nil {
		logger.L.Fatal("RabbitMQ 初始化失败", zap.Error(err))
	}
	logger.L.Info("RabbitMQ 就绪")

	// 7. 种子数据：默认管理员 + 示例车辆
	seedData()
	service.SeedAlarmRules()

	// 8. 启动遥测消费者（异步落库，断线自动恢复）
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	go func() {
		if err := mq.Client.ConsumeTelemetry(consumerCtx, service.PersistTelemetry); err != nil {
			logger.L.Error("遥测消费者异常退出", zap.Error(err))
		}
	}()

	// 8.1 启动控车指令分发消费者（MQ → WebSocket 推送给车机）
	go func() {
		if err := mq.Client.ConsumeControl(consumerCtx, "control.dispatch", nil, "control-dispatcher", service.DispatchCommand); err != nil {
			logger.L.Error("控车分发消费者异常退出", zap.Error(err))
		}
	}()

	// 8.2 死信记录：消息重试耗尽后落 dead_letters 表（管理端可查/重投）
	mq.OnDeadLetter = func(queue, exchange, routingKey, body, reason string) {
		metrics.DeadLettersTotal.Inc()
		if err := database.DB.Create(&model.DeadLetter{
			Queue: queue, Exchange: exchange, RoutingKey: routingKey,
			Body: body, Reason: reason,
		}).Error; err != nil {
			logger.L.Error("死信落表失败", zap.String("queue", queue), zap.Error(err))
		}
	}

	// 8.3 启动 Outbox 投递器（控车指令 → MQ，断连恢复后自动补投）
	go service.StartOutboxRelay(consumerCtx)

	// 9. 启动 WebSocket Hub
	go ws.GlobalHub.Run()

	// 10. 启动 HTTP 服务（优雅启停）
	r := router.Setup(cfg.Server.Mode)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
		// 超时防护：慢客户端不能无限占用连接（WS 升级后连接被 hijack，不受此限制）
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.L.Info("HTTP 服务已启动", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("HTTP 服务异常", zap.Error(err))
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.L.Info("收到退出信号，开始优雅关闭...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L.Error("HTTP 优雅关闭失败", zap.Error(err))
	}

	// 统一收尾：模拟器 → 消费者/Outbox → WS Hub → MQ → Redis → MySQL
	simulator.Global.Stop()
	consumerCancel()
	ws.GlobalHub.Stop()
	mq.Client.Close()
	cache.Close()
	database.Close()
	logger.L.Info("服务已安全退出")
}

// persistTelemetry 已迁移至 service.PersistTelemetry（供集成测试复用）


// seedData 初始化管理员、车主账号与示例车辆
func seedData() {
	var admin model.User
	if err := database.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		admin = model.User{Username: "admin", PasswordHash: string(hash), Role: "admin"}
		database.DB.Create(&admin)
		logger.L.Info("已创建默认管理员 admin")
	}
	// 默认车主（车机端与车主 App 的登录账号）
	var driver model.User
	if err := database.DB.Where("username = ?", "driver1").First(&driver).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("driver123456"), bcrypt.DefaultCost)
		driver = model.User{Username: "driver1", PasswordHash: string(hash), Role: "owner"}
		database.DB.Create(&driver)
		logger.L.Info("已创建默认车主 driver1")
	}
	// 示例车辆归车主所有（归属校验要求：车机以 driver1 身份控车/回执）
	var v model.Vehicle
	if err := database.DB.Where("vin = ?", "LVSFACBE8MF012345").First(&v).Error; err != nil {
		database.DB.Create(&model.Vehicle{
			VIN: "LVSFACBE8MF012345", Model: "Car-System Demo EV",
			OwnerID: driver.ID, Status: "online",
		})
		logger.L.Info("已创建示例车辆 LVSFACBE8MF012345")
	} else if v.OwnerID != driver.ID {
		database.DB.Model(&v).Update("owner_id", driver.ID)
		logger.L.Info("示例车辆归属已迁移至 driver1（归属校验适配）")
	}
}
