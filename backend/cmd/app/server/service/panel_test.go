package service

import (
	"context"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"testing"
)

// boardPanels 返回看板中未进入面板归档的面板名称，按标签页顺序。
func boardPanels(t *testing.T, userID, boardID int64) []string {
	t.Helper()
	panels, err := repository.Panel.ListActiveInBoard(context.Background(), userID, boardID)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(panels))
	for _, panel := range panels {
		names = append(names, panel.Name)
	}
	return names
}

func mustBoard(t *testing.T, userID int64, name string) *model.Board {
	t.Helper()
	board, err := Board.Create(context.Background(), userID, nil, name, nil)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func mustPanel(t *testing.T, userID, boardID int64, name string) *model.Panel {
	t.Helper()
	panel, err := Panel.Create(context.Background(), userID, boardID, name)
	if err != nil {
		t.Fatal(err)
	}
	return panel
}

func mainPanel(t *testing.T, userID, boardID int64) *int64 {
	t.Helper()
	board, err := repository.Board.FindByID(context.Background(), userID, boardID)
	if err != nil {
		t.Fatal(err)
	}
	return board.MainPanelID
}

func TestNewBoardGetsDefaultPanel(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		board := mustBoard(t, alice.ID, "工作")

		if got := boardPanels(t, alice.ID, board.ID); !equalNames(got, []string{"默认"}) {
			t.Fatalf("新看板的面板 = %v", got)
		}
		if mainPanel(t, alice.ID, board.ID) != nil {
			t.Fatal("新看板不应设置主面板")
		}
		panels, _ := repository.Panel.ListActiveInBoard(ctx, alice.ID, board.ID)
		levels, err := repository.PriorityLevel.ListByPanel(ctx, alice.ID, panels[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"P0 #E5484D", "P1 #F76B15", "P2 #3E63DD", "P3 #8B8D98"}
		for i, level := range levels {
			if got := level.Name + " " + level.Color; i >= len(want) || got != want[i] {
				t.Fatalf("默认挡位 = %+v", levels)
			}
		}
		if len(levels) != len(want) {
			t.Fatalf("默认挡位数量 = %d", len(levels))
		}
	})
}

func TestPanelReorderMoveAndMainPanel(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		work := mustBoard(t, alice.ID, "工作")
		life := mustBoard(t, alice.ID, "生活")
		plan := mustPanel(t, alice.ID, work.ID, "计划")
		mustPanel(t, alice.ID, work.ID, "复盘")

		if _, err := Panel.Reorder(ctx, alice.ID, plan.ID, ptr(0)); err != nil {
			t.Fatal(err)
		}
		if got := boardPanels(t, alice.ID, work.ID); !equalNames(got, []string{"计划", "默认", "复盘"}) {
			t.Fatalf("排序后 = %v", got)
		}

		// 主面板必须属于该看板。
		if _, err := Panel.SetMain(ctx, alice.ID, life.ID, &plan.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("把其他看板的面板设为主面板 err = %v", err)
		}
		if _, err := Panel.SetMain(ctx, alice.ID, work.ID, &plan.ID); err != nil {
			t.Fatal(err)
		}

		// 主面板移到其他看板：原看板的主面板清空，移入的面板排在最后、不成为目标看板的主面板。
		if _, err := Panel.MoveToBoard(ctx, alice.ID, plan.ID, life.ID); err != nil {
			t.Fatal(err)
		}
		if mainPanel(t, alice.ID, work.ID) != nil || mainPanel(t, alice.ID, life.ID) != nil {
			t.Fatal("移动主面板后主面板指向不正确")
		}
		if got := boardPanels(t, alice.ID, life.ID); !equalNames(got, []string{"默认", "计划"}) {
			t.Fatalf("目标看板的面板 = %v", got)
		}
		if _, err := Panel.MoveToBoard(ctx, alice.ID, plan.ID, life.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("移到自己所在的看板 err = %v", err)
		}
	})
}

func TestPanelArchiveAndRestore(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		work := mustBoard(t, alice.ID, "工作")
		life := mustBoard(t, alice.ID, "生活")
		plan := mustPanel(t, alice.ID, work.ID, "计划")
		if _, err := Panel.SetMain(ctx, alice.ID, work.ID, &plan.ID); err != nil {
			t.Fatal(err)
		}

		// 归档主面板：清空主面板指向，不能再修改，也不能设为主面板。
		if err := Panel.Archive(ctx, alice.ID, plan.ID); err != nil {
			t.Fatal(err)
		}
		if mainPanel(t, alice.ID, work.ID) != nil {
			t.Fatal("主面板归档后指向未清空")
		}
		if _, err := Panel.Rename(ctx, alice.ID, plan.ID, "改名"); errorKind(err) != KindBadRequest {
			t.Fatalf("修改已归档的面板 err = %v", err)
		}
		if _, err := Label.Create(ctx, alice.ID, plan.ID, "紧急", "#ff0000"); errorKind(err) != KindBadRequest {
			t.Fatalf("在已归档的面板中建标签 err = %v", err)
		}

		// 恢复到另一个看板：排在最后，不成为主面板。
		restored, err := Panel.Restore(ctx, alice.ID, plan.ID, life.ID)
		if err != nil {
			t.Fatal(err)
		}
		if restored.Archived() || restored.BoardID != life.ID {
			t.Fatalf("恢复后的面板 = %+v", restored)
		}
		if got := boardPanels(t, alice.ID, life.ID); !equalNames(got, []string{"默认", "计划"}) {
			t.Fatalf("恢复后目标看板的面板 = %v", got)
		}
		if _, err = Panel.Restore(ctx, alice.ID, plan.ID, work.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("恢复不在归档中的面板 err = %v", err)
		}
	})
}

