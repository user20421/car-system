package main

import (
	"context"
	"encoding/json"
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
	"backend-go/internal/middleware"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/router"
	"backend-go/internal/service"
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

	// 8. 启动遥测消费者（异步落库）
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	go func() {
		if err := mq.Client.ConsumeTelemetry(consumerCtx, persistTelemetry); err != nil {
			logger.L.Error("遥测消费者异常退出", zap.Error(err))
		}
	}()

	// 8.1 启动控车指令分发消费者（MQ → WebSocket 推送给车机）
	go func() {
		if err := mq.Client.ConsumeControl(consumerCtx, "control.dispatch", nil, "control-dispatcher", service.DispatchCommand); err != nil {
			logger.L.Error("控车分发消费者异常退出", zap.Error(err))
		}
	}()

	// 9. 启动 WebSocket Hub
	go ws.GlobalHub.Run()

	// 10. 启动 HTTP 服务（优雅启停）
	r := router.Setup(cfg.Server.Mode)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
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
	consumerCancel()
	mq.Client.Close()
	cache.Close()
	logger.L.Info("服务已安全退出")
}

// persistTelemetry MQ 消费者回调：将车况数据落库（带幂等）
func persistTelemetry(body []byte) error {
	var req struct {
		VIN       string  `json:"vin"`
		Speed     float64 `json:"speed"`
		RPM       int     `json:"rpm"`
		Battery   float64 `json:"battery"`
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
		FaultCode string  `json:"fault_code"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return err
	}
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", req.VIN).First(&vehicle).Error; err != nil {
		return err
	}
	t := model.Telemetry{
		VehicleID: vehicle.ID, Speed: req.Speed, RPM: req.RPM,
		Battery: req.Battery, Lat: req.Lat, Lng: req.Lng, FaultCode: req.FaultCode,
	}
	if err := database.DB.Create(&t).Error; err != nil {
		return err
	}
	service.MarkPersisted()
	return nil
}

// seedData 初始化管理员账号与示例车辆
func seedData() {
	var admin model.User
	if err := database.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		admin = model.User{Username: "admin", PasswordHash: string(hash), Role: "admin"}
		database.DB.Create(&admin)
		logger.L.Info("已创建默认管理员 admin")
	}
	var v model.Vehicle
	if err := database.DB.Where("vin = ?", "LVSFACBE8MF012345").First(&v).Error; err != nil {
		database.DB.Create(&model.Vehicle{
			VIN: "LVSFACBE8MF012345", Model: "Car-System Demo EV",
			OwnerID: admin.ID, Status: "online",
		})
		logger.L.Info("已创建示例车辆 LVSFACBE8MF012345")
	}
}
