package service

import (
	"context"
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"testing"
)

// firstPanel 返回看板中的第一个面板。
func firstPanel(t *testing.T, userID, boardID int64) *model.Panel {
	t.Helper()
	panels, err := repository.Panel.ListActiveInBoard(context.Background(), userID, boardID)
	if err != nil || len(panels) == 0 {
		t.Fatalf("看板没有面板: %v", err)
	}
	return panels[0]
}

func mustList(t *testing.T, userID, panelID int64, name string) *model.List {
	t.Helper()
	list, err := List.Create(context.Background(), userID, panelID, name)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

// activeListNames 返回面板中未归档列表的名称，从左到右。
func activeListNames(t *testing.T, userID, panelID int64) []string {
	t.Helper()
	lists, err := repository.List.ListActiveByPanel(context.Background(), userID, panelID)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(lists))
	for _, list := range lists {
		names = append(names, list.Name)
	}
	return names
}

func TestListCreateSettingsAndOrder(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		panel := firstPanel(t, alice.ID, mustBoard(t, alice.ID, "工作").ID)

		todo := mustList(t, alice.ID, panel.ID, "待办")
		if !todo.ShowAge || todo.SortMode != listsort.Manual || todo.HeadAdd != listsort.Head || todo.TailAdd != listsort.Tail ||
			todo.RemindOff || todo.DueOff || todo.Color != "" {
			t.Fatalf("新列表的默认设置 = %+v", todo)
		}
		mustList(t, alice.ID, panel.ID, "进行中")
		done := mustList(t, alice.ID, panel.ID, "完成")
		if _, err := List.Reorder(ctx, alice.ID, done.ID, ptr(0)); err != nil {
			t.Fatal(err)
		}
		if got := activeListNames(t, alice.ID, panel.ID); !equalNames(got, []string{"完成", "待办", "进行中"}) {
			t.Fatalf("列表顺序 = %v", got)
		}

		updated, err := List.UpdateSettings(ctx, alice.ID, done.ID, ListSettings{
			Color: "#30a46c", ShowAge: false, SortMode: listsort.DueAt, SortDir: listsort.Desc,
			HeadAdd: listsort.Tail, TailAdd: listsort.Tail, RemindOff: true, DueOff: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Color != "#30A46C" || updated.ShowAge || updated.SortMode != listsort.DueAt || !updated.DueOff {
			t.Fatalf("修改后的设置 = %+v", updated)
		}
		invalid := []ListSettings{
			{Color: "green", SortMode: listsort.Manual, SortDir: listsort.Asc, HeadAdd: listsort.Head, TailAdd: listsort.Tail},
			{SortMode: "random", SortDir: listsort.Asc, HeadAdd: listsort.Head, TailAdd: listsort.Tail},
			{SortMode: listsort.Manual, SortDir: listsort.Asc, HeadAdd: "middle", TailAdd: listsort.Tail},
		}
		for _, settings := range invalid {
			if _, err = List.UpdateSettings(ctx, alice.ID, done.ID, settings); errorKind(err) != KindBadRequest {
				t.Fatalf("UpdateSettings(%+v) err = %v", settings, err)
			}
		}
	})
}

