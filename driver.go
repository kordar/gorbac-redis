package gorbac_redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kordar/gorbac"
	"github.com/redis/go-redis/v9"
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

func (rbac *RedisRbac) hscanEachValue(ctx context.Context, key string, match string, count int64, f func(value string)) error {
	var cursor uint64
	for {
		pairs, next, err := rbac.rdb.HScan(ctx, key, cursor, match, count).Result()
		if err != nil {
			return err
		}

		for i := 1; i < len(pairs); i += 2 {
			f(pairs[i])
		}

		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (rbac *RedisRbac) scanFilterItems(t int32, f func(authItem AuthItem)) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item"))
	_ = rbac.hscanEachValue(ctx, key, "*", 0, func(value string) {
		element := AuthItem{}
		if err := element.UnmarshalBinaryStr(value); err != nil {
			return
		}
		if t == gorbac.NoneType.Value() {
			f(element)
			return
		}
		if (t == gorbac.RoleType.Value() || t == gorbac.PermissionType.Value()) && t == element.Type {
			f(element)
		}
	})
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
		if err := rbac.rdb.HDel(ctx, itemChildKey, removeIds...).Err(); err != nil {
			return err
		}
	}

	// 将所有assignment关联的itemName清除
	assigmentNameKey := rbac.assigmentNameKey(name)
	iter := rbac.rdb.SScan(ctx, assigmentNameKey, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		assigmentUserKey := rbac.assigmentUserKey(iter.Val())
		if err := rbac.rdb.SRem(ctx, assigmentUserKey, name).Err(); err != nil {
			return err
		}
	}
	if err := rbac.rdb.Del(ctx, assigmentNameKey).Err(); err != nil {
		return err
	}

	// 移除item
	itemKey := rbac.key(gorbac.GetTableName("item"))
	if err := rbac.rdb.HDel(ctx, itemKey, name).Err(); err != nil {
		return err
	}

	return nil
}

func (rbac *RedisRbac) UpdateItem(itemName string, updateItem gorbac.Item) error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	if itemName != updateItem.GetName() {
		// 校验更新待更新的item是否已存在
		if rbac.rdb.HExists(ctx, itemKey, updateItem.GetName()).Val() {
			return fmt.Errorf("item `%s` already exists", updateItem.GetName())
		}

		itemChildKey := rbac.key(gorbac.GetTableName("item-child"))
		removeIds := make([]string, 0)
		var childErr error
		rbac.scanItemChild(itemName+"::*", func(authItemChild AuthItemChild) {
			if childErr != nil {
				return
			}
			removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			childErr = rbac.AddItemChild(gorbac.ItemChild{Parent: updateItem.GetName(), Child: authItemChild.Child})
		})
		if childErr != nil {
			return childErr
		}

		if len(removeIds) > 0 {
			if err := rbac.rdb.HDel(ctx, itemChildKey, removeIds...).Err(); err != nil {
				return err
			}
		}
		//
		removeIds = removeIds[:0]
		childErr = nil
		rbac.scanItemChild("*::"+itemName, func(authItemChild AuthItemChild) {
			if childErr != nil {
				return
			}
			removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			childErr = rbac.AddItemChild(gorbac.ItemChild{Parent: authItemChild.Parent, Child: updateItem.GetName()})
		})
		if childErr != nil {
			return childErr
		}

		if len(removeIds) > 0 {
			if err := rbac.rdb.HDel(ctx, itemChildKey, removeIds...).Err(); err != nil {
				return err
			}
		}

		assigmentNameKey := rbac.assigmentNameKey(itemName)
		iter := rbac.rdb.SScan(ctx, assigmentNameKey, 0, "*", 0).Iterator()
		for iter.Next(ctx) {
			_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				assigmentUserKey := rbac.assigmentUserKey(iter.Val())
				pipe.SRem(ctx, assigmentUserKey, itemName)
				pipe.SAdd(ctx, assigmentUserKey, updateItem.GetName())
				return nil
			})
			if err != nil {
				return err
			}
		}
		if err := iter.Err(); err != nil {
			return err
		}

		targetAssigmentNameKey := rbac.assigmentNameKey(updateItem.GetName())
		_ = rbac.rdb.Rename(ctx, assigmentNameKey, targetAssigmentNameKey).Err()

		if err := rbac.rdb.HDel(ctx, itemKey, itemName).Err(); err != nil {
			return err
		}
	}

	authItem := ToAuthItem(updateItem)
	authItem.UpdateTime = time.Now()

	return rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem).Err()
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
	_ = rbac.hscanEachValue(ctx, key, "*", 0, func(value string) {
		ele := AuthRule{}
		if err := ele.UnmarshalBinaryStr(value); err != nil {
			return
		}
		f(ele)
	})
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
		_ = rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem).Err()
	})

	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	return rbac.rdb.HDel(ctx, ruleKey, ruleName).Err()
}

func (rbac *RedisRbac) UpdateRule(ruleName string, updateRule gorbac.Rule) error {
	ctx := context.Background()
	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	if ruleName != updateRule.Name {
		// 校验更新待更新的rule是否已存在
		if rbac.rdb.HExists(ctx, ruleKey, updateRule.Name).Val() {
			return fmt.Errorf("rule `%s` already exists", updateRule.Name)
		}

		itemKey := rbac.key(gorbac.GetTableName("item"))
		rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
			if authItem.RuleName != ruleName {
				return
			}
			authItem.RuleName = updateRule.Name
			authItem.UpdateTime = time.Now()
			_ = rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem).Err()
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
	_ = rbac.hscanEachValue(ctx, key, match, 0, func(value string) {
		ele := AuthItemChild{}
		if err := ele.UnmarshalBinaryStr(value); err != nil {
			return
		}
		f(ele)
	})
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
	return rbac.rdb.HDel(ctx, key, childKey).Err()
}

