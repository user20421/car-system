package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"backend-go/internal/config"
)

var RDB *redis.Client

// Init 连接 Redis 并做连通性检查
func Init(cfg config.RedisConfig) error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RDB.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return nil
}

// Close 关闭连接
func Close() {
	if RDB != nil {
		_ = RDB.Close()
	}
}
