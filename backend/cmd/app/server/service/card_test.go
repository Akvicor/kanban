package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"slices"
	"testing"
	"time"
)

// cardEnv 是一个用户、一个面板和其中的列表，供卡片测试使用。
type cardEnv struct {
	t      *testing.T
	ctx    context.Context
	user   *model.User
	panel  *model.Panel
	lists  map[string]*model.List
	labels map[string]*model.Label
}

func newCardEnv(t *testing.T, lists ...string) *cardEnv {
	t.Helper()
	mustAdmin(t)
	user := mustUser(t, "alice")
	env := &cardEnv{t: t, ctx: context.Background(), user: user, lists: map[string]*model.List{}, labels: map[string]*model.Label{}}
	env.panel = firstPanel(t, user.ID, mustBoard(t, user.ID, "工作").ID)
	for _, name := range lists {
		env.lists[name] = mustList(t, user.ID, env.panel.ID, name)
	}
	return env
}

func (e *cardEnv) label(name string) int64 {
	e.t.Helper()
	if label, ok := e.labels[name]; ok {
		return label.ID
	}
	label, err := Label.Create(e.ctx, e.user.ID, e.panel.ID, name, "#3E63DD")
	if err != nil {
		e.t.Fatal(err)
	}
	e.labels[name] = label
	return label.ID
}

func (e *cardEnv) rules(list string, rules listrule.Rules) {
	e.t.Helper()
	if _, err := List.UpdateRules(e.ctx, e.user.ID, e.lists[list].ID, rules); err != nil {
		e.t.Fatal(err)
	}
}

func (e *cardEnv) create(list, title string, button listsort.End) *model.Card {
	e.t.Helper()
	card, err := Card.Create(e.ctx, e.user.ID, e.lists[list].ID, title, button)
	if err != nil {
		e.t.Fatal(err)
	}
	return card
}

func (e *cardEnv) reload(id int64) *model.Card {
	e.t.Helper()
	card, err := repository.Card.FindByID(e.ctx, e.user.ID, id)
	if err != nil {
		e.t.Fatal(err)
	}
	return card
}

// labelNames 返回卡片上的标签名，按名称排序。
func (e *cardEnv) labelNames(id int64) []string {
	e.t.Helper()
	labels, err := repository.Card.LabelIDs(e.ctx, e.user.ID, []int64{id})
	if err != nil {
		e.t.Fatal(err)
	}
	names := []string{}
	for name, label := range e.labels {
		if slices.Contains(labels[id], label.ID) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// titles 返回列表中卡片的标题，按隐藏序号。
func (e *cardEnv) titles(list string) []string {
	e.t.Helper()
	cards, err := cardsInList(e.ctx, e.user.ID, e.lists[list].ID)
	if err != nil {
		e.t.Fatal(err)
	}
	names := make([]string, 0, len(cards))
	for _, card := range cards {
		names = append(names, card.Title)
	}
	return names
}

func (e *cardEnv) actions(id int64) []string {
	e.t.Helper()
	actions, err := repository.CardAction.ListByCards(e.ctx, e.user.ID, []int64{id})
	if err != nil {
		e.t.Fatal(err)
	}
	types := make([]string, 0, len(actions))
	for _, action := range actions {
		types = append(types, action.Type)
	}
	return types
}

func TestCardCreateUsesButtonEndAndCreateRules(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办")
		e.rules("待办", listrule.Rules{Create: listrule.OpRules{AddLabels: []int64{e.label("新")}, Start: listrule.TimeSet}})

		first := e.create("待办", "一", listsort.Tail)
		e.create("待办", "二", listsort.Tail)
		e.create("待办", "三", listsort.Head)
		if got := e.titles("待办"); !equalNames(got, []string{"三", "一", "二"}) {
			t.Fatalf("顺序 = %v", got)
		}
		// 把列首按钮配成插到列尾。
		list := e.lists["待办"]
		if _, err := List.UpdateSettings(e.ctx, e.user.ID, list.ID, ListSettings{
			SortMode: listsort.Manual, SortDir: listsort.Asc, HeadAdd: listsort.Tail, TailAdd: listsort.Tail, ShowAge: true,
		}); err != nil {
			t.Fatal(err)
		}
		e.create("待办", "四", listsort.Head)
		if got := e.titles("待办"); got[len(got)-1] != "四" {
			t.Fatalf("列首按钮配成列尾后 = %v", got)
		}

		card := e.reload(first.ID)
		if !equalNames(e.labelNames(card.ID), []string{"新"}) || card.StartedAt == nil || card.CompletedAt != nil {
			t.Fatalf("创建配置未执行: 标签 %v，开始 %v", e.labelNames(card.ID), card.StartedAt)
		}
		if card.StartedAt.Sub(card.CreatedAt) != 0 {
			t.Fatal("创建时间和本次写入的开始时间不是同一个时刻")
		}
		if got := e.actions(card.ID); !equalNames(got, []string{model.ActionCreate}) {
			t.Fatalf("操作记录 = %v", got)
		}
		if _, err := Card.Create(e.ctx, e.user.ID, list.ID, "  ", listsort.Tail); errorKind(err) != KindBadRequest {
			t.Fatalf("空标题 err = %v", err)
		}
	})
}

