package gorbac_redis

import (
	"encoding/json"
	"github.com/kordar/gorbac"
	"time"
)

// AuthRule 规则绑定，实现Execute接口完成特殊权限校验功能
type AuthRule struct {
	Name        string    `redis:"name"`
	ExecuteName string    `redis:"execute_name"`
	CreateTime  time.Time `redis:"create_time"`
	UpdateTime  time.Time `redis:"update_time"`
}

func (t *AuthRule) MarshalBinary() (data []byte, err error) {
	return json.Marshal(t)
}

func (t *AuthRule) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t *AuthRule) UnmarshalBinaryStr(data string) error {
	return json.Unmarshal([]byte(data), t)
}

func (t *AuthRule) TableName() string {
	return gorbac.GetTableName("rule")
}

// AuthItem 权限节点
type AuthItem struct {
	Name        string    `redis:"name"`
	Type        int32     `redis:"type"`
	Description string    `redis:"description"`
	RuleName    string    `redis:"rule_name"`
	ExecuteName string    `redis:"execute_name"`
	CreateTime  time.Time `redis:"create_time"`
	UpdateTime  time.Time `redis:"update_time"`
}

func (t *AuthItem) MarshalBinary() (data []byte, err error) {
	return json.Marshal(t)
}

func (t *AuthItem) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t *AuthItem) UnmarshalBinaryStr(data string) error {
	return json.Unmarshal([]byte(data), t)
}

func (t *AuthItem) TableName() string {
	return gorbac.GetTableName("item")
}

// AuthItemChild 权限赋值关系
type AuthItemChild struct {
	Parent string `redis:"parent"`
	Child  string `redis:"child"`
}

func (t *AuthItemChild) TableName() string {
	return gorbac.GetTableName("item-child")
}

func (t *AuthItemChild) MarshalBinary() (data []byte, err error) {
	return json.Marshal(t)
}

func (t *AuthItemChild) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t *AuthItemChild) UnmarshalBinaryStr(data string) error {
	return json.Unmarshal([]byte(data), t)
}

// AuthAssignment 用户赋权，userId->关联权限
type AuthAssignment struct {
	ItemName   string    `redis:"item_name"`
	UserId     string    `redis:"user_id"`
	CreateTime time.Time `redis:"create_time"`
}

func (t *AuthAssignment) TableName() string {
	return gorbac.GetTableName("assignment")
}

func (t *AuthAssignment) MarshalBinary() (data []byte, err error) {
	return json.Marshal(t)
}

func (t *AuthAssignment) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t *AuthAssignment) UnmarshalBinaryStr(data string) error {
	return json.Unmarshal([]byte(data), t)
}
