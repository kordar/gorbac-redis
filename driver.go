package gorbac_redis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kordar/gorbac/db"
	"github.com/redis/go-redis/v9"
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

func (rbac *RedisRbac) AddItem(authItem db.AuthItem) error {
	ctx := context.Background()
	_, err := rbac.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		key := rbac.key(authItem.TableName())
		if marshal, err := json.Marshal(&authItem); err == nil {
			pipe.HSet(ctx, key, authItem.Name, string(marshal))
			typeKey := fmt.Sprintf("%s-%d", key, authItem.Type)
			pipe.SAdd(ctx, typeKey, authItem.Name)
			return nil
		} else {
			return err
		}
	})

	return err
}

func (rbac *RedisRbac) GetItem(name string) (*db.AuthItem, error) {
	ctx := context.Background()
	item := db.AuthItem{}
	if str, err := rbac.rdb.HGet(ctx, rbac.key(item.TableName()), name).Bytes(); err == nil {
		err2 := json.Unmarshal(str, &item)
		return &item, err2
	} else {
		return nil, err
	}
}

//
//func (rbac *RedisRbac) GetItems(t int32) ([]*db.AuthItem, error) {
//	var items []*db.AuthItem
//	item := db.AuthItem{}
//	key := rbac.key(item.TableName())
//	typeKey := fmt.Sprintf("%s-%d", key, t)
//	members := rbac.db.SMembers(typeKey)
//	rbac.db.HMGet(key, members.Val()...)
//	//err := rbac.db.Where("type = ?", t).Find(&items).Error
//	return items, err
//}
//
//func (rbac *RedisRbac) FindAllItems() ([]*db.AuthItem, error) {
//	var items []*db.AuthItem
//	err := rbac.db.Find(&items).Error
//	return items, err
//}
//
//func (rbac *RedisRbac) AddRule(rule db.AuthRule) error {
//	return rbac.db.Create(&rule).Error
//}
//
//func (rbac *RedisRbac) GetRule(name string) (*db.AuthRule, error) {
//	var rule db.AuthRule
//	err := rbac.db.Where("name = ?", name).First(&rule).Error
//	return &rule, err
//}
//
//func (rbac *RedisRbac) GetRules() ([]*db.AuthRule, error) {
//	var rules []*db.AuthRule
//	err := rbac.db.Find(&rules).Error
//	return rules, err
//}
//
//func (rbac *RedisRbac) RemoveItem(name string) error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		itemChild := db.AuthItemChild{}
//		tx.Where("parent = ? or child = ?", name, name).Delete(&itemChild)
//		assignment := db.AuthAssignment{}
//		tx.Where("item_name = ?", name).Delete(&assignment)
//		item := db.AuthItem{}
//		tx.Where("name = ?", name).Delete(&item)
//		return nil
//	})
//}
//
//func (rbac *RedisRbac) RemoveRule(ruleName string) error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		item := db.AuthItem{}
//		tx.Model(&item).Where("rule_name = ?", ruleName).Update("rule_name", nil)
//		rule := db.AuthRule{}
//		tx.Where("name = ?", ruleName).Delete(&rule)
//		return nil
//	})
//}
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
//func (rbac *RedisRbac) UpdateRule(ruleName string, updateRule db.AuthRule) error {
//	return rbac.db.Transaction(func(tx *gorm.DB) error {
//		if ruleName != updateRule.Name {
//			item := db.AuthItem{}
//			tx.Model(&item).Where("rule_name = ?", ruleName).Update("rule_name", updateRule.Name)
//		}
//		rule := db.AuthRule{}
//		return tx.Model(&rule).Where("name = ?", ruleName).Omit("create_at").Updates(&updateRule).Error
//	})
//}
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
//
//func (rbac *RedisRbac) AddItemChild(itemChild db.AuthItemChild) error {
//	return rbac.db.Create(&itemChild).Error
//}
//
//func (rbac *RedisRbac) RemoveChild(parent string, child string) error {
//	var itemChild db.AuthItemChild
//	return rbac.db.Where("parent = ? and child = ?", parent, child).Delete(&itemChild).Error
//}
//
//func (rbac *RedisRbac) RemoveChildren(parent string) error {
//	var itemChild db.AuthItemChild
//	return rbac.db.Where("parent = ?", parent).Delete(&itemChild).Error
//}
//
//func (rbac *RedisRbac) HasChild(parent string, child string) bool {
//	var itemChild db.AuthItemChild
//	first := rbac.db.Model(&itemChild).Where("parent = ? and child = ?", parent, child).First(&itemChild)
//	return first.Error == nil
//}
//
//func (rbac *RedisRbac) FindChildren(name string) ([]*db.AuthItem, error) {
//	var items []*db.AuthItem
//	item := db.AuthItem{}
//	err := rbac.db.Model(&item).
//		Joins(fmt.Sprintf("inner join %s on %s.name = %s.child", db.GetTableName("item-child"), db.GetTableName("item"), db.GetTableName("item-child"))).
//		Where(fmt.Sprintf("%s.parent = ?", db.GetTableName("item-child")), name).Error
//	return items, err
//}
//
//func (rbac *RedisRbac) Assign(assignment db.AuthAssignment) error {
//	return rbac.db.Create(&assignment).Error
//}
//
//func (rbac *RedisRbac) RemoveAssignment(userId interface{}, name string) error {
//	var assignment db.AuthAssignment
//	return rbac.db.Where("user_id = ? and item_name = ?", userId, name).Delete(&assignment).Error
//}
//
//func (rbac *RedisRbac) RemoveAllAssignmentByUser(userId interface{}) error {
//	var assignment db.AuthAssignment
//	return rbac.db.Where("user_id = ?", userId).Delete(&assignment).Error
//}
//
//func (rbac *RedisRbac) RemoveAllAssignments() error {
//	var assignment db.AuthAssignment
//	return rbac.db.Delete(&assignment).Error
//}
//
//func (rbac *RedisRbac) GetAssignment(userId interface{}, name string) (*db.AuthAssignment, error) {
//	var assignments *db.AuthAssignment
//	err := rbac.db.Where("user_id = ? and item_name = ?", userId, name).First(assignments).Error
//	return assignments, err
//}
//
//func (rbac *RedisRbac) GetAssignmentByItems(name string) ([]*db.AuthAssignment, error) {
//	var assignments []*db.AuthAssignment
//	err := rbac.db.Where("item_name = ?", name).Find(&assignments).Error
//	return assignments, err
//}
//
//func (rbac *RedisRbac) GetAssignments(userId interface{}) ([]*db.AuthAssignment, error) {
//	var assignments []*db.AuthAssignment
//	err := rbac.db.Where("user_id = ?", userId).Find(&assignments).Error
//	return assignments, err
//}
//
//func (rbac *RedisRbac) GetAllAssignment() ([]*db.AuthAssignment, error) {
//	var assignments []*db.AuthAssignment
//	err := rbac.db.Find(&assignments).Error
//	return assignments, err
//}
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
