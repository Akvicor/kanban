package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Task 是任务的仓储。查询都带用户条件，只能取到该用户自己的任务。
var Task = new(taskRepository)

type taskRepository struct{}

// Create 新建任务。
func (*taskRepository) Create(ctx context.Context, task *model.Task) error {
	return conn(ctx).Create(task).Error
}

// FindByID 按 ID 查找用户的任务。
func (*taskRepository) FindByID(ctx context.Context, userID, id int64) (*model.Task, error) {
	task := new(model.Task)
	return task, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(task).Error
}

// ListByCards 返回这些卡片的全部任务，按顺序排列。
func (*taskRepository) ListByCards(ctx context.Context, userID int64, cardIDs []int64) ([]*model.Task, error) {
	tasks := make([]*model.Task, 0)
	if len(cardIDs) == 0 {
		return tasks, nil
	}
	return tasks, conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("position ASC").Order("id ASC").Find(&tasks).Error
}

// Update 按 ID 更新用户任务的指定列。
func (*taskRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Task{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// DeleteByUser 删除用户的全部任务。
func (*taskRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Task{}).Error
}

// Delete 删除这些任务。
func (*taskRepository) Delete(ctx context.Context, userID int64, ids []int64) error {
	return conn(ctx).Where("user_id = ? AND id IN ?", userID, ids).Delete(&model.Task{}).Error
}
