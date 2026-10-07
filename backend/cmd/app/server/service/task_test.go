package service

import (
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"testing"
)

func TestTaskDepthAndMove(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办")
		card := e.create("待办", "卡片", listsort.Tail)
		mustTasks := func(parent *int64, titles ...string) []*model.Task {
			t.Helper()
			tasks, err := Task.Create(e.ctx, e.user.ID, card.ID, parent, titles)
			if err != nil {
				t.Fatal(err)
			}
			return tasks
		}

		// 粘贴多行文字：每行一个任务，空行忽略。
		top := mustTasks(nil, "一", "", "  ", "二")
		if len(top) != 2 {
			t.Fatalf("批量新建 = %d 个任务", len(top))
		}
		second := mustTasks(&top[0].ID, "一.一")
		third := mustTasks(&second[0].ID, "一.一.一")
		if _, err := Task.Create(e.ctx, e.user.ID, card.ID, &third[0].ID, []string{"第四层"}); errorKind(err) != KindBadRequest {
			t.Fatalf("第四层 err = %v", err)
		}

		// 把有两层下级的「一」移到「二」下面会出现第四层，拒绝；移到第一层的最前可以。
		if _, err := Task.Move(e.ctx, e.user.ID, top[0].ID, &top[1].ID, nil); errorKind(err) != KindBadRequest {
			t.Fatalf("移动后超过三层 err = %v", err)
		}
		if _, err := Task.Move(e.ctx, e.user.ID, top[0].ID, &third[0].ID, nil); errorKind(err) != KindBadRequest {
			t.Fatalf("移到自己的下级下面 err = %v", err)
		}
		if _, err := Task.Move(e.ctx, e.user.ID, top[1].ID, nil, ptr(0)); err != nil {
			t.Fatal(err)
		}
		// 把第三层的任务移到「二」下面成为第二层。
		if _, err := Task.Move(e.ctx, e.user.ID, third[0].ID, &top[1].ID, nil); err != nil {
			t.Fatal(err)
		}

		// 勾选父任务时下级一起勾选，取消勾选只改它自己；勾选和取消勾选各记一条操作。
		changed, err := Task.SetDone(e.ctx, e.user.ID, top[0].ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(changed) != 2 || changed[0].ID != top[0].ID || changed[1].ID != second[0].ID || !changed[1].Done {
			t.Fatalf("勾选父任务改动的任务 = %+v", changed)
		}
		if changed, err = Task.SetDone(e.ctx, e.user.ID, top[0].ID, false); err != nil || len(changed) != 1 {
			t.Fatalf("取消勾选改动的任务 = %+v, err = %v", changed, err)
		}
		child, _ := repository.Task.FindByID(e.ctx, e.user.ID, second[0].ID)
		if !child.Done {
			t.Fatal("取消勾选父任务改动了子任务")
		}
		if got := e.actions(card.ID); !equalNames(got, []string{model.ActionCreate, model.ActionTaskComplete, model.ActionTaskUncomplete}) {
			t.Fatalf("操作记录 = %v", got)
		}

		// 删除任务连同下级一起删除。
		if err := Task.Delete(e.ctx, e.user.ID, top[0].ID); err != nil {
			t.Fatal(err)
		}
		tasks, _ := repository.Task.ListByCards(e.ctx, e.user.ID, []int64{card.ID})
		if len(tasks) != 2 {
			t.Fatalf("删除后剩余任务 = %+v", tasks)
		}
	})
}

func TestCardLinks(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newCardEnv(t, "待办")
		card := e.create("待办", "卡片", listsort.Tail)
		other := mustBoard(t, e.user.ID, "生活")

		link, err := CardLink.Add(e.ctx, e.user.ID, card.ID, &other.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = CardLink.Add(e.ctx, e.user.ID, card.ID, &other.ID, nil); errorKind(err) != KindConflict {
			t.Fatalf("重复关联 err = %v", err)
		}
		// 可以关联卡片自己所在的面板；目标归档后关联保留。
		if _, err = CardLink.Add(e.ctx, e.user.ID, card.ID, nil, &e.panel.ID); err != nil {
			t.Fatal(err)
		}
		if err = Board.Archive(e.ctx, e.user.ID, other.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = CardLink.Add(e.ctx, e.user.ID, card.ID, &other.ID, &e.panel.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("同时指定看板和面板 err = %v", err)
		}
		if err = CardLink.Delete(e.ctx, e.user.ID, link.ID); err != nil {
			t.Fatal(err)
		}
		links, _ := repository.CardLink.ListByCards(e.ctx, e.user.ID, []int64{card.ID})
		if len(links) != 1 {
			t.Fatalf("剩余关联 = %+v", links)
		}
	})
}
