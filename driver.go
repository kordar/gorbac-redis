package gorbac_redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/kordar/gorbac"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisRbac struct {
	rdb   redis.UniversalClient
	table string
}

func NewRedisRbac(rdb redis.UniversalClient, tb string) *RedisRbac {
	return &RedisRbac{rdb: rdb, table: tb}
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

func (rbac *RedisRbac) scanItems(f func(authItem AuthItem)) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("item"))
	iter := rbac.rdb.HScan(ctx, key, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		ele := AuthItem{}
		if err := ele.UnmarshalBinaryStr(iter.Val()); err == nil {
			f(ele)
		}
	}
}

func (rbac *RedisRbac) GetItems(t int32) ([]gorbac.Item, error) {
	items := make([]gorbac.Item, 0)
	rbac.scanItems(func(authItem AuthItem) {
		if t == authItem.Type {
			item := ToItem(authItem)
			items = append(items, item)
		}
	})
	return items, nil
}

func (rbac *RedisRbac) FindAllItems() ([]gorbac.Item, error) {
	items := make([]gorbac.Item, 0)
	rbac.scanItems(func(authItem AuthItem) {
		item := ToItem(authItem)
		items = append(items, item)
	})
	return items, nil
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

func (rbac *RedisRbac) RemoveItem(name string) error {
	//ctx := context.Background()
	//
	//// 解除所有父类关联name的元素，并删除子类、父类均为name的key
	//authItemChild := AuthItemChild{}
	//authItemChildKey := rbac.key(authItemChild.TableName())
	//members := rbac.rdb.SMembers(ctx, fmt.Sprintf("%s-c:%s", authItemChildKey, name)).Val()
	//if members != nil && len(members) > 0 {
	//	for _, parent := range members {
	//		rbac.rdb.SRem(ctx, fmt.Sprintf("%s:%s", authItemChildKey, parent), name)
	//	}
	//}
	//rbac.rdb.Del(ctx, fmt.Sprintf("%s-c:%s", authItemChildKey, name))
	//rbac.rdb.Del(ctx, fmt.Sprintf("%s:%s", authItemChildKey, name))
	//
	//// 将所有assignment关联的itemName清除
	//authAssignmentKey := rbac.key(gorbac.GetTableName("assignment"))
	//authAssignmentScanKey := fmt.Sprintf("%s:%s-*", authAssignmentKey, name)
	//iter := rbac.rdb.Scan(ctx, 0, authAssignmentScanKey, 0).Iterator()
	//for iter.Next(ctx) {
	//	authAssignment := AuthAssignment{}
	//	err := rbac.rdb.HGetAll(ctx, iter.Val()).Scan(&authAssignment)
	//	if err == nil {
	//		_ = rbac.RemoveAssignment(authAssignment.UserId, authAssignment.ItemName)
	//	}
	//}
	//if err := iter.Err(); err != nil {
	//	return err
	//}
	//
	//// 移除item
	//_ = rbac.RemoveItem(name)

	return nil
}

func (rbac *RedisRbac) RemoveRule(ruleName string) error {
	ctx := context.Background()
	itemKey := rbac.key(gorbac.GetTableName("item"))
	rbac.scanItems(func(authItem AuthItem) {
		if authItem.RuleName == ruleName {
			authItem.RuleName = ""
			authItem.UpdateTime = time.Now()
			rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
		}
	})

	ruleKey := rbac.key(gorbac.GetTableName("rule"))
	rbac.rdb.HDel(ctx, ruleKey, ruleName)
	return nil
}

//
//func (rbac *RedisRbac) UpdateItem(itemName string, updateItem db.AuthItem) error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		if itemName != updateItem.Name {
//			child := db.AuthItemChild{}
//			assignment := db.AuthAssignment{}
//			tx.Model(&child).Where("parent = ?", itemName).Update("parent", updateItem.Name)
//			tx.Model(&child).Where("child = ?", itemName).Update("child", updateItem.Name)
//			tx.Model(&assignment).Where("item_name = ?", itemName).Update("item_name", updateItem.Name)
//		}
//		authItem := db.AuthItem{}
//		return tx.Model(&authItem).Where("name = ?", itemName).Omit("create_at").Updates(&updateItem).Error
//	})
//}
//
func (rbac *RedisRbac) UpdateRule(ruleName string, updateRule gorbac.Rule) error {
	ctx := context.Background()
	if ruleName != updateRule.Name {
		itemKey := rbac.key(gorbac.GetTableName("item"))
		rbac.scanItems(func(authItem AuthItem) {
			if authItem.RuleName == ruleName {
				authItem.RuleName = updateRule.Name
				authItem.UpdateTime = time.Now()
				rbac.rdb.HSet(ctx, itemKey, authItem.Name, &authItem)
			}
		})
	}

	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		ruleKey := rbac.key(gorbac.GetTableName("rule"))
		pipe.HDel(ctx, ruleKey, ruleName)
		authRule := ToAuthRule(updateRule)
		pipe.HSet(ctx, ruleKey, authRule.Name, &authRule)
		return nil
	})

	return err
}

