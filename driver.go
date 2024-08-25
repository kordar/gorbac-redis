package gorbac_redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/kordar/gorbac"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type RedisRbac struct {
	rdb   redis.UniversalClient
	table string
	mod   int
}

func NewRedisRbac(rdb redis.UniversalClient, tb string) *RedisRbac {
	return NewRedisRbacWithMod(rdb, tb, 10)
}

func NewRedisRbacWithMod(rdb redis.UniversalClient, tb string, mod int) *RedisRbac {
	return &RedisRbac{rdb: rdb, table: tb, mod: mod}
}

func (rbac *RedisRbac) key(tb string) string {
	return rbac.table + ":" + tb
}

func (rbac *RedisRbac) AddItem(item gorbac.Item) error {
	ctx := context.Background()
	authItem := ToAuthItem(item)
	key := rbac.key(authItem.TableName())
	return rbac.rdb.HSet(ctx, key, item.GetName(), &authItem).Err()
}

func (rbac *RedisRbac) scanFilterItems(t int32, f func(authItem AuthItem)) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item"))
	iter := rbac.rdb.HScan(ctx, key, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		element := AuthItem{}
		if err := element.UnmarshalBinaryStr(iter.Val()); err == nil {
			if t == gorbac.NoneType.Value() {
				f(element)
			} else if (t == gorbac.RoleType.Value() || t == gorbac.PermissionType.Value()) && t == element.Type {
				f(element)
			}
		}
	}
}

func (rbac *RedisRbac) GetItem(name string) (gorbac.Item, error) {
	ctx := context.Background()
	authItem := AuthItem{}
	key := rbac.key(authItem.TableName())
	err := rbac.rdb.HGet(ctx, key, name).Scan(&authItem)
	if err == nil {
		item := ToItem(authItem)
		return item, nil
	} else {
		return nil, err
	}
}

func (rbac *RedisRbac) GetItemsByType(itemType gorbac.ItemType) ([]gorbac.Item, error) {
	items := make([]gorbac.Item, 0)
	rbac.scanFilterItems(itemType.Value(), func(authItem AuthItem) {
		item := ToItem(authItem)
		items = append(items, item)
	})
	return items, nil
}

func (rbac *RedisRbac) FindAllItems() ([]gorbac.Item, error) {
	return rbac.GetItemsByType(gorbac.NoneType)
}

func (rbac *RedisRbac) cleanItems() {
	rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
		_ = rbac.RemoveItem(authItem.Name)
	})
}

func (rbac *RedisRbac) RemoveItem(name string) error {
	ctx := context.Background()

	// 解除所有父类关联name的元素，并删除子类、父类均为name的key
	itemChildKey := rbac.key(gorbac.GetTableName("item-child"))
	removeIds := make([]string, 0)
	rbac.scanItemChild(name+"::*", func(authItemChild AuthItemChild) {
		removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
	})
	rbac.scanItemChild("*::"+name, func(authItemChild AuthItemChild) {
		removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
	})
	if len(removeIds) > 0 {
		rbac.rdb.HDel(ctx, itemChildKey, removeIds...)
	}

	// 将所有assignment关联的itemName清除
	assigmentNameKey := rbac.assigmentNameKey(name)
	iter := rbac.rdb.SScan(ctx, assigmentNameKey, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		assigmentUserKey := rbac.assigmentUserKey(iter.Val())
		rbac.rdb.SRem(ctx, assigmentUserKey, name)
	}
	rbac.rdb.Del(ctx, assigmentNameKey)

	// 移除item
	itemKey := rbac.key(gorbac.GetTableName("item"))
	_ = rbac.rdb.HDel(ctx, itemKey, name)

	return nil
}

