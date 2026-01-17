# GoRBAC Redis 实现

`gorbac_redis` 是基于 [GoRBAC](https://github.com/kordar/gorbac) 的 Redis 存储实现，支持 RBAC 权限管理系统在 Redis 中的持久化。  
支持角色（Role）、权限（Permission）、规则（Rule）、子项继承关系、用户分配（Assignment）等。

---

## 功能特点

- **权限节点存储**：角色、权限可持久化存储到 Redis
- **规则与执行器**：支持绑定规则，实现动态权限校验
- **父子继承关系**：角色和权限可形成父子层级
- **用户分配**：支持用户与角色/权限绑定，自动管理 assignment
- **Redis 集群兼容**：支持单机、哨兵、集群模式
- **批量操作**：支持批量分配、批量删除
- **缓存友好**：基于 Redis，查询效率高

---

## 安装

```bash
go get github.com/kordar/gorbac-redis
```

依赖：

- Go 1.20+
- [github.com/kordar/gorbac](https://github.com/kordar/gorbac)
- [github.com/redis/go-redis/v9](https://github.com/redis/go-redis)

------

## 快速使用示例

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/kordar/gorbac"
    "github.com/kordar/gorbac_redis"
    "github.com/redis/go-redis/v9"
)

func main() {
    // 初始化 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    // 初始化 Redis RBAC
    rbac := gorbac_redis.NewRedisRbac(rdb, "gorbac")

    // 创建角色和权限
    adminRole := gorbac.NewRole("admin", "管理员角色", "", "", time.Now(), time.Now())
    viewPerm := gorbac.NewPermission("view_dashboard", "查看仪表盘权限", "", "", time.Now(), time.Now())

    _ = rbac.AddItem(adminRole)
    _ = rbac.AddItem(viewPerm)

    // 建立父子关系
    _ = rbac.AddItemChild(gorbac.ItemChild{Parent: "admin", Child: "view_dashboard"})

    // 用户分配角色
    userId := 1001
    assignment := gorbac.Assignment{UserId: userId, ItemName: "admin", CreateTime: time.Now()}
    _ = rbac.Assign(assignment)

    // 查询用户角色
    roles, _ := rbac.FindRolesByUser(userId)
    fmt.Println("用户角色:", roles)

    // 查询用户权限
    perms, _ := rbac.FindPermissionsByUser(userId)
    fmt.Println("用户权限:", perms)
}
```

------

## 核心接口说明

### RedisRbac

- `AddItem(item gorbac.Item) error` – 添加角色或权限
- `GetItem(name string) (gorbac.Item, error)` – 获取角色或权限
- `GetItemsByType(itemType gorbac.ItemType) ([]gorbac.Item, error)` – 按类型获取节点
- `RemoveItem(name string) error` – 删除角色或权限
- `UpdateItem(itemName string, updateItem gorbac.Item) error` – 更新节点信息
- `AddRule(rule gorbac.Rule) error` – 添加规则
- `GetRule(name string) (*gorbac.Rule, error)` – 获取规则
- `RemoveRule(ruleName string) error` – 删除规则
- `UpdateRule(ruleName string, updateRule gorbac.Rule) error` – 更新规则
- `AddItemChild(itemChild gorbac.ItemChild) error` – 添加父子关系
- `RemoveChild(parent, child string) error` – 删除父子关系
- `HasChild(parent, child string) bool` – 判断是否存在父子关系
- `FindChildren(name string) ([]gorbac.Item, error)` – 查找某节点的子节点
- `Assign(assignment gorbac.Assignment) error` – 用户分配角色/权限
- `Assigns(assignments ...*gorbac.Assignment) error` – 批量分配
- `RemoveAssignment(userId interface{}, name string) error` – 移除用户分配
- `RemoveAllAssignmentByUser(userId interface{}) error` – 删除用户所有分配
- `FindRolesByUser(userId interface{}) ([]gorbac.Item, error)` – 获取用户所有角色
- `FindPermissionsByUser(userId interface{}) ([]gorbac.Item, error)` – 获取用户所有权限
- `RemoveAll() error` – 清空所有节点与规则
- `RemoveAllRules() error` – 清空所有规则

------

## 数据结构

- **AuthItem**：角色/权限节点存储
- **AuthRule**：规则存储
- **AuthItemChild**：节点父子关系存储
- **AuthAssignment**：用户分配存储

数据存储均为 Redis Hash 或 Set 结构，支持快速扫描与批量操作。

------

## License

MIT License
