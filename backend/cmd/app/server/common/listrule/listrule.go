// Package listrule 定义列表的操作配置，并计算卡片在创建、移入、移出列表时标签和时间的变化。
//
// 每个列表分别配置三种操作：创建（在该列表中新建卡片）、移入（从同一面板的另一列进入）、移出（离开该列表进入另一列）。
// 每种操作可以增加或删除若干标签，并分别决定开始时间和完成时间的动作。没有配置的部分不改动卡片。
package listrule

import (
	"slices"
	"time"
)

// Op 是卡片相对列表的操作。
type Op string

const (
	OpCreate Op = "create"
	OpEnter  Op = "enter"
	OpExit   Op = "exit"
)

// Ops 是全部操作，顺序即界面中的展示顺序。
var Ops = []Op{OpCreate, OpEnter, OpExit}

// Valid 判断操作取值是否合法。
func (o Op) Valid() bool {
	return slices.Contains(Ops, o)
}

// TimeAction 是一次操作对某个时间字段的动作。空字符串表示不影响。
type TimeAction string

const (
	TimeNone   TimeAction = ""
	TimeSet    TimeAction = "set"    // 字段为空时写入本次操作时间，已有值则保留
	TimeUpdate TimeAction = "update" // 无论是否已有值，都写成本次操作时间
	TimeClear  TimeAction = "clear"  // 把字段改为空
)

// Valid 判断时间动作取值是否合法。
func (a TimeAction) Valid() bool {
	switch a {
	case TimeNone, TimeSet, TimeUpdate, TimeClear:
		return true
	}
	return false
}

// OpRules 是一种操作的配置。
type OpRules struct {
	AddLabels    []int64    `json:"add_labels"`
	RemoveLabels []int64    `json:"remove_labels"`
	Start        TimeAction `json:"start"`    // 开始时间的动作
	Complete     TimeAction `json:"complete"` // 完成时间的动作
}

// Rules 是一个列表的三种操作配置。
type Rules struct {
	Create OpRules `json:"create"`
	Enter  OpRules `json:"enter"`
	Exit   OpRules `json:"exit"`
}

// For 返回某种操作的配置。
func (r *Rules) For(op Op) *OpRules {
	switch op {
	case OpCreate:
		return &r.Create
	case OpEnter:
		return &r.Enter
	default:
		return &r.Exit
	}
}

// Card 是操作配置会改动的卡片状态：标签、开始时间和完成时间。
type Card struct {
	Labels      []int64
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// Apply 按一种操作的配置改动卡片。先删除标签，再增加标签：同一标签同时出现在增加和删除中时，最终保留。
// 本次被设置或更新的时间字段都写 now。
func (r OpRules) Apply(card Card, now time.Time) Card {
	labels := slices.DeleteFunc(slices.Clone(card.Labels), func(id int64) bool {
		return slices.Contains(r.RemoveLabels, id)
	})
	for _, id := range r.AddLabels {
		if !slices.Contains(labels, id) {
			labels = append(labels, id)
		}
	}
	slices.Sort(labels)
	return Card{
		Labels:      labels,
		StartedAt:   applyTime(card.StartedAt, r.Start, now),
		CompletedAt: applyTime(card.CompletedAt, r.Complete, now),
	}
}

func applyTime(current *time.Time, action TimeAction, now time.Time) *time.Time {
	switch action {
	case TimeSet:
		if current == nil {
			return &now
		}
	case TimeUpdate:
		return &now
	case TimeClear:
		return nil
	}
	return current
}

// Move 计算卡片从 from 列移到 to 列后的状态：先执行来源列的移出配置，再执行目标列的移入配置，两步使用同一个 now。
func Move(card Card, from, to *Rules, now time.Time) Card {
	return to.Enter.Apply(from.Exit.Apply(card, now), now)
}