func (rbac *RedisRbac) UpdateItem(itemName string, updateItem gorbac.Item) error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	if itemName != updateItem.GetName() {
		// 校验更新待更新的item是否已存在
		if rbac.rdb.HExists(ctx, itemKey, updateItem.GetName()).Val() {
			return errors.New(fmt.Sprintf("item `%s` already exists", updateItem.GetName()))
		}

		itemChildKey := rbac.key(gorbac.GetTableName("item-child"))
		removeIds := make([]string, 0)
		rbac.scanItemChild(itemName+"::*", func(authItemChild AuthItemChild) {
			removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			_ = rbac.AddItemChild(gorbac.ItemChild{Parent: updateItem.GetName(), Child: authItemChild.Child})
		})

		if len(removeIds) > 0 {
			rbac.rdb.HDel(ctx, itemChildKey, removeIds...)
		}
		//
		removeIds = removeIds[:0]
		rbac.scanItemChild("*::"+itemName, func(authItemChild AuthItemChild) {
			removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			_ = rbac.AddItemChild(gorbac.ItemChild{Parent: authItemChild.Parent, Child: updateItem.GetName()})
		})

		if len(removeIds) > 0 {
			rbac.rdb.HDel(ctx, itemChildKey, removeIds...)
		}

		assigmentNameKey := rbac.assigmentNameKey(itemName)
		iter := rbac.rdb.SScan(ctx, assigmentNameKey, 0, "*", 0).Iterator()
		for iter.Next(ctx) {
			_, _ = rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				assigmentUserKey := rbac.assigmentUserKey(iter.Val())
				pipe.SRem(ctx, assigmentUserKey, itemName)
				pipe.SAdd(ctx, assigmentUserKey, updateItem.GetName())
				return nil
			})
		}

		targetAssigmentNameKey := rbac.assigmentNameKey(updateItem.GetName())
		rbac.rdb.Rename(ctx, assigmentNameKey, targetAssigmentNameKey)

		rbac.rdb.HDel(ctx, itemKey, itemName)
	}

	authItem := ToAuthItem(updateItem)
	authItem.UpdateTime = time.Now()

	rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
	return nil
}

func (rbac *RedisRbac) AddRule(rule gorbac.Rule) error {
	ctx := context.Background()
	authRule := ToAuthRule(rule)
	key := rbac.key(authRule.TableName())
	return rbac.rdb.HSet(ctx, key, authRule.Name, &authRule).Err()
}

func (rbac *RedisRbac) GetRule(name string) (*gorbac.Rule, error) {
	ctx := context.Background()
	authRule := AuthRule{}
	key := rbac.key(authRule.TableName())
	err := rbac.rdb.HGet(ctx, key, name).Scan(&authRule)
	if err == nil {
		rule := ToRule(authRule)
		return rule, nil
	} else {
		return nil, err
	}
}

func (rbac *RedisRbac) scanRules(f func(authRule AuthRule)) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("rule"))
	iter := rbac.rdb.HScan(ctx, key, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		ele := AuthRule{}
		if err := ele.UnmarshalBinaryStr(iter.Val()); err == nil {
			f(ele)
		}
	}
}

func (rbac *RedisRbac) GetRules() ([]*gorbac.Rule, error) {
	rules := make([]*gorbac.Rule, 0)
	rbac.scanRules(func(authRule AuthRule) {
		rule := ToRule(authRule)
		rules = append(rules, rule)
	})
	return rules, nil
}

func (rbac *RedisRbac) RemoveRule(ruleName string) error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
		if authItem.RuleName != ruleName {
			return
		}
		authItem.RuleName = ""
		authItem.UpdateTime = time.Now()
		rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
	})

	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	rbac.rdb.HDel(ctx, ruleKey, ruleName)
	return nil
}

func (rbac *RedisRbac) UpdateRule(ruleName string, updateRule gorbac.Rule) error {
	ctx := context.Background()
	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	if ruleName != updateRule.Name {
		// 校验更新待更新的rule是否已存在
		if rbac.rdb.HExists(ctx, ruleKey, updateRule.Name).Val() {
			return errors.New(fmt.Sprintf("rule `%s` already exists", updateRule.Name))
		}

		itemKey := rbac.key(gorbac.GetTableName("item"))
		rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
			if authItem.RuleName != ruleName {
				return
			}
			authItem.RuleName = updateRule.Name
			authItem.UpdateTime = time.Now()
			rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
		})
	}

	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HDel(ctx, ruleKey, ruleName)
		authRule := ToAuthRule(updateRule)
		pipe.HSet(ctx, ruleKey, authRule.Name, &authRule)
		return nil
	})

	return err
}

