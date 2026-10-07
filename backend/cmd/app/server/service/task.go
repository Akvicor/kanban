package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	// maxTaskDepth 是任务的最大层数。
	maxTaskDepth = 3
	// taskTitleMaxLength 是任务标题的最大字符数。
	taskTitleMaxLength = 1000
	// maxTasksPerCreate 是一次粘贴多行文字最多创建的任务数。
	maxTasksPerCreate = 200
)

// Task 是卡片中任务的服务：新建（含粘贴多行批量新建）、改标题、勾选、拖动、删除。
var Task = new(taskService)

type taskService struct{}

// cardTasks 是一张卡片的全部任务，用于计算层级和子树。
type cardTasks struct {
	byID     map[int64]*model.Task
	children map[int64][]*model.Task // 父任务 ID → 子任务；第一层的父任务 ID 记为 0
}

func loadCardTasks(ctx context.Context, userID, cardID int64) (*cardTasks, error) {
	tasks, err := repository.Task.ListByCards(ctx, userID, []int64{cardID})
	if err != nil {
		return nil, err
	}
	result := &cardTasks{byID: map[int64]*model.Task{}, children: map[int64][]*model.Task{}}
	for _, task := range tasks {
		result.byID[task.ID] = task
		parent := int64(0)
		if task.ParentID != nil {
			parent = *task.ParentID
		}
		result.children[parent] = append(result.children[parent], task)
	}
	return result, nil
}

// depth 返回任务的层数，第一层为 1；id 为 nil 表示卡片本身，层数为 0。
func (t *cardTasks) depth(id *int64) int {
	depth := 0
	for id != nil && depth <= maxTaskDepth {
		depth++
		task, ok := t.byID[*id]
		if !ok {
			break
		}
		id = task.ParentID
	}
	return depth
}

// height 返回以 id 为根的子树层数，只有它自己时为 1。
func (t *cardTasks) height(id int64) int {
	height := 0
	for _, child := range t.children[id] {
		height = max(height, t.height(child.ID))
	}
	return height + 1
}

// subtree 返回 id 及其全部下级任务的 ID。
func (t *cardTasks) subtree(id int64) []int64 {
	ids := []int64{id}
	for _, child := range t.children[id] {
		ids = append(ids, t.subtree(child.ID)...)
	}
	return ids
}

// siblings 返回父任务（nil 为第一层）下的任务，按顺序。
func (t *cardTasks) siblings(parentID *int64) []*model.Task {
	key := int64(0)
	if parentID != nil {
		key = *parentID
	}
	return t.children[key]
}

func validateTaskTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", badRequest(resp.TaskTitleEmpty, "任务标题不能为空")
	}
	if utf8.RuneCountInString(title) > taskTitleMaxLength {
		return "", badRequest(resp.TaskTitleTooLong, "任务标题不能超过 1000 个字符")
	}
	return title, nil
}

// usableTask 查找任务，并确认它所在的卡片可以修改。返回任务和所在卡片。
func usableTask(ctx context.Context, userID, id int64) (*model.Task, *model.Card, error) {
	task, err := repository.Task.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, notFound(resp.TaskNotFound, "任务不存在")
	}
	if err != nil {
		return nil, nil, err
	}
	card, _, err := usableCard(ctx, userID, task.CardID)
	if err != nil {
		return nil, nil, err
	}
	return task, card, nil
}

// Create 在父任务（nil 为第一层）下按顺序新建一个或多个任务，排在末尾。
// titles 来自粘贴的多行文字时，每行一个任务，空行忽略。新任务的层数不能超过 3。
func (s *taskService) Create(ctx context.Context, userID, cardID int64, parentID *int64, titles []string) ([]*model.Task, error) {
	var valid []string
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		title, err := validateTaskTitle(title)
		if err != nil {
			return nil, err
		}
		valid = append(valid, title)
	}
	if len(valid) == 0 {
		return nil, badRequest(resp.TaskTitleEmpty, "任务标题不能为空")
	}
	if len(valid) > maxTasksPerCreate {
		return nil, badRequest(resp.TaskBatchLimit, "一次最多新建 200 个任务")
	}
	var created []*model.Task
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, _, err := usableCard(ctx, userID, cardID); err != nil {
			return err
		}
		tasks, err := loadCardTasks(ctx, userID, cardID)
		if err != nil {
			return err
		}
		if parentID != nil {
			if _, ok := tasks.byID[*parentID]; !ok {
				return badRequest(resp.TaskParentWrongCard, "父任务不属于这张卡片")
			}
		}
		if tasks.depth(parentID)+1 > maxTaskDepth {
			return badRequest(resp.TaskDepthLimit, "任务最多三层")
		}
		positions := make([]int64, 0)
		for _, sibling := range tasks.siblings(parentID) {
			positions = append(positions, sibling.Position)
		}
		for _, title := range valid {
			position, _ := slotPosition(positions, nil)
			positions = append(positions, position)
			task := &model.Task{UserID: userID, CardID: cardID, ParentID: parentID, Title: title, Position: position}
			if err = repository.Task.Create(ctx, task); err != nil {
				return err
			}
			if err = recordTask(ctx, task); err != nil {
				return err
			}
			created = append(created, task)
		}
		return nil
	})
	return created, err
}