func (rbac *RedisRbac) RemoveChildParentByNames(names []string) error {
	if len(names) > 0 {
		removeIds := make([]string, 0)
		for _, name := range names {
			rbac.scanItemChild(name+"::*", func(authItemChild AuthItemChild) {
				removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			})
		}
		if len(removeIds) > 0 {
			ctx := context.Background()
			key := rbac.key(gorbac.GetTableName("item-child"))
			return rbac.rdb.HDel(ctx, key, removeIds...).Err()
		}
	}
	return nil
}

func (rbac *RedisRbac) RemoveChildChildByNames(names []string) error {
	if len(names) > 0 {
		removeIds := make([]string, 0)
		for _, name := range names {
			rbac.scanItemChild("*::"+name, func(authItemChild AuthItemChild) {
				removeIds = append(removeIds, rbac.itemChildKey(authItemChild.Parent, authItemChild.Child))
			})
		}
		if len(removeIds) > 0 {
			ctx := context.Background()
			key := rbac.key(gorbac.GetTableName("item-child"))
			return rbac.rdb.HDel(ctx, key, removeIds...).Err()
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
		return rbac.rdb.HDel(ctx, rbac.key(gorbac.GetTableName("item")), removeIds...).Err()
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
		return rbac.rdb.HDel(ctx, key, keys...).Err()
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

func (rbac *RedisRbac) Assigns(assignments ...*gorbac.Assignment) error {
	if len(assignments) == 0 {
		return nil
	}

	ctx := context.Background()
	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, a := range assignments {
			pipe.SAdd(ctx, rbac.assigmentUserKey(a.UserId), a.ItemName)
			pipe.SAdd(ctx, rbac.assigmentNameKey(a.ItemName), a.UserId)
		}
		return nil
	})
	return err
}

func (rbac *RedisRbac) RemoveAssignment(userId interface{}, name string) error {
	ctx := context.Background()
	if rbac.rdb.SIsMember(ctx, rbac.assigmentUserKey(userId), name).Val() {
		_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.SRem(ctx, rbac.assigmentUserKey(userId), name)
			pipe.SRem(ctx, rbac.assigmentNameKey(name), userId)
			return nil
		})
		return err
	}
	return nil
}

func (rbac *RedisRbac) removeAssignmentByName(name string) error {
	ctx := context.Background()
	iter := rbac.rdb.SScan(ctx, rbac.assigmentNameKey(name), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		userId := iter.Val()
		if err := rbac.rdb.SRem(ctx, rbac.assigmentUserKey(userId), name).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return rbac.rdb.Del(ctx, rbac.assigmentNameKey(name)).Err()
}

func (rbac *RedisRbac) RemoveAssignmentByNames(names []string) error {
	if len(names) > 0 {
		for _, name := range names {
			if err := rbac.removeAssignmentByName(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rbac *RedisRbac) RemoveAllAssignmentByUser(userId interface{}) error {
	ctx := context.Background()
	iter := rbac.rdb.SScan(ctx, rbac.assigmentUserKey(userId), 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		name := iter.Val()
		if err := rbac.rdb.SRem(ctx, rbac.assigmentNameKey(name), userId).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return rbac.rdb.Del(ctx, rbac.assigmentUserKey(userId)).Err()
}

func (rbac *RedisRbac) RemoveAllAssignments() error {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, key+"-*", 0).Iterator()
	for iter.Next(ctx) {
		if err := rbac.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
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
	if err := iter.Err(); err != nil {
		return nil, err
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
	if err := iter.Err(); err != nil {
		return nil, err
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
		items, err := rbac.GetAssignmentsByItem(name)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, items...)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return assignments, nil
}

func (rbac *RedisRbac) findItemsByUser(userId interface{}, t int32) ([]gorbac.Item, error) {
	ctx := context.Background()
	assigmentUserKey := rbac.assigmentUserKey(userId)
	itemFields := rbac.rdb.SMembers(ctx, assigmentUserKey).Val()

	items := make([]gorbac.Item, 0)
	if len(itemFields) > 0 {
		itemKey := rbac.key(gorbac.GetTableName("item"))
		values := rbac.rdb.HMGet(ctx, itemKey, itemFields...).Val()
		for _, value := range values {
			authItem := AuthItem{}
			if value == nil {
				continue
			}
			str, ok := value.(string)
			if !ok {
				continue
			}
			if err := authItem.UnmarshalBinaryStr(str); err == nil {
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
		if value == nil {
			continue
		}
		str, ok := value.(string)
		if !ok {
			continue
		}
		if err := authItem.UnmarshalBinaryStr(str); err == nil {
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
	if err := rbac.cleanItems(); err != nil {
		return err
	}
	return rbac.cleanRules()
}

func (rbac *RedisRbac) cleanItems() error {
	var firstErr error
	rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
		if firstErr != nil {
			return
		}
		firstErr = rbac.RemoveItem(authItem.Name)
	})
	return firstErr
}

func (rbac *RedisRbac) cleanRules() error {
	ctx := context.Background()
	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	return rbac.rdb.Del(ctx, ruleKey).Err()
}

func (rbac *RedisRbac) RemoveAllRules() error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	var firstErr error
	rbac.scanFilterItems(gorbac.NoneType.Value(), func(authItem AuthItem) {
		if firstErr != nil {
			return
		}
		authItem.RuleName = ""
		authItem.UpdateTime = time.Now()
		firstErr = rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem).Err()
	})
	if firstErr != nil {
		return firstErr
	}
	return rbac.cleanRules()
}