func (rbac *RedisRbac) scanItemChild(match string, f func(authItemChild AuthItemChild)) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item-child"))
	iter := rbac.rdb.HScan(ctx, key, 0, match, 0).Iterator()
	for iter.Next(ctx) {
		ele := AuthItemChild{}
		if err := ele.UnmarshalBinaryStr(iter.Val()); err == nil {
			f(ele)
		}
	}
}

func (rbac *RedisRbac) itemChildKey(parent string, child string) string {
	return fmt.Sprintf("%s::%s", parent, child)
}

func (rbac *RedisRbac) AddItemChild(itemChild gorbac.ItemChild) error {
	ctx := context.Background()
	authItemChild := ToAuthItemChild(itemChild.Parent, itemChild.Child)
	key := rbac.key(authItemChild.TableName())
	childKey := rbac.itemChildKey(authItemChild.Parent, authItemChild.Child)
	return rbac.rdb.HSet(ctx, key, childKey, &authItemChild).Err()
}

func (rbac *RedisRbac) RemoveChild(parent string, child string) error {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item-child"))
	childKey := rbac.itemChildKey(parent, child)
	rbac.rdb.HDel(ctx, key, childKey)
	return nil
}

func (rbac *RedisRbac) RemoveChildParentByNames(names []string) error {
	if names != nil && len(names) > 0 {
		removeIds := make([]string, 0)
		for _, name := range names {
			rbac.scanItemChild(name+"::*", func(authItemChild AuthItemChild) {
				removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			})
		}
		if len(removeIds) > 0 {
			ctx := context.Background()
			key := rbac.key(gorbac.GetTableName("item-child"))
			rbac.rdb.HDel(ctx, key, removeIds...)
		}
	}
	return nil
}

func (rbac *RedisRbac) RemoveChildChildByNames(names []string) error {
	if names != nil && len(names) > 0 {
		removeIds := make([]string, 0)
		for _, name := range names {
			rbac.scanItemChild("*::"+name, func(authItemChild AuthItemChild) {
				removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			})
		}
		if len(removeIds) > 0 {
			ctx := context.Background()
			key := rbac.key(gorbac.GetTableName("item-child"))
			rbac.rdb.HDel(ctx, key, removeIds...)
		}
	}
	return nil
}

func (rbac *RedisRbac) RemoveChildByNames(t gorbac.ItemType, names []string) error {
	if t == gorbac.PermissionType {
		return rbac.RemoveChildChildByNames(names)
	} else {
		return rbac.RemoveChildParentByNames(names)
	}
}

func (rbac *RedisRbac) RemoveItemByType(itemType gorbac.ItemType) error {
	removeIds := make([]string, 0)
	rbac.scanFilterItems(itemType.Value(), func(authItem AuthItem) {
		removeIds = append(removeIds, authItem.Name)
	})
	if len(removeIds) > 0 {
		ctx := context.Background()
		rbac.rdb.HDel(ctx, rbac.key(gorbac.GetTableName("item")), removeIds...)
	}
	return nil
}

func (rbac *RedisRbac) RemoveChildren(parent string) error {
	keys := make([]string, 0)
	rbac.scanItemChild(parent+"::*", func(authItemChild AuthItemChild) {
		childKey := rbac.itemChildKey(authItemChild.Parent, authItemChild.Child)
		keys = append(keys, childKey)
	})
	if len(keys) > 0 {
		ctx := context.Background()
		key := rbac.key(gorbac.GetTableName("item-child"))
		rbac.rdb.HDel(ctx, key, keys...)
	}
	return nil
}

