package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kryonic-fluke/aegis/internal/llm"
	"github.com/redis/go-redis/v9"
)

type canonicalRequest struct {
	Model       string        `json:"model"`
	Messages    []llm.Message `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type ExactCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewExactCache(client *redis.Client, ttl time.Duration) *ExactCache {
	return &ExactCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *ExactCache) GenerateKey(req *llm.ChatRequest) (string, error) {
	canonical := canonicalRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request for key generation: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("cache:exact:%x", hash), nil
}

func (c *ExactCache) Get(ctx context.Context, key string) (*llm.ChatResponse, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var resp llm.ChatResponse
	if err := json.Unmarshal([]byte(val), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached response: %w", err)
	}

	return &resp, nil
}

func (c *ExactCache) Set(ctx context.Context, key string, resp *llm.ChatResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response for cache set: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}
