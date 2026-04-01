package gorbac_redis_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"log/slog"
	"github.com/kordar/gorbac"
	gorbac_redis "github.com/kordar/gorbac-redis"
	"github.com/redis/go-redis/v9"
)

func handle(t *testing.T) *gorbac_redis.RedisRbac {
	addrsEnv := strings.TrimSpace(os.Getenv("RBAC_REDIS_ADDRS"))
	password := os.Getenv("RBAC_REDIS_PASSWORD")
	table := os.Getenv("RBAC_REDIS_TABLE")
	if table == "" {
		table = "RBAC_TEST"
	}
	// 没有提供地址则跳过集成测试，避免泄露凭据或连接失败
	if addrsEnv == "" {
		t.Skip("skip redis integration tests: set RBAC_REDIS_ADDRS to enable")
		return nil
	}
	addrs := strings.Split(addrsEnv, ",")
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addrs,
		Password: password,
	})
	return gorbac_redis.NewRedisRbac(rdb, table)
}

func TestService(t *testing.T) {
	service := gorbac.NewRbacService(handle(t), false)
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
	slog.Info("------------------", "flag", flag)
}

func print(item interface{}) {
	marshal, _ := json.Marshal(item)
	slog.Info("------------", "value", string(marshal))
}

func TestRedis(t *testing.T) {
	rbac := handle(t)
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
	slog.Info("--------------", "err", err)
}

func TestRules(t *testing.T) {
	rbac := handle(t)
	rule := gorbac.Rule{
		Name:        "theRule",
		ExecuteName: "xxxx",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}
	addErr := rbac.AddRule(rule)
	slog.Info("add rule", "err", addErr)
	getRule, getRuleErr := rbac.GetRule("theRule")
	slog.Info("get rule", "err", getRuleErr)
	print(getRule)
	rules, getRulesErr := rbac.GetRules()
	slog.Info("get rules", "err", getRulesErr)
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
	rbac := handle(t)
	_ = rbac.AddItemChild(gorbac.ItemChild{"TTT", "BBB"})
	_ = rbac.AddItemChild(gorbac.ItemChild{"TTT", "CCC"})
	_ = rbac.AddItemChild(gorbac.ItemChild{"DDD", "TTT"})
	//rbac.RemoveChild("AAA", "CCC")
	//rbac.RemoveChildren("AAA")
	//logger.Infof("----------%v", rbac.HasChild("AAA", "BBB"))
	//logger.Infof("----------%v", rbac.HasChild("AAA", "EEE"))
	children, err := rbac.FindChildren("AAA")
	//children, err := rbac.FindChildren("ccc")
	slog.Info("-------------", "err", err)
	print(children)
	//child := rbac.HasChild("ccc", "bbb")
	//logger.Infof("========%v", child)
	//rbac.RemoveChildren("ccc")
}

func TestRedisRbac_GetAssignment(t *testing.T) {
	rbac := handle(t)
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
	rbac := handle(t)
	//user, err := rbac.FindPermissionsByUser(124)
	//list, err := rbac.FindChildrenList()
	list, err := rbac.GetItemList(2, []string{"CCC"})
	slog.Info("--------------", "err", err)
	print(list)
}

func TestManager(t *testing.T) {
	service := gorbac.NewRbacService(handle(t), false)
	manager := service.GetAuthManager()
	role := gorbac.NewRole("guest", "guest", "", "", time.Now(), time.Now())
	p1 := gorbac.NewPermission("AAA", "", "demo", "demo", time.Now(), time.Now())
	p2 := gorbac.NewPermission("BBB", "", "", "", time.Now(), time.Now())
	p3 := gorbac.NewPermission("CCC", "", "", "", time.Now(), time.Now())
	manager.Add(role)
	manager.Add(p1)
	manager.Add(p2)
	manager.Add(p3)
	manager.AddChild(role, p1)
	manager.AddChild(role, p2)
	manager.AddChild(role, p3)
	admin := manager.CreateRole("admin")
	manager.Add(admin)
	manager.AddChild(admin, role)
	err := manager.AddChild(role, admin)
	slog.Error("xxxxxxxxx", "err", err)
	//manager.SetDefaultRoles(role)

	gorbac.ExecuteManager.AddExecutor(&gorbac.DemoExecutor{})

	//manager.Assign(role, 213)

	//manager.RemoveAllAssignmentByUser(213)

	access := manager.CheckAccess(nil, 213, "AAA")
	slog.Info("=================", "access", access)
}
