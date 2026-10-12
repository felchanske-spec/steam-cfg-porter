package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"steam-cfg-portet/internal/config_service/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) domain.RedisRepo {
	return &RedisCache{
		client: client,
	}
}

func (r *RedisCache) GetConfig(ctx context.Context, key string) (*domain.Config, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrCacheMiss
		}
		return nil, fmt.Errorf("redis Get failed: %v", err)
	}

	if value == "null" {
		return nil, nil
	}

	cfg := &domain.Config{}
	if err := json.Unmarshal([]byte(value), cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from cache: %v", err)
	}

	return cfg, nil
}

func (r *RedisCache) SetConfig(ctx context.Context, key string, cfg *domain.Config, ttl time.Duration) error {
	valueBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config for cache: %v", err)
	}

	err = r.client.Set(ctx, key, valueBytes, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis Set failed: %v", err)
	}
	return nil
}

func (r *RedisCache) DeleteConfig(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis Del failed: %v", err)
	}
	return nil
}
