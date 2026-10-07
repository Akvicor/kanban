package listrule

import (
	"slices"
	"testing"
	"time"
)

// 标签 ID：运行中、阻塞、已完成。
const (
	running  int64 = 1
	blocked  int64 = 2
	finished int64 = 3
	other    int64 = 9
)

// 列表操作示例中的三个列表。
var (
	todo  = &Rules{}
	doing = &Rules{
		Enter: OpRules{AddLabels: []int64{running}, Start: TimeSet},
		Exit:  OpRules{RemoveLabels: []int64{running}, AddLabels: []int64{blocked}},
	}
	done = &Rules{
		Enter: OpRules{RemoveLabels: []int64{running, blocked}, AddLabels: []int64{finished}, Complete: TimeSet},
		Exit:  OpRules{Complete: TimeClear},
	}
)

var (
	earlier = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	now     = time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
)

func at(t time.Time) *time.Time { return &t }

func assertCard(t *testing.T, got Card, labels []int64, started, completed *time.Time) {
	t.Helper()
	if !slices.Equal(got.Labels, labels) {
		t.Fatalf("标签 = %v，期望 %v", got.Labels, labels)
	}
	if !sameTime(got.StartedAt, started) || !sameTime(got.CompletedAt, completed) {
		t.Fatalf("开始 = %v，完成 = %v；期望 %v、%v", got.StartedAt, got.CompletedAt, started, completed)
	}
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func TestDesignExamples(t *testing.T) {
	card := Card{Labels: []int64{other}}

	t.Run("在待办中创建，或在待办移入、移出：标签和时间都不变", func(t *testing.T) {
		assertCard(t, todo.Create.Apply(card, now), []int64{other}, nil, nil)
		assertCard(t, Move(card, todo, todo, now), []int64{other}, nil, nil)
	})

	t.Run("从进行中移到待办：去掉 running，加上 blocked，开始时间不变", func(t *testing.T) {
		in := Card{Labels: []int64{running, other}, StartedAt: at(earlier)}
		assertCard(t, Move(in, doing, todo, now), []int64{blocked, other}, at(earlier), nil)
	})

	t.Run("从进行中移到完成：最终标签只有 finished，完成时间为空时被设置", func(t *testing.T) {
		in := Card{Labels: []int64{running}, StartedAt: at(earlier)}
		assertCard(t, Move(in, doing, done, now), []int64{finished}, at(earlier), at(now))
	})

	t.Run("从完成拖回进行中：完成时间被清空，开始时间已有值时保留", func(t *testing.T) {
		in := Card{Labels: []int64{finished}, StartedAt: at(earlier), CompletedAt: at(earlier)}
		assertCard(t, Move(in, done, doing, now), []int64{running, finished}, at(earlier), nil)
	})

	t.Run("直接在进行中创建：不执行移入配置", func(t *testing.T) {
		assertCard(t, doing.Create.Apply(card, now), []int64{other}, nil, nil)
	})

	t.Run("移入进行中时开始时间为空则设置", func(t *testing.T) {
		assertCard(t, Move(card, todo, doing, now), []int64{running, other}, at(now), nil)
	})
}

func TestTimeActions(t *testing.T) {
	cases := []struct {
		action  TimeAction
		current *time.Time
		want    *time.Time
	}{
		{TimeNone, nil, nil},
		{TimeNone, at(earlier), at(earlier)},
		{TimeSet, nil, at(now)},
		{TimeSet, at(earlier), at(earlier)},
		{TimeUpdate, nil, at(now)},
		{TimeUpdate, at(earlier), at(now)},
		{TimeClear, at(earlier), nil},
	}
	for _, c := range cases {
		got := OpRules{Start: c.action}.Apply(Card{StartedAt: c.current}, now)
		if !sameTime(got.StartedAt, c.want) {
			t.Fatalf("动作 %q、原值 %v：得到 %v，期望 %v", c.action, c.current, got.StartedAt, c.want)
		}
	}
}

func TestLabelOrder(t *testing.T) {
	// 同一操作中同一标签同时增加和删除：先删后加，最终保留；已有的标签不重复增加；不存在的标签删除无影响。
	rules := OpRules{AddLabels: []int64{running, blocked}, RemoveLabels: []int64{running, finished}}
	assertCard(t, rules.Apply(Card{Labels: []int64{blocked}}, now), []int64{running, blocked}, nil, nil)
}

func TestMoveUsesOneTimestamp(t *testing.T) {
	// 移出更新开始时间，移入设置完成时间：两个字段写的是同一个时间。
	from := &Rules{Exit: OpRules{Start: TimeUpdate}}
	to := &Rules{Enter: OpRules{Complete: TimeSet}}
	got := Move(Card{}, from, to, now)
	if got.StartedAt == nil || got.CompletedAt == nil || !got.StartedAt.Equal(*got.CompletedAt) {
		t.Fatalf("两步使用的时间不同: %v %v", got.StartedAt, got.CompletedAt)
	}
	// 来源列清空的时间，目标列可以重新设置。
	from = &Rules{Exit: OpRules{Complete: TimeClear}}
	to = &Rules{Enter: OpRules{Complete: TimeSet}}
	got = Move(Card{CompletedAt: at(earlier)}, from, to, now)
	assertCard(t, got, nil, nil, at(now))
}