func TestListRules(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		panel := firstPanel(t, alice.ID, mustBoard(t, alice.ID, "工作").ID)
		otherPanel := firstPanel(t, alice.ID, mustBoard(t, alice.ID, "生活").ID)
		running, _ := Label.Create(ctx, alice.ID, panel.ID, "running", "#3E63DD")
		blocked, _ := Label.Create(ctx, alice.ID, panel.ID, "blocked", "#E5484D")
		foreign, _ := Label.Create(ctx, alice.ID, otherPanel.ID, "其他面板", "#000000")
		doing := mustList(t, alice.ID, panel.ID, "进行中")

		rules := listrule.Rules{
			Enter: listrule.OpRules{AddLabels: []int64{running.ID, running.ID}, Start: listrule.TimeSet},
			Exit:  listrule.OpRules{RemoveLabels: []int64{running.ID}, AddLabels: []int64{blocked.ID}},
		}
		if _, err := List.UpdateRules(ctx, alice.ID, doing.ID, rules); err != nil {
			t.Fatal(err)
		}
		saved, err := listRules(ctx, alice.ID, []int64{doing.ID})
		if err != nil {
			t.Fatal(err)
		}
		got := saved[doing.ID]
		if len(got.Enter.AddLabels) != 1 || got.Enter.Start != listrule.TimeSet || len(got.Exit.RemoveLabels) != 1 ||
			len(got.Exit.AddLabels) != 1 || got.Create.Start != listrule.TimeNone {
			t.Fatalf("保存的配置 = %+v", got)
		}

		// 只能使用本面板的标签；时间动作必须合法。
		if _, err = List.UpdateRules(ctx, alice.ID, doing.ID, listrule.Rules{Create: listrule.OpRules{AddLabels: []int64{foreign.ID}}}); errorKind(err) != KindBadRequest {
			t.Fatalf("使用其他面板的标签 err = %v", err)
		}
		if _, err = List.UpdateRules(ctx, alice.ID, doing.ID, listrule.Rules{Exit: listrule.OpRules{Complete: "later"}}); errorKind(err) != KindBadRequest {
			t.Fatalf("无效的时间动作 err = %v", err)
		}

		// 删除标签时，引用它的配置一并删除，其他配置保留。
		if err = Label.Delete(ctx, alice.ID, running.ID); err != nil {
			t.Fatal(err)
		}
		saved, _ = listRules(ctx, alice.ID, []int64{doing.ID})
		got = saved[doing.ID]
		if len(got.Enter.AddLabels) != 0 || len(got.Exit.RemoveLabels) != 0 || len(got.Exit.AddLabels) != 1 || got.Enter.Start != listrule.TimeSet {
			t.Fatalf("删除标签后的配置 = %+v", got)
		}
	})
}

func TestListArchiveAndRestore(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		board := mustBoard(t, alice.ID, "工作")
		panel := firstPanel(t, alice.ID, board.ID)
		todo := mustList(t, alice.ID, panel.ID, "待办")
		mustList(t, alice.ID, panel.ID, "进行中")
		done := mustList(t, alice.ID, panel.ID, "完成")

		if err := List.Archive(ctx, alice.ID, todo.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := List.Rename(ctx, alice.ID, todo.ID, "改名"); errorKind(err) != KindBadRequest {
			t.Fatalf("修改归档中的列表 err = %v", err)
		}
		if err := List.Archive(ctx, alice.ID, done.ID); err != nil {
			t.Fatal(err)
		}
		if got := activeListNames(t, alice.ID, panel.ID); !equalNames(got, []string{"进行中"}) {
			t.Fatalf("归档后的列表 = %v", got)
		}

		// 恢复到最右边和最左边。
		if _, err := List.Restore(ctx, alice.ID, todo.ID, false); err != nil {
			t.Fatal(err)
		}
		if _, err := List.Restore(ctx, alice.ID, done.ID, true); err != nil {
			t.Fatal(err)
		}
		if got := activeListNames(t, alice.ID, panel.ID); !equalNames(got, []string{"完成", "进行中", "待办"}) {
			t.Fatalf("恢复后的列表 = %v", got)
		}

		// 面板不在正常使用时，不能恢复其中的列表。
		if err := List.Archive(ctx, alice.ID, todo.ID); err != nil {
			t.Fatal(err)
		}
		if err := Panel.Archive(ctx, alice.ID, panel.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := List.Restore(ctx, alice.ID, todo.ID, false); errorKind(err) != KindBadRequest {
			t.Fatalf("恢复已归档面板中的列表 err = %v", err)
		}

		// 已归档的面板仍可以加载内容，用于只读查看；内容包含列表归档中的列表。
		content, err := List.Content(ctx, alice.ID, panel.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(content.Lists) != 3 || content.Revision == 0 {
			t.Fatalf("面板内容 = %+v", content)
		}
	})
}

func TestListIsolatedByUser(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		panel := firstPanel(t, alice.ID, mustBoard(t, alice.ID, "工作").ID)
		list := mustList(t, alice.ID, panel.ID, "待办")

		if _, err := List.Create(ctx, bob.ID, panel.ID, "插进别人的面板"); errorKind(err) != KindNotFound {
			t.Fatalf("在其他用户的面板中建列表 err = %v", err)
		}
		if _, err := List.Content(ctx, bob.ID, panel.ID); errorKind(err) != KindNotFound {
			t.Fatalf("加载其他用户的面板 err = %v", err)
		}
		if err := List.Archive(ctx, bob.ID, list.ID); errorKind(err) != KindNotFound {
			t.Fatalf("归档其他用户的列表 err = %v", err)
		}
	})
}