//
//// FindRolesByUser 通过会员id获取关联的所有角色
//func (rbac *RedisRbac) FindRolesByUser(userId interface{}) ([]*db.AuthItem, error) {
//	assignment := db.AuthAssignment{}
//	var items []*db.AuthItem
//	err := rbac.db.Model(&assignment).
//		Joins(fmt.Sprintf("inner join %s on %s.item_name = %s.name", db.GetTableName("item"), db.GetTableName("assignment"), db.GetTableName("item"))).
//		Where(fmt.Sprintf("%s.user_id = ? and %s.`type` = 1", db.GetTableName("assignment"), db.GetTableName("item")), userId).
//		Find(&items).Error
//	return items, err
//}
//
//func (rbac *RedisRbac) FindChildrenList() ([]*db.AuthItemChild, error) {
//	var children []*db.AuthItemChild
//	err := rbac.db.Find(&children).Error
//	return children, err
//}
//
//func (rbac *RedisRbac) FindChildrenFormChild(child string) ([]*db.AuthItemChild, error) {
//	var children []*db.AuthItemChild
//	err := rbac.db.Where("child = ?", child).Find(&children).Error
//	return children, err
//}
//
//func (rbac *RedisRbac) GetItemList(t int32, names []string) ([]*db.AuthItem, error) {
//	var items []*db.AuthItem
//	if len(names) > 0 {
//		err := rbac.db.Where("type = ? and name in ?", t, names).Find(&items).Error
//		return items, err
//	} else {
//		err := rbac.db.Where("type = ?", t).Find(&items).Error
//		return items, err
//	}
//}
//
//func (rbac *RedisRbac) FindPermissionsByUser(userId interface{}) ([]*db.AuthItem, error) {
//	assignment := db.AuthAssignment{}
//	var items []*db.AuthItem
//	err := rbac.db.Model(&assignment).
//		Joins(fmt.Sprintf("inner join %s on %s.item_name = %s.name", db.GetTableName("item"), db.GetTableName("assignment"), db.GetTableName("item"))).
//		Where(fmt.Sprintf("%s.user_id = ? and %s.type = 2", db.GetTableName("assignment"), db.GetTableName("item")), userId).
//		Find(&items).Error
//	return items, err
//}
//
//func (rbac *RedisRbac) FindAssignmentByUser(userId interface{}) ([]*db.AuthAssignment, error) {
//	var assignments []*db.AuthAssignment
//	err := rbac.db.Where("user_id = ?", userId).Find(&assignments).Error
//	return assignments, err
//}

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
		rbac.scanItems(func(authItem AuthItem) {
			if itemExits[authItem.Name] {
				item := ToItem(authItem)
				items = append(items, item)
			}
		})
	}

	return items, nil
}

//func (rbac *RedisRbac) scanAssignment(match string, f func(authAssignment AuthAssignment)) {
//	ctx := context.Background()
//	key := rbac.key(gorbac.GetTableName("assignment"))
//	iter := rbac.rdb.HScan(ctx, key, 0, match, 0).Iterator()
//	for iter.Next(ctx) {
//		ele := AuthAssignment{}
//		if err := ele.UnmarshalBinaryStr(iter.Val()); err == nil {
//			f(ele)
//		}
//	}
//}
//
//func (rbac *RedisRbac) assignmentKey(userId interface{}, itemName string) string {
//	return fmt.Sprintf("%v::%s", userId, itemName)
//}

func (rbac *RedisRbac) Assign(assignment gorbac.Assignment) error {
	ctx := context.Background()
	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		authAssignment := ToAuthAssignment(assignment)
		key := rbac.key(authAssignment.TableName())
		pipe.HSet(ctx, fmt.Sprintf("%s:%v", key, assignment.ItemName), assignment.UserId, &assignment)
		pipe.SAdd(ctx, fmt.Sprintf("%s-u:%v", key, assignment.UserId), assignment.ItemName)
		return nil
	})

	return err
}

func (rbac *RedisRbac) RemoveAssignment(userId interface{}, name string) error {
	ctx := context.Background()
	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		key := rbac.key(gorbac.GetTableName("assignment"))
		pipe.HDel(ctx, fmt.Sprintf("%s:%s", key, name), fmt.Sprintf("%v", userId))
		pipe.SRem(ctx, fmt.Sprintf("%s-u:%v", key, userId), name)
		return nil
	})

	return err
}

