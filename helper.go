package gorbac_redis

import (
	"github.com/kordar/gorbac"
	"github.com/spf13/cast"
)

func ToAuthItem(item gorbac.Item) AuthItem {
	return AuthItem{
		Name:        item.GetName(),
		Type:        item.GetType().Value(),
		Description: item.GetDescription(),
		RuleName:    item.GetRuleName(),
		ExecuteName: item.GetExecuteName(),
		CreateTime:  item.GetCreateTime(),
		UpdateTime:  item.GetUpdateTime(),
	}
}

// ToItem AuthItem转item对象
func ToItem(authItem AuthItem) gorbac.Item {
	if gorbac.RoleType.Value() == authItem.Type {
		return gorbac.NewRole(authItem.Name, authItem.Description, authItem.RuleName, authItem.ExecuteName, authItem.CreateTime, authItem.UpdateTime)
	} else {
		return gorbac.NewPermission(authItem.Name, authItem.Description, authItem.RuleName, authItem.ExecuteName, authItem.CreateTime, authItem.UpdateTime)
	}
}

func ToItems(authItems []AuthItem) []gorbac.Item {
	items := make([]gorbac.Item, 0)
	for _, authItem := range authItems {
		item := ToItem(authItem)
		items = append(items, item)
	}
	return items
}

func ToAuthRule(rule gorbac.Rule) AuthRule {
	return AuthRule{
		Name:       rule.Name,
		CreateTime: rule.CreateTime,
		UpdateTime: rule.UpdateTime,
	}
}

func ToRule(rule AuthRule) *gorbac.Rule {
	return gorbac.NewRule(rule.Name, rule.ExecuteName, rule.CreateTime, rule.UpdateTime)
}

func ToRules(authRules []AuthRule) []*gorbac.Rule {
	rules := make([]*gorbac.Rule, 0)
	for _, authRule := range authRules {
		rule := ToRule(authRule)
		rules = append(rules, rule)
	}
	return rules
}

func ToAuthItemChild(parent string, child string) AuthItemChild {
	return AuthItemChild{Parent: parent, Child: child}
}

func ToItemChild(authItemChild AuthItemChild) *gorbac.ItemChild {
	return gorbac.NewItemChild(authItemChild.Parent, authItemChild.Child)
}

func ToItemChildren(authItemChildren []AuthItemChild) []*gorbac.ItemChild {
	itemChildren := make([]*gorbac.ItemChild, 0)
	for _, authItemChild := range authItemChildren {
		itemChild := ToItemChild(authItemChild)
		itemChildren = append(itemChildren, itemChild)
	}
	return itemChildren
}

func ToAssignment(authAssignment AuthAssignment) *gorbac.Assignment {
	return gorbac.NewAssignment(authAssignment.UserId, authAssignment.ItemName)
}

func ToAssignments(authAssignments []AuthAssignment) []*gorbac.Assignment {
	assignments := make([]*gorbac.Assignment, 0)
	for _, authAssignment := range authAssignments {
		assignment := ToAssignment(authAssignment)
		assignments = append(assignments, assignment)
	}
	return assignments
}

func ToAuthAssignment(assignment gorbac.Assignment) AuthAssignment {
	return AuthAssignment{
		ItemName:   assignment.ItemName,
		UserId:     cast.ToString(assignment.UserId),
		CreateTime: assignment.CreateTime,
	}
}