func TestPanelsFollowBoardArchive(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		work := mustBoard(t, alice.ID, "工作")
		life := mustBoard(t, alice.ID, "生活")
		plan := mustPanel(t, alice.ID, work.ID, "计划")
		review := mustPanel(t, alice.ID, work.ID, "复盘")
		if err := Panel.Archive(ctx, alice.ID, review.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := Panel.SetMain(ctx, alice.ID, work.ID, &plan.ID); err != nil {
			t.Fatal(err)
		}
		if err := Board.Archive(ctx, alice.ID, work.ID); err != nil {
			t.Fatal(err)
		}

		// 看板归档后，其中的面板不能修改、不能移走，也不能在其中新建面板。
		if _, err := Panel.Rename(ctx, alice.ID, plan.ID, "改名"); errorKind(err) != KindBadRequest {
			t.Fatalf("修改归档看板中的面板 err = %v", err)
		}
		if _, err := Panel.Create(ctx, alice.ID, work.ID, "新面板"); errorKind(err) != KindBadRequest {
			t.Fatalf("在归档看板中新建面板 err = %v", err)
		}

		// 从看板归档中只恢复一个面板：面板移到目标看板，从归档的看板中移走，并清空原看板对它的主面板指向。
		if _, err := Panel.Restore(ctx, alice.ID, plan.ID, life.ID); err != nil {
			t.Fatal(err)
		}
		if mainPanel(t, alice.ID, work.ID) != nil {
			t.Fatal("恢复出去的面板仍是归档看板的主面板")
		}
		if got := boardPanels(t, alice.ID, work.ID); !equalNames(got, []string{"默认"}) {
			t.Fatalf("归档看板中剩下的面板 = %v", got)
		}

		// 恢复整个看板：单独归档的面板仍留在面板归档中。
		if _, err := Board.Restore(ctx, alice.ID, work.ID, nil, nil); err != nil {
			t.Fatal(err)
		}
		if got := boardPanels(t, alice.ID, work.ID); !equalNames(got, []string{"默认"}) {
			t.Fatalf("恢复看板后的面板 = %v", got)
		}
		reviewAfter, _ := repository.Panel.FindByID(ctx, alice.ID, review.ID)
		if !reviewAfter.Archived() {
			t.Fatal("恢复看板时一并恢复了单独归档的面板")
		}
	})
}

func TestBoardRestoreKeepsMainPanel(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		work := mustBoard(t, alice.ID, "工作")
		plan := mustPanel(t, alice.ID, work.ID, "计划")
		if _, err := Panel.SetMain(ctx, alice.ID, work.ID, &plan.ID); err != nil {
			t.Fatal(err)
		}
		if err := Board.Archive(ctx, alice.ID, work.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := Board.Restore(ctx, alice.ID, work.ID, nil, nil); err != nil {
			t.Fatal(err)
		}
		if main := mainPanel(t, alice.ID, work.ID); main == nil || *main != plan.ID {
			t.Fatalf("看板恢复后主面板 = %v", main)
		}
	})
}

func TestLabelsAndPriorityLevels(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		board := mustBoard(t, alice.ID, "工作")
		panels, _ := repository.Panel.ListActiveInBoard(ctx, alice.ID, board.ID)
		panelID := panels[0].ID

		urgent, err := Label.Create(ctx, alice.ID, panelID, " 紧急 ", "#e5484d")
		if err != nil {
			t.Fatal(err)
		}
		if urgent.Name != "紧急" || urgent.Color != "#E5484D" {
			t.Fatalf("标签 = %+v", urgent)
		}
		later, _ := Label.Create(ctx, alice.ID, panelID, "", "#3E63DD") // 只有颜色的标签
		if _, err = Label.Reorder(ctx, alice.ID, later.ID, ptr(0)); err != nil {
			t.Fatal(err)
		}
		labels, _ := repository.Label.ListByPanel(ctx, alice.ID, panelID)
		if len(labels) != 2 || labels[0].ID != later.ID {
			t.Fatalf("标签顺序 = %+v", labels)
		}
		if _, err = Label.Create(ctx, alice.ID, panelID, "坏颜色", "red"); errorKind(err) != KindBadRequest {
			t.Fatalf("颜色格式错误 err = %v", err)
		}
		if _, err = Label.Update(ctx, bob.ID, urgent.ID, "改名", "#000000"); errorKind(err) != KindNotFound {
			t.Fatalf("修改其他用户的标签 err = %v", err)
		}
		if err = Label.Delete(ctx, alice.ID, urgent.ID); err != nil {
			t.Fatal(err)
		}

		levels, _ := repository.PriorityLevel.ListByPanel(ctx, alice.ID, panelID)
		p3 := levels[3]
		if _, err = PriorityLevel.Reorder(ctx, alice.ID, p3.ID, ptr(0)); err != nil {
			t.Fatal(err)
		}
		if _, err = PriorityLevel.Update(ctx, alice.ID, p3.ID, "最高", "#000000"); err != nil {
			t.Fatal(err)
		}
		if _, err = PriorityLevel.Update(ctx, alice.ID, p3.ID, " ", "#000000"); errorKind(err) != KindBadRequest {
			t.Fatalf("空挡位名称 err = %v", err)
		}
		if err = PriorityLevel.Delete(ctx, alice.ID, levels[0].ID); err != nil {
			t.Fatal(err)
		}
		levels, _ = repository.PriorityLevel.ListByPanel(ctx, alice.ID, panelID)
		names := make([]string, 0, len(levels))
		for _, level := range levels {
			names = append(names, level.Name)
		}
		if !equalNames(names, []string{"最高", "P1", "P2"}) {
			t.Fatalf("挡位 = %v", names)
		}
	})
}
