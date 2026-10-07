package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// UserFile 是用户文件的仓储。查询都带用户条件。
var UserFile = new(userFileRepository)

type userFileRepository struct{}

// Create 新建用户文件记录。
func (*userFileRepository) Create(ctx context.Context, file *model.UserFile) error {
	return conn(ctx).Create(file).Error
}

// FindByID 按 ID 查找用户的文件。
func (*userFileRepository) FindByID(ctx context.Context, userID, id int64) (*model.UserFile, error) {
	file := new(model.UserFile)
	return file, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(file).Error
}

// FindByBlob 查找用户引用某个全局文件的记录。
func (*userFileRepository) FindByBlob(ctx context.Context, userID, blobID int64) (*model.UserFile, error) {
	file := new(model.UserFile)
	return file, conn(ctx).Where("user_id = ? AND blob_id = ?", userID, blobID).Take(file).Error
}

// List 返回用户的全部文件。
func (*userFileRepository) List(ctx context.Context, userID int64) ([]*model.UserFile, error) {
	files := make([]*model.UserFile, 0)
	return files, conn(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&files).Error
}

// ListByIDs 按 ID 返回用户的文件，结果以 ID 为键。
func (*userFileRepository) ListByIDs(ctx context.Context, userID int64, ids []int64) (map[int64]*model.UserFile, error) {
	result := map[int64]*model.UserFile{}
	if len(ids) == 0 {
		return result, nil
	}
	var files []*model.UserFile
	if err := conn(ctx).Where("user_id = ? AND id IN ?", userID, ids).Find(&files).Error; err != nil {
		return nil, err
	}
	for _, file := range files {
		result[file.ID] = file
	}
	return result, nil
}

// Update 更新用户文件的字段。
func (*userFileRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.UserFile{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Delete 删除用户的文件记录。
func (*userFileRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.UserFile{}))
}

// DeleteByUser 删除用户的全部文件记录。
func (*userFileRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.UserFile{}).Error
}