func (rbac *RedisRbac) RemoveAllAssignmentByUser(userId interface{}) error {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	names := rbac.rdb.SMembers(ctx, fmt.Sprintf("%s-u:%v", key, userId))
	if names.Err() != nil {
		return names.Err()
	}
	for _, name := range names.Val() {
		rbac.rdb.HDel(ctx, fmt.Sprintf("%s:%s", key, name), fmt.Sprintf("%v", userId))
	}
	return nil
}

func (rbac *RedisRbac) RemoveAllAssignments() error {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, key+"*", 0).Iterator()
	for iter.Next(ctx) {
		rbac.rdb.Del(ctx, iter.Val())
	}
	return iter.Err()
}

func (rbac *RedisRbac) GetAssignment(userId interface{}, name string) (*gorbac.Assignment, error) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	if rbac.rdb.Exists(ctx, fmt.Sprintf("%s:%s", key, name)).Val() == 0 {
		return nil, errors.New("assignment not found")
	}
	authAssignment := AuthAssignment{}
	err := rbac.rdb.HGet(ctx, fmt.Sprintf("%s:%s", key, name), fmt.Sprintf("%v", userId)).Scan(&authAssignment)
	if err != nil {
		return nil, err
	} else {
		assignment := ToAssignment(authAssignment)
		return assignment, nil
	}
}

func (rbac *RedisRbac) GetAssignmentByItems(name string) ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, fmt.Sprintf("%s:%s-*", key, name), 0).Iterator()
	assignments := make([]*gorbac.Assignment, 0)
	for iter.Next(ctx) {
		authAssignment := AuthAssignment{}
		err := rbac.rdb.HGetAll(ctx, iter.Val()).Scan(&authAssignment)
		if err == nil {
			assignment := ToAssignment(authAssignment)
			assignments = append(assignments, assignment)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, err
	} else {
		return assignments, nil
	}
}

func (rbac *RedisRbac) GetAssignments(userId interface{}) ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	members := rbac.rdb.SMembers(ctx, fmt.Sprintf("%s-u:%v", key, userId))
	if members.Err() != nil {
		return nil, members.Err()
	}

	assignments := make([]*gorbac.Assignment, 0)
	for _, name := range members.Val() {
		k := fmt.Sprintf("%s:%s-%v", key, name, userId)
		authAssignment := AuthAssignment{}
		err := rbac.rdb.HGetAll(ctx, k).Scan(&authAssignment)
		if err == nil {
			assignment := ToAssignment(authAssignment)
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

func (rbac *RedisRbac) GetAllAssignment() ([]*gorbac.Assignment, error) {
	ctx := context.Background()
	key := rbac.key(gorbac.GetTableName("assignment"))
	iter := rbac.rdb.Scan(ctx, 0, fmt.Sprintf("%s:*", key), 0).Iterator()
	assignments := make([]*gorbac.Assignment, 0)
	for iter.Next(ctx) {
		authAssignment := AuthAssignment{}
		err := rbac.rdb.HGetAll(ctx, iter.Val()).Scan(&authAssignment)
		if err == nil {
			assignment := ToAssignment(authAssignment)
			assignments = append(assignments, assignment)
		}
	}

	if err := iter.Err(); err == nil {
		return assignments, nil
	} else {
		return nil, err
	}
}

//
//func (rbac *RedisRbac) RemoveAll() error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		var assignment db.AuthAssignment
//		tx.Delete(&assignment)
//		var item db.AuthItem
//		tx.Delete(&item)
//		var rule db.AuthRule
//		tx.Delete(&rule)
//		return nil
//	})
//}
//
//func (rbac *RedisRbac) RemoveChildByNames(key string, names []string) error {
//	if names != nil && len(names) > 0 {
//		var itemChild db.AuthItemChild
//		return rbac.db.Where(key+" in (?)", names).Delete(&itemChild).Error
//	}
//	return nil
//}
//
//func (rbac *RedisRbac) RemoveAssignmentByName(names []string) error {
//	if names != nil && len(names) > 0 {
//		var assignments db.AuthAssignment
//		return rbac.db.Where("item_name in (?)", names).Delete(&assignments).Error
//	}
//	return nil
//}
//
//func (rbac *RedisRbac) RemoveItemByType(t int32) error {
//	var item db.AuthItem
//	return rbac.db.Where("type = ?", t).Delete(&item).Error
//}
//
//func (rbac *RedisRbac) RemoveAllRules() error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		var item db.AuthItem
//		tx.Model(&item).Update("rule_name", nil)
//		var rule db.AuthRule
//		tx.Delete(&rule)
//		return nil
//	})
//}