func TestCardMoveRunsExitThenEnter(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办", "进行中", "完成")
		running, blocked, finished := e.label("running"), e.label("blocked"), e.label("finished")
		e.rules("进行中", listrule.Rules{
			Enter: listrule.OpRules{AddLabels: []int64{running}, Start: listrule.TimeSet},
			Exit:  listrule.OpRules{RemoveLabels: []int64{running}, AddLabels: []int64{blocked}},
		})
		e.rules("完成", listrule.Rules{
			Enter: listrule.OpRules{RemoveLabels: []int64{running, blocked}, AddLabels: []int64{finished}, Complete: listrule.TimeSet},
			Exit:  listrule.OpRules{Complete: listrule.TimeClear},
		})
		card := e.create("待办", "任务", listsort.Tail)
		move := func(list string, index *int) *model.Card {
			t.Helper()
			moved, err := Card.Move(e.ctx, e.user.ID, card.ID, e.lists[list].ID, index)
			if err != nil {
				t.Fatal(err)
			}
			return e.reload(moved.ID)
		}

		got := move("进行中", nil)
		if !equalNames(e.labelNames(card.ID), []string{"running"}) || got.StartedAt == nil {
			t.Fatalf("移入进行中: %v %v", e.labelNames(card.ID), got.StartedAt)
		}
		started := *got.StartedAt

		got = move("完成", nil)
		if !equalNames(e.labelNames(card.ID), []string{"finished"}) || got.CompletedAt == nil || !got.StartedAt.Equal(started) {
			t.Fatalf("进行中到完成: %v %+v", e.labelNames(card.ID), got)
		}

		got = move("进行中", nil)
		if got.CompletedAt != nil || !got.StartedAt.Equal(started) {
			t.Fatalf("从完成拖回进行中: %+v", got)
		}

		got = move("待办", nil)
		if !equalNames(e.labelNames(card.ID), []string{"blocked", "finished"}) {
			t.Fatalf("进行中到待办: %v", e.labelNames(card.ID))
		}
		if n := len(e.actions(card.ID)); n != 5 {
			t.Fatalf("操作记录 %d 条，应为 1 条创建和 4 条移动", n)
		}

		// 同列调整顺序只改序号，不执行操作配置，也不记操作。
		e.create("待办", "另一张", listsort.Tail)
		before := e.labelNames(card.ID)
		move("待办", nil)
		if got := e.titles("待办"); !equalNames(got, []string{"另一张", "任务"}) || !equalNames(e.labelNames(card.ID), before) {
			t.Fatalf("同列调整: %v %v", got, e.labelNames(card.ID))
		}
		if n := len(e.actions(card.ID)); n != 5 {
			t.Fatalf("同列调整记了操作: %d 条", n)
		}
	})
}

