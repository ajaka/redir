package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/configs"
	fts "github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Sredis struct {
	rdb *redis.Client
}

func InitializeRedis(ctx context.Context, cfg *configs.EnvData, logger *slog.Logger) *Sredis {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.REDIS_ADDR,
		Password: cfg.REDIS_PASSWORD,
		DB:       0,
		Protocol: 2,
	})
	err := rdb.Set(ctx, "ping", "pong", 0).Err()
	if err != nil {
		panic(err)
	}
	logger.Info("Redis cache connected successfully")
	return &Sredis{
		rdb,
	}
}

func (r *Sredis) CheckHealth(ctx context.Context, logger *slog.Logger) error {
	logger.Info("redis: checking readiness")

	if r == nil || r.rdb == nil {
		return fmt.Errorf("redis: client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := r.rdb.Ping(ctx).Err(); err != nil {
		logger.Error("redis: readiness check failed", "error", err.Error())
		return fmt.Errorf("redis: %w", err)
	}

	logger.Info("redis: readiness check passed")
	return nil
}

func structToInterface(s any) map[string]any {
	t := fts.New(s)
	t.TagName = "redis"
	m := t.Map()
	for k, v := range m {
		if u, ok := v.(uuid.UUID); ok {
			m[k] = u.String()
		}
	}
	return m
}
