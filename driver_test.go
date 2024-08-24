package gorbac_redis_test

import (
	"encoding/json"
	logger "github.com/kordar/gologger"
	"github.com/kordar/gorbac"
	gorbac_redis "github.com/kordar/gorbac-redis"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func handle() *gorbac_redis.RedisRbac {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{"43.139.223.7:8088"},
		Password: "940430Dex",
	})
	return gorbac_redis.NewRedisRbac(rdb, "test")
}

func print(item interface{}) {
	marshal, _ := json.Marshal(item)
	logger.Infof("------------%v", string(marshal))
}

func TestRedis(t *testing.T) {
	rbac := handle()
	//ctx := context.Background()
	//rdb.SAdd(ctx, "aaa", "2324", "32132")
	//authItem := gorbac_redis.AuthItem{
	//	Name:        "bbb",
	//	Type:        0,
	//	Description: "",
	//	RuleName:    "",
	//	ExecuteName: "",
	//	CreateTime:  time.Now(),
	//	UpdateTime:  time.Now(),
	//}
	//item := gorbac_redis.ToItem(authItem)
	//err := rbac.AddItem(item)

	//item, err := rbac.GetItem("bbb")

	//
	items, err := rbac.GetItems(1)
	logger.Infof("--------------%v", err)
	print(items)

	//items, _ := rbac.FindAllItems()
	//logger.Infof("============%+v", items)

}

func TestRules(t *testing.T) {
	rbac := handle()
	rule := gorbac.Rule{
		Name:        "theRule",
		ExecuteName: "xxxx",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}
	addErr := rbac.AddRule(rule)
	logger.Infof("add rule = %v", addErr)
	getRule, getRuleErr := rbac.GetRule("theRule")
	logger.Infof("get rule = %v", getRuleErr)
	print(getRule)
	rules, getRulesErr := rbac.GetRules()
	logger.Infof("get rules = %v", getRulesErr)
	print(rules)
	//rbac.RemoveRule("theRule")
	rule2 := gorbac.Rule{
		Name:        "TTT",
		ExecuteName: "ccc",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}
	rbac.UpdateRule("AAA", rule2)
}

func TestChildren(t *testing.T) {
	rbac := handle()
	//_ = rbac.AddItemChild(gorbac.ItemChild{"AAA", "BBB"})
	//_ = rbac.AddItemChild(gorbac.ItemChild{"AAA", "CCC"})
	//_ = rbac.AddItemChild(gorbac.ItemChild{"AAA", "DDD"})
	//rbac.RemoveChild("AAA", "CCC")
	//rbac.RemoveChildren("AAA")
	//logger.Infof("----------%v", rbac.HasChild("AAA", "BBB"))
	//logger.Infof("----------%v", rbac.HasChild("AAA", "EEE"))
	children, err := rbac.FindChildren("AAA")
	//children, err := rbac.FindChildren("ccc")
	logger.Infof("-------------%v", err)
	print(children)
	//child := rbac.HasChild("ccc", "bbb")
	//logger.Infof("========%v", child)
	//rbac.RemoveChildren("ccc")
}

func TestRedisRbac_GetAssignment(t *testing.T) {
	rbac := handle()
	//assignment := gorbac.AuthAssignment{
	//	ItemName:   "bbb",
	//	UserId:     10000,
	//	CreateTime: time.Now(),
	//}
	//rbac.Assign(assignment)
	//assignment, err := rbac.GetAssignment(10000, "bbb")
	rbac.RemoveAllAssignments()

	//logger.Infof("-------------%v", err)
	//data, _ := json.Marshal(assignment)
	//logger.Infof("=============%v", string(data))
}