func TestCardTitleConflict(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办")
		card := e.create("待办", "原标题", listsort.Tail)
		user, _ := User.FindByID(e.ctx, e.user.ID)
		base := user.Revision // 两台设备都从这个序号开始编辑

		if _, err := Card.UpdateTitle(e.ctx, e.user.ID, card.ID, "设备一的标题", base); err != nil {
			t.Fatal(err)
		}
		_, err := Card.UpdateTitle(e.ctx, e.user.ID, card.ID, "设备二的标题", base)
		var conflictErr *Error
		if !errors.As(err, &conflictErr) || conflictErr.Kind != KindConflict {
			t.Fatalf("同一字段冲突 err = %v", err)
		}
		// 冲突带上当前卡片和读取时的同步序号，客户端据此显示最新标题。
		if view, ok := conflictErr.Data.(dro.Card); !ok || view.Title != "设备一的标题" || conflictErr.Revision <= base {
			t.Fatalf("冲突附带的数据 = %+v, 序号 = %d", conflictErr.Data, conflictErr.Revision)
		}
		// 不同字段可以合并：标题被改过，从旧序号开始编辑描述仍然成功。
		if _, err := Card.UpdateDescription(e.ctx, e.user.ID, card.ID, "描述", base); err != nil {
			t.Fatalf("修改另一字段 err = %v", err)
		}
		saved := e.reload(card.ID)
		if saved.Title != "设备一的标题" || saved.Description != "描述" {
			t.Fatalf("保存的卡片 = %+v", saved)
		}
		// 以冲突返回的序号重新提交（用户选择覆盖）可以成功。
		if _, err := Card.UpdateTitle(e.ctx, e.user.ID, card.ID, "设备二的标题", conflictErr.Revision); err != nil {
			t.Fatal(err)
		}
		if saved = e.reload(card.ID); saved.Title != "设备二的标题" {
			t.Fatalf("覆盖后的标题 = %q", saved.Title)
		}
	})
}

func TestCardTimerAndArchive(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办", "完成")
		e.rules("完成", listrule.Rules{Enter: listrule.OpRules{AddLabels: []int64{e.label("finished")}}})
		card := e.create("待办", "计时", listsort.Tail)
		if _, err := Card.Timer(e.ctx, e.user.ID, card.ID, TimerSet, 100); err != nil {
			t.Fatal(err)
		}
		if _, err := Card.Timer(e.ctx, e.user.ID, card.ID, TimerStart, 0); err != nil {
			t.Fatal(err)
		}
		// 让定时器已经走了 30 秒。
		started := nowUTC().Add(-30 * time.Second)
		if err := repository.Card.Update(e.ctx, e.user.ID, card.ID, map[string]any{"timer_started_at": started}); err != nil {
			t.Fatal(err)
		}

		// 归档停止定时器，卡片离开列表，不能再修改。
		if err := Card.Archive(e.ctx, e.user.ID, card.ID); err != nil {
			t.Fatal(err)
		}
		archived := e.reload(card.ID)
		if archived.TimerStartedAt != nil || archived.TimerSeconds < 130 || archived.ListID != nil || archived.Position != nil {
			t.Fatalf("归档后的卡片 = %+v", archived)
		}
		if _, err := Card.UpdateTitle(e.ctx, e.user.ID, card.ID, "改名", 1<<40); errorKind(err) != KindBadRequest {
			t.Fatalf("修改归档卡片 err = %v", err)
		}

		// 恢复到完成的列首：不执行移入配置，不重新启动定时器，记一条恢复操作。
		e.create("完成", "已有", listsort.Tail)
		if _, err := Card.Restore(e.ctx, e.user.ID, card.ID, e.lists["完成"].ID, listsort.Head); err != nil {
			t.Fatal(err)
		}
		restored := e.reload(card.ID)
		if restored.Archived() || restored.TimerStartedAt != nil || len(e.labelNames(card.ID)) != 0 {
			t.Fatalf("恢复后的卡片 = %+v，标签 %v", restored, e.labelNames(card.ID))
		}
		if got := e.titles("完成"); !equalNames(got, []string{"计时", "已有"}) {
			t.Fatalf("恢复后的顺序 = %v", got)
		}
		if got := e.actions(card.ID); !equalNames(got, []string{model.ActionCreate, model.ActionArchive, model.ActionRestore}) {
			t.Fatalf("操作记录 = %v", got)
		}

		// 列表归档、面板归档也停止其中的定时器。
		if _, err := Card.Timer(e.ctx, e.user.ID, card.ID, TimerStart, 0); err != nil {
			t.Fatal(err)
		}
		if err := List.Archive(e.ctx, e.user.ID, e.lists["完成"].ID); err != nil {
			t.Fatal(err)
		}
		if e.reload(card.ID).TimerStartedAt != nil {
			t.Fatal("列表归档后定时器仍在计时")
		}
	})
}

