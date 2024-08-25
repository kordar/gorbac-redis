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
	return gorbac_redis.NewRedisRbac(rdb, "TEST0000001")
}

func TestService(t *testing.T) {
	service := gorbac.NewRbacService(handle(), false)
	//service.AddRole("AAA", "", "")
	//service.AddPermission("BBB", "", "")
	//err := service.AssignRole("AAA", "BBB")
	//roles := service.Roles()
	//logger.Info("------------------", roles, err)
	//err := service.AssignPermission("BBB", "AAA")
	//logger.Info("------------------", err)
	//service.CleanChildren("AAA")
	//err := service.AssignChildren("AAA", "BBB", "BBB")
	//flag := service.Assign(123, "AAA")
	//service.CleanAssigns(123)
	flag := service.UpdateRole("AAA", "TTT", "teste", "")
	logger.Info("------------------", flag)
}

func print(item interface{}) {
	marshal, _ := json.Marshal(item)
	logger.Infof("------------%v", string(marshal))
}

func TestRedis(t *testing.T) {
	rbac := handle()
	//ctx := context.Background()
	//rdb.SAdd(ctx, "aaa", "2324", "32132")
	//authItem := gorbac_redis.AuthItem{Name: "DDD", Type: 0, Description: "", RuleName: "", ExecuteName: "", CreateTime: time.Now(), UpdateTime: time.Now()}
	//item := gorbac_redis.ToItem(authItem)
	//err := rbac.AddItem(item)

	//item, err := rbac.GetItem("bbb")

	//
	//items, err := rbac.GetItems(1)
	//logger.Infof("--------------%v", err)
	//print(items)

	//items, _ := rbac.FindAllItems()
	//logger.Infof("============%+v", items)
	//err := rbac.UpdateItem("AAA", gorbac.NewPermission("TTT", "test", "", "", time.Now(), time.Now()))
	err := rbac.RemoveItem("TTT")
	logger.Infof("--------------%v", err)
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
	_ = rbac.AddItemChild(gorbac.ItemChild{"TTT", "BBB"})
	_ = rbac.AddItemChild(gorbac.ItemChild{"TTT", "CCC"})
	_ = rbac.AddItemChild(gorbac.ItemChild{"DDD", "TTT"})
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
	rbac.Assign(*gorbac.NewAssignment(123, "AAA"))
	rbac.Assign(*gorbac.NewAssignment(123, "BBB"))
	rbac.Assign(*gorbac.NewAssignment(124, "BBB"))
	//rbac.RemoveAssignment(123, "AAA")
	//rbac.RemoveAllAssignmentByUser(123)
	//rbac.RemoveAllAssignments()
	//assignment, _ := rbac.GetAssignment(123, "BBB")
	//items, _ := rbac.GetAssignmentByItems("BBB")
	//assignments, _ := rbac.GetAssignments(124)
	assignments, _ := rbac.GetAllAssignment()
	print(assignments)
}

func TestUser(t *testing.T) {
	rbac := handle()
	//user, err := rbac.FindPermissionsByUser(124)
	//list, err := rbac.FindChildrenList()
	list, err := rbac.GetItemList(2, []string{"CCC"})
	logger.Infof("--------------%v", err)
	print(list)
}
