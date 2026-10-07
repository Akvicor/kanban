package repository

import (
	"context"
	"kanban/cmd/app/server/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Blob 是全局文件的仓储。全局文件属于系统，查询不带用户条件；调用方通过用户文件确认归属。
var Blob = new(blobRepository)

type blobRepository struct{}

// CreateIfAbsent 插入全局文件记录；同一 sha256 已存在时不插入，返回 false。
func (*blobRepository) CreateIfAbsent(ctx context.Context, blob *model.Blob) (bool, error) {
	result := conn(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "sha256"}}, DoNothing: true}).Create(blob)
	return result.RowsAffected == 1, result.Error
}

// FindBySHA256 按 sha256 查找全局文件。
func (*blobRepository) FindBySHA256(ctx context.Context, sha string) (*model.Blob, error) {
	blob := new(model.Blob)
	return blob, conn(ctx).Where("sha256 = ?", sha).Take(blob).Error
}

// ListByIDs 按 ID 返回全局文件，结果以 ID 为键。
func (*blobRepository) ListByIDs(ctx context.Context, ids []int64) (map[int64]*model.Blob, error) {
	result := map[int64]*model.Blob{}
	if len(ids) == 0 {
		return result, nil
	}
	var blobs []*model.Blob
	if err := conn(ctx).Where("id IN ?", ids).Find(&blobs).Error; err != nil {
		return nil, err
	}
	for _, blob := range blobs {
		result[blob.ID] = blob
	}
	return result, nil
}

// AddReference 把全局引用数加 delta（可以为负），返回更新后的记录。记录不存在时返回 gorm.ErrRecordNotFound。
func (*blobRepository) AddReference(ctx context.Context, id, delta int64) (*model.Blob, error) {
	err := affected(conn(ctx).Model(&model.Blob{}).Where("id = ?", id).
		UpdateColumn("reference_count", gorm.Expr("reference_count + ?", delta)))
	if err != nil {
		return nil, err
	}
	blob := new(model.Blob)
	return blob, conn(ctx).Where("id = ?", id).Take(blob).Error
}

// Delete 删除全局文件记录。
func (*blobRepository) Delete(ctx context.Context, id int64) error {
	return conn(ctx).Where("id = ?", id).Delete(&model.Blob{}).Error
}

// ExistsSHA256 返回这些 sha256 中在库里有记录的部分。
func (*blobRepository) ExistsSHA256(ctx context.Context, shas []string) (map[string]bool, error) {
	result := map[string]bool{}
	if len(shas) == 0 {
		return result, nil
	}
	var found []string
	if err := conn(ctx).Model(&model.Blob{}).Where("sha256 IN ?", shas).Pluck("sha256", &found).Error; err != nil {
		return nil, err
	}
	for _, sha := range found {
		result[sha] = true
	}
	return result, nil
}