func TestArchiveAllCardsInList(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "完成")
		for _, title := range []string{"一", "二", "三"} {
			e.create("完成", title, listsort.Tail)
		}
		if err := Card.ArchiveAllInList(e.ctx, e.user.ID, e.lists["完成"].ID); err != nil {
			t.Fatal(err)
		}
		if got := e.titles("完成"); len(got) != 0 {
			t.Fatalf("列表中剩余 %v", got)
		}
		bundle, err := Card.ArchivedInPanel(e.ctx, e.user.ID, e.panel.ID)
		if err != nil || len(bundle.Cards) != 3 {
			t.Fatalf("卡片归档 = %+v, %v", bundle, err)
		}
		if _, err = findList(e.ctx, e.user.ID, e.lists["完成"].ID); err != nil {
			t.Fatal("列表本身被删除")
		}
	})
}

func TestCardCopy(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办", "完成")
		finished := e.label("finished")
		e.label("重要")
		e.rules("完成", listrule.Rules{Create: listrule.OpRules{AddLabels: []int64{finished}, Complete: listrule.TimeSet}})
		source := e.create("待办", "原卡片", listsort.Tail)
		e.create("待办", "下一张", listsort.Tail)
		if _, err := Card.SetLabel(e.ctx, e.user.ID, source.ID, e.labels["重要"].ID, true); err != nil {
			t.Fatal(err)
		}
		due := nowUTC().Add(48 * time.Hour).Truncate(time.Second)
		if _, err := Card.SetDates(e.ctx, e.user.ID, source.ID, CardDates{DueAt: &due, DueNotify: true}); err != nil {
			t.Fatal(err)
		}
		parent, _ := Task.Create(e.ctx, e.user.ID, source.ID, nil, []string{"父任务"})
		children, _ := Task.Create(e.ctx, e.user.ID, source.ID, &parent[0].ID, []string{"子任务"})
		if _, err := Task.SetDone(e.ctx, e.user.ID, children[0].ID, true); err != nil {
			t.Fatal(err)
		}
		if _, err := CardLink.Add(e.ctx, e.user.ID, source.ID, nil, &e.panel.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := Card.Timer(e.ctx, e.user.ID, source.ID, TimerSet, 500); err != nil {
			t.Fatal(err)
		}

		// 菜单复制：插在原卡片下方。
		copied, err := Card.Copy(e.ctx, e.user.ID, source.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := e.titles("待办"); !equalNames(got, []string{"原卡片", "原卡片(副本)", "下一张"}) {
			t.Fatalf("复制后的顺序 = %v", got)
		}
		saved := e.reload(copied.ID)
		if saved.DueAt == nil || !saved.DueAt.Equal(due) || !saved.DueNotify || saved.TimerSeconds != 0 || saved.StartedAt != nil {
			t.Fatalf("副本 = %+v", saved)
		}
		if !equalNames(e.labelNames(copied.ID), []string{"重要"}) {
			t.Fatalf("副本标签 = %v", e.labelNames(copied.ID))
		}
		tasks, _ := repository.Task.ListByCards(e.ctx, e.user.ID, []int64{copied.ID})
		if len(tasks) != 2 {
			t.Fatalf("副本任务 = %+v", tasks)
		}
		for _, task := range tasks {
			if task.Title == "子任务" && (task.ParentID == nil || !task.Done) {
				t.Fatalf("子任务的层级或完成状态未保留: %+v", task)
			}
		}
		links, _ := repository.CardLink.ListByCards(e.ctx, e.user.ID, []int64{copied.ID})
		if len(links) != 1 {
			t.Fatalf("副本关联 = %+v", links)
		}
		if got := e.actions(copied.ID); !equalNames(got, []string{model.ActionCreate}) {
			t.Fatalf("副本操作记录 = %v", got)
		}

		// 粘贴到另一列：插到列首并执行该列的创建配置。
		pasted, err := Card.Copy(e.ctx, e.user.ID, source.ID, &e.lists["完成"].ID)
		if err != nil {
			t.Fatal(err)
		}
		if !equalNames(e.labelNames(pasted.ID), []string{"finished", "重要"}) || e.reload(pasted.ID).CompletedAt == nil {
			t.Fatalf("粘贴的副本未执行创建配置: %v", e.labelNames(pasted.ID))
		}

		// 不能复制到其他面板。
		otherPanel := firstPanel(t, e.user.ID, mustBoard(t, e.user.ID, "生活").ID)
		otherList := mustList(t, e.user.ID, otherPanel.ID, "别处")
		if _, err = Card.Copy(e.ctx, e.user.ID, source.ID, &otherList.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("跨面板复制 err = %v", err)
		}
		if _, err = Card.Move(e.ctx, e.user.ID, source.ID, otherList.ID, nil); errorKind(err) != KindBadRequest {
			t.Fatalf("跨面板移动 err = %v", err)
		}
	})
}

