package gorbac_redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheStore struct {
	rdb redis.UniversalClient
}

func NewRedisCacheStore(rdb redis.UniversalClient) *RedisCacheStore {
	return &RedisCacheStore{rdb: rdb}
}

func (s *RedisCacheStore) Get(ctx context.Context, key string) (value []byte, ok bool, err error) {
	value, err = s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (s *RedisCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

func (s *RedisCacheStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

