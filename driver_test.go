package gorbac_redis_test

import (
	"context"
	"github.com/redis/go-redis/v9"
	"testing"
)

func TestRedis(t *testing.T) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{"43.139.223.7:8088"},
		Password: "940430Dex",
	})
	//rbac := gorbac_redis.NewRedisRbac(rdb, "test")
	ctx := context.Background()
	rdb.SAdd(ctx, "aaa", "2324", "32132")
}