func TestDeletingLabelAndPriorityUpdatesCards(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办")
		card := e.create("待办", "卡片", listsort.Tail)
		urgent := e.label("紧急")
		if _, err := Card.SetLabel(e.ctx, e.user.ID, card.ID, urgent, true); err != nil {
			t.Fatal(err)
		}
		levels, _ := repository.PriorityLevel.ListByPanel(e.ctx, e.user.ID, e.panel.ID)
		if _, err := Card.SetPriority(e.ctx, e.user.ID, card.ID, &levels[0].ID); err != nil {
			t.Fatal(err)
		}

		if err := Label.Delete(e.ctx, e.user.ID, urgent); err != nil {
			t.Fatal(err)
		}
		delete(e.labels, "紧急")
		if err := PriorityLevel.Delete(e.ctx, e.user.ID, levels[0].ID); err != nil {
			t.Fatal(err)
		}
		saved := e.reload(card.ID)
		labels, _ := repository.Card.LabelIDs(e.ctx, e.user.ID, []int64{card.ID})
		if len(labels[card.ID]) != 0 || saved.PriorityLevelID != nil {
			t.Fatalf("删除后卡片标签 %v，优先级 %v", labels[card.ID], saved.PriorityLevelID)
		}

		// 其他面板的挡位和标签不能用在这张卡片上。
		otherPanel := firstPanel(t, e.user.ID, mustBoard(t, e.user.ID, "生活").ID)
		otherLevels, _ := repository.PriorityLevel.ListByPanel(e.ctx, e.user.ID, otherPanel.ID)
		if _, err := Card.SetPriority(e.ctx, e.user.ID, card.ID, &otherLevels[0].ID); errorKind(err) != KindBadRequest {
			t.Fatalf("使用其他面板的挡位 err = %v", err)
		}
	})
}

func TestPanelContentAndRestoreBundles(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办", "完成")
		card := e.create("待办", "卡片", listsort.Tail)
		Task.Create(e.ctx, e.user.ID, card.ID, nil, []string{"任务"})
		archived := e.create("完成", "归档的", listsort.Tail)
		if err := Card.Archive(e.ctx, e.user.ID, archived.ID); err != nil {
			t.Fatal(err)
		}
		e.create("完成", "列表归档里的", listsort.Tail)
		if err := List.Archive(e.ctx, e.user.ID, e.lists["完成"].ID); err != nil {
			t.Fatal(err)
		}

		// 面板内容只包含不在列表归档中的列表里、不在卡片归档中的卡片。
		content, err := List.Content(e.ctx, e.user.ID, e.panel.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(content.Cards) != 1 || content.Cards[0].ID != card.ID || len(content.Tasks) != 1 || len(content.Actions) != 1 {
			t.Fatalf("面板内容 = %d 张卡片，%d 个任务，%d 条操作", len(content.Cards), len(content.Tasks), len(content.Actions))
		}
		bundle, err := Card.InArchivedList(e.ctx, e.user.ID, e.lists["完成"].ID)
		if err != nil || len(bundle.Cards) != 1 || bundle.Cards[0].Title != "列表归档里的" {
			t.Fatalf("列表归档中的卡片 = %+v, %v", bundle, err)
		}
	})
}
