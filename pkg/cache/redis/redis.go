package cacheredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func New(addr string, pass string, db int) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
			DB:       db,
		}),
	}
}

func blacklistKey(sessionID int) string {
	return fmt.Sprintf("blacklist:%v", sessionID)
}

func (r *RedisCache) Set(ctx context.Context, key string, val any, exp time.Duration) error {
	return r.client.Set(ctx, key, val, exp).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) BlacklistSession(ctx context.Context, sessionID int, duration time.Duration) error {
	return r.client.Set(ctx, blacklistKey(sessionID), true, duration).Err()
}

func (r *RedisCache) IsBlacklisted(ctx context.Context, sessionID int) (bool, error) {
	n, err := r.client.Exists(ctx, blacklistKey(sessionID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
