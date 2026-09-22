package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const luaTokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last_updated')
local tokens = tonumber(data[1])
local last_updated = tonumber(data[2])

if tokens == nil then
    tokens = capacity
    last_updated = now
else
    local elapsed = math.max(0, now - last_updated)
    tokens = math.min(capacity, tokens + (elapsed * refill_rate))
    last_updated = now
end

if tokens >= requested then
    tokens = tokens - requested
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    redis.call('EXPIRE', key, math.ceil(capacity / refill_rate) * 2)
    return {1, math.floor(tokens)}
else
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    redis.call('EXPIRE', key, math.ceil(capacity / refill_rate) * 2)
    return {0, math.floor(tokens)}
end
`

type Result struct {
	Allowed   bool
	Remaining int
}

type TokenBucketLimiter struct {
	client     *redis.Client
	capacity   int
	refillRate float64
	script     *redis.Script
}

func NewTokenBucketLimiter(client *redis.Client, capacity int, refillRate float64) *TokenBucketLimiter {
	script := redis.NewScript(luaTokenBucketScript)

	return &TokenBucketLimiter{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		script:     script,
	}
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	now := time.Now().Unix()
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	res, err := l.script.Run(ctx, l.client, []string{redisKey}, l.capacity, l.refillRate, now, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	values, ok := res.([]interface{})
	if !ok || len(values) < 2 {
		return nil, fmt.Errorf("unexpected script response format: %v", res)
	}

	allowed, _ := values[0].(int64)
	remaining, _ := values[1].(int64)

	return &Result{
		Allowed:   allowed == 1,
		Remaining: int(remaining),
	}, nil
}
