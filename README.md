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

- Go 1.18+
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
    gorbac_redis "github.com/kordar/gorbac-redis"
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

## 作为 CacheStore 使用（可选）

如果你的核心库 gorbac 开启了进程内缓存，同时希望在多实例间共享 RBAC 快照（items/rules/parents），可以使用独立仓库 `gorbac-cache-redis` 提供的 `RedisCacheStore` 注入到 gorbac：

```go
package main

import (
    "time"

    "github.com/kordar/gorbac"
    gorbac_cache_redis "github.com/kordar/gorbac-cache-redis"
    "github.com/redis/go-redis/v9"
)

func main() {
    repo := NewYourAuthRepository()
    rdb := redis.NewClient(&redis.Options{ Addr: "127.0.0.1:6379" })
    store := gorbac_cache_redis.NewRedisCacheStore(rdb)

    service := gorbac.NewRbacServiceWithCacheStore(
        repo,
        true,          // 开启进程内缓存
        store,         // Redis 快照存储
        "myapp",       // key 前缀
        10*time.Minute // 快照 TTL
    )
    _ = service
}
```

键示例：`myapp:rbac:snapshot`。

------

## 连接模式示例

- 单机：
```go
redis.NewClient(&redis.Options{ Addr: "127.0.0.1:6379" })
```
- 哨兵：
```go
redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName: "mymaster",
    SentinelAddrs: []string{"10.0.0.1:26379","10.0.0.2:26379"},
})
```
- 集群（推荐 UniversalClient）：
```go
redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{"10.0.0.1:6379","10.0.0.2:6379","10.0.0.3:6379"},
})
```

------

## 测试说明

仓库内的集成测试会在未设置环境变量时跳过，避免连接真实 Redis：
- `RBAC_REDIS_ADDRS`：多个地址用`,`分隔
- `RBAC_REDIS_PASSWORD`：密码，可选
- `RBAC_REDIS_TABLE`：表前缀，默认 `RBAC_TEST`

## License

MIT License
