package gorbac_redis_test

import (
	"encoding/json"
	"fmt"
	gorbac_redis "github.com/kordar/gorbac-redis"
	"github.com/redis/go-redis/v9"
	"testing"
)

func TestRedis(t *testing.T) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{"43.139.223.7:8088"},
		Password: "940430Dex",
	})
	rbac := gorbac_redis.NewRedisRbac(rdb, "test")
	//ctx := context.Background()
	//rdb.SAdd(ctx, "aaa", "2324", "32132")
	//item := db.AuthItem{
	//	Name:        "ccc",
	//	Type:        0,
	//	Description: "",
	//	AuthRules:   db.AuthRule{},
	//	RuleName:    "",
	//	ExecuteName: "",
	//	CreateTime:  time.Now(),
	//	UpdateTime:  time.Now(),
	//}
	//rbac.AddItem(item)

	item, _ := rbac.GetItem("ccc")
	marshal, _ := json.Marshal(item)
	fmt.Printf("---------%v\n", string(marshal))

}
