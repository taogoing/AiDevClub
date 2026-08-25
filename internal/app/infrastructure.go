package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/platform"
)

type Infrastructure struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func OpenInfrastructure(cfg *platform.Config) (*Infrastructure, error) {
	db, err := platform.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	return &Infrastructure{
		DB:    db,
		Redis: platform.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB),
	}, nil
}

func (i *Infrastructure) Ping(ctx context.Context) error {
	sqlDB, err := i.DB.DB()
	if err != nil {
		return fmt.Errorf("get mysql client: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}
	if err := i.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func (i *Infrastructure) Close() error {
	var errs []error
	if i.Redis != nil {
		if err := i.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close redis: %w", err))
		}
	}
	if i.DB != nil {
		sqlDB, err := i.DB.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("get mysql client: %w", err))
		} else if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close mysql: %w", err))
		}
	}
	return errors.Join(errs...)
}
