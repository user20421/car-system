package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend-go/internal/config"
	"backend-go/internal/model"
)

var DB *gorm.DB

// Init 连接 MySQL（自动建库）、配置连接池并自动迁移表结构
func Init(cfg config.MySQLConfig, mode string) error {
	// 1. 先不带库名连接，确保数据库存在
	bootstrap, err := gorm.Open(mysql.Open(cfg.DSNWithoutDB()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	createSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DBName)
	if err := bootstrap.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	// 2. 正式连接业务库
	gormCfg := &gorm.Config{}
	if mode == "debug" {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	}
	DB, err = gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return fmt.Errorf("连接业务库失败: %w", err)
	}

	// 3. 连接池调优
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMinutes) * time.Minute)

	// 4. 自动迁移表结构
	if err := DB.AutoMigrate(&model.User{}, &model.Vehicle{}, &model.Telemetry{}); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}
	return nil
}