// Rename 修改任务标题。
func (s *taskService) Rename(ctx context.Context, userID, id int64, title string) (*model.Task, error) {
	title, err := validateTaskTitle(title)
	if err != nil {
		return nil, err
	}
	var task *model.Task
	err = write(ctx, func(ctx context.Context) error {
		if task, _, err = usableTask(ctx, userID, id); err != nil {
			return err
		}
		task.Title = title
		if err = repository.Task.Update(ctx, userID, id, map[string]any{"title": title}); err != nil {
			return err
		}
		return recordTask(ctx, task)
	})
	return task, err
}

// SetDone 勾选或取消勾选任务，返回状态被改动的任务，被勾选的任务在最前；状态没有变化时只返回它自己。
// 勾选时它的全部下级任务一起勾选；取消勾选只改它自己。
// 只记一条完成或取消完成任务的操作，完成时数据中的 children 是随之完成的下级任务数。
func (s *taskService) SetDone(ctx context.Context, userID, id int64, done bool) ([]*model.Task, error) {
	var changed []*model.Task
	err := write(ctx, func(ctx context.Context) error {
		task, card, err := usableTask(ctx, userID, id)
		if err != nil {
			return err
		}
		changed = []*model.Task{task}
		if task.Done == done {
			return nil
		}
		if done {
			tasks, err := loadCardTasks(ctx, userID, task.CardID)
			if err != nil {
				return err
			}
			for _, childID := range tasks.subtree(id)[1:] {
				if child := tasks.byID[childID]; !child.Done {
					changed = append(changed, child)
				}
			}
		}
		for _, t := range changed {
			t.Done = done
			if err = repository.Task.Update(ctx, userID, t.ID, map[string]any{"done": done}); err != nil {
				return err
			}
			if err = recordTask(ctx, t); err != nil {
				return err
			}
		}
		if !done {
			return addAction(ctx, card, model.ActionTaskUncomplete, map[string]any{"task_id": id, "title": task.Title}, nowUTC())
		}
		return addAction(ctx, card, model.ActionTaskComplete, map[string]any{"task_id": id, "title": task.Title, "children": len(changed) - 1}, nowUTC())
	})
	return changed, err
}

// Move 把任务连同下级移到父任务（nil 为第一层）下的第 index 位，index 为 nil 时排在末尾。
// 不能移到自己或自己的下级下面；移动后任何任务的层数都不能超过 3。
func (s *taskService) Move(ctx context.Context, userID, id int64, parentID *int64, index *int) (*model.Task, error) {
	var task *model.Task
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if task, _, err = usableTask(ctx, userID, id); err != nil {
			return err
		}
		tasks, err := loadCardTasks(ctx, userID, task.CardID)
		if err != nil {
			return err
		}
		if parentID != nil {
			if _, ok := tasks.byID[*parentID]; !ok {
				return badRequest(resp.TaskParentWrongCard, "父任务不属于这张卡片")
			}
			if slices.Contains(tasks.subtree(id), *parentID) {
				return badRequest(resp.TaskSelfMove, "不能把任务移到它自己或它的下级下面")
			}
		}
		if tasks.depth(parentID)+tasks.height(id) > maxTaskDepth {
			return badRequest(resp.TaskDepthLimit, "任务最多三层")
		}
		save := func(ctx context.Context, t *model.Task, value int64) error {
			t.Position = value
			if err := repository.Task.Update(ctx, userID, t.ID, map[string]any{"position": value}); err != nil {
				return err
			}
			return recordTask(ctx, t)
		}
		position, err := placeInList(ctx, tasks.siblings(parentID), id, index,
			func(t *model.Task) int64 { return t.ID },
			func(t *model.Task) int64 { return t.Position },
			save)
		if err != nil {
			return err
		}
		task.ParentID, task.Position = parentID, position
		if err = repository.Task.Update(ctx, userID, id, map[string]any{"parent_id": parentID, "position": position}); err != nil {
			return err
		}
		return recordTask(ctx, task)
	})
	return task, err
}

// Delete 彻底删除任务连同它的全部下级任务，不能恢复。
func (s *taskService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		task, _, err := usableTask(ctx, userID, id)
		if err != nil {
			return err
		}
		tasks, err := loadCardTasks(ctx, userID, task.CardID)
		if err != nil {
			return err
		}
		ids := tasks.subtree(id)
		if err = repository.Task.Delete(ctx, userID, ids); err != nil {
			return err
		}
		for _, taskID := range ids {
			if err = recordChange(ctx, userID, hub.OpDelete, EntityTask, taskID, nil); err != nil {
				return err
			}
		}
		return nil
	})
}