func (rbac *RedisRbac) HasChild(parent string, child string) bool {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item-child"))
	childKey := rbac.itemChildKey(parent, child)
	return rbac.rdb.HExists(ctx, key, childKey).Val()
}

func (rbac *RedisRbac) FindChildren(name string) ([]gorbac.Item, error) {

	itemExits := make(map[string]bool, 0)
	rbac.scanItemChild(name+"::*", func(authItemChild AuthItemChild) {
		itemExits[authItemChild.Child] = true
	})

	items := make([]gorbac.Item, 0)
	if len(itemExits) > 0 {
		rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
			if itemExits[authItem.Name] {
				item := ToItem(authItem)
				items = append(items, item)
			}
		})
	}

	return items, nil
}

func (rbac *RedisRbac) FindChildrenList() ([]*gorbac.ItemChild, error) {
	children := make([]*gorbac.ItemChild, 0)
	rbac.scanItemChild("*", func(authItemChild AuthItemChild) {
		itemChild := ToItemChild(authItemChild)
		children = append(children, itemChild)
	})
	return children, nil
}

func (rbac *RedisRbac) FindChildrenFormChild(child string) ([]*gorbac.ItemChild, error) {
	children := make([]*gorbac.ItemChild, 0)
	rbac.scanItemChild("*::"+child, func(authItemChild AuthItemChild) {
		itemChild := ToItemChild(authItemChild)
		children = append(children, itemChild)
	})
	return children, nil
}

func (rbac *RedisRbac) assigmentNameKey(name string) string {
	key := rbac.key(gorbac.GetTableName("assignment"))
	return fmt.Sprintf("%s-n:%s", key, name)
}

func (rbac *RedisRbac) assigmentUserKey(userId interface{}) string {
	key := rbac.key(gorbac.GetTableName("assignment"))
	return fmt.Sprintf("%s-u:%v", key, userId)
}

func (rbac *RedisRbac) Assign(assignment gorbac.Assignment) error {
	ctx := context.Background()
	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.SAdd(ctx, rbac.assigmentUserKey(assignment.UserId), assignment.ItemName)
		pipe.SAdd(ctx, rbac.assigmentNameKey(assignment.ItemName), assignment.UserId)
		return nil
	})

	return err
}

func (rbac *RedisRbac) RemoveAssignment(userId interface{}, name string) error {
	ctx := context.Background()
	if rbac.rdb.SIsMember(ctx, rbac.assigmentUserKey(userId), name).Val() {
		_, _ = rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.SRem(ctx, rbac.assigmentUserKey(userId), name)
			pipe.SRem(ctx, rbac.assigmentNameKey(name), userId)
			return nil
		})
	}
	return nil
}

func (rbac *RedisRbac) removeAssignmentByName(name string) {
	ctx := context.Background()
	iter := rbac.rdb.SScan(ctx, rbac.assigmentNameKey(name), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		userId := iter.Val()
		rbac.rdb.SRem(ctx, rbac.assigmentUserKey(userId), name)
	}
	rbac.rdb.Del(ctx, rbac.assigmentNameKey(name))
}

func (rbac *RedisRbac) RemoveAssignmentByNames(names []string) error {
	if names != nil && len(names) > 0 {
		for _, name := range names {
			rbac.removeAssignmentByName(name)
		}
	}
	return nil
}

func (rbac *RedisRbac) RemoveAllAssignmentByUser(userId interface{}) error {
	ctx := context.Background()
	iter := rbac.rdb.SScan(ctx, rbac.assigmentUserKey(userId), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		name := iter.Val()
		rbac.rdb.SRem(ctx, rbac.assigmentNameKey(name), userId)
	}
	rbac.rdb.Del(ctx, rbac.assigmentUserKey(userId))
	return nil
}

func (rbac *RedisRbac) RemoveAllAssignments() error {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, key+"-*", 0).Iterator()
	for iter.Next(ctx) {
		rbac.rdb.Del(ctx, iter.Val())
	}
	return iter.Err()
}

