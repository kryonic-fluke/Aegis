package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr string) (*redis.Client, error) {
	var opt *redis.Options
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		parsedOpt, err := redis.ParseURL(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis url: %w", err)
		}
		opt = parsedOpt
	} else {
		opt = &redis.Options{
			Addr: addr,
		}
	}

	opt.PoolSize = 100
	opt.MinIdleConns = 10
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