func (rbac *RedisRbac) GetAssignment(userId interface{}, name string) (*gorbac.Assignment, error) {
	ctx := context.Background()
	if rbac.rdb.SIsMember(ctx, rbac.assigmentUserKey(userId), name).Val() {
		return gorbac.NewAssignment(userId, name), nil
	} else {
		return nil, errors.New("assignment not found")
	}
}

func (rbac *RedisRbac) FindAssignmentsByUser(userId interface{}) ([]*gorbac.Assignment, error) {
	return rbac.GetAssignments(userId)
}

func (rbac *RedisRbac) GetAssignmentsByItem(name string) ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	assignments := make([]*gorbac.Assignment, 0)
	iter := rbac.rdb.SScan(ctx, rbac.assigmentNameKey(name), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		assignment := gorbac.NewAssignment(iter.Val(), name)
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func (rbac *RedisRbac) GetAssignments(userId interface{}) ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	assignments := make([]*gorbac.Assignment, 0)
	iter := rbac.rdb.SScan(ctx, rbac.assigmentUserKey(userId), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		assignment := gorbac.NewAssignment(userId, iter.Val())
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func (rbac *RedisRbac) GetAllAssignment() ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, fmt.Sprintf("%s-n:*", key), 0).Iterator()
	assignments := make([]*gorbac.Assignment, 0)
	for iter.Next(ctx) {
		name := strings.Replace(iter.Val(), fmt.Sprintf("%s-n:", key), "", 1)
		if items, err := rbac.GetAssignmentsByItem(name); err == nil {
			assignments = append(assignments, items...)
		}
	}
	return assignments, nil
}

func (rbac *RedisRbac) findItemsByUser(userId interface{}, t int32) ([]gorbac.Item, error) {
	ctx := context.Background()
	assigmentUserKey := rbac.assigmentUserKey(userId)
	itemFields := rbac.rdb.SMembers(ctx, assigmentUserKey).Val()

	items := make([]gorbac.Item, 0)
	if itemFields != nil && len(itemFields) > 0 {
		itemKey := rbac.key(gorbac.GetTableName("item"))
		values := rbac.rdb.HMGet(ctx, itemKey, itemFields...).Val()
		for _, value := range values {
			authItem := AuthItem{}
			if err := authItem.UnmarshalBinaryStr(value.(string)); err == nil {
				if authItem.Type != t {
					continue
				}
				item := ToItem(authItem)
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// FindRolesByUser 通过会员id获取关联的所有角色
func (rbac *RedisRbac) FindRolesByUser(userId interface{}) ([]gorbac.Item, error) {
	return rbac.findItemsByUser(userId, gorbac.RoleType.Value())
}

func (rbac *RedisRbac) GetItemList(t int32, names []string) ([]gorbac.Item, error) {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	items := make([]gorbac.Item, 0)
	values := rbac.rdb.HMGet(ctx, itemKey, names...).Val()
	for _, value := range values {
		authItem := AuthItem{}
		if err := authItem.UnmarshalBinaryStr(value.(string)); err == nil {
			if authItem.Type != t {
				continue
			}
			item := ToItem(authItem)
			items = append(items, item)
		}
	}

	return items, nil
}

func (rbac *RedisRbac) FindPermissionsByUser(userId interface{}) ([]gorbac.Item, error) {
	return rbac.findItemsByUser(userId, gorbac.PermissionType.Value())
}

func (rbac *RedisRbac) RemoveAll() error {
	rbac.cleanItems()
	rbac.cleanRules()
	return nil
}

func (rbac *RedisRbac) cleanRules() {
	ctx := context.Background()
	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	rbac.rdb.Del(ctx, ruleKey)
}

func (rbac *RedisRbac) RemoveAllRules() error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
		authItem.RuleName = ""
		authItem.UpdateTime = time.Now()
		rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
	})
	rbac.cleanRules()
	return nil
}
