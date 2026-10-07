package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"strconv"
	"sync"
	"time"

	"github.com/Akvicor/glog"
	"gorm.io/gorm"
)

// 文件相关的同步实体类型。全局文件属于系统，不同步；附件和用户文件随用户的变更推送。
const (
	EntityAttachment = "attachment"
	EntityUserFile   = "user_file"
)

// File 是文件管理的服务：列出用户的文件，删除不再被引用的文件，以及按用户文件读取内容。
//
// 引用计数规则：
//   - 用户文件的引用数是该用户引用它的附件数，新建附件、复制卡片时加一，删除附件时减一，降到 0 时记下归零时间；
//   - 全局文件的引用数是引用它的用户文件数，用户第一次引用时加一，删除用户文件记录时减一；
//   - 全局引用数降到 0 时删除全局记录，事务提交后删除存储中的原文件和缩略图。
var File = new(fileService)

type fileService struct{}

// blobStripes 按 sha256 分段的进程内锁。服务端是单进程，同一 sha256 的「上传完成后放入存储并建立记录」
// 「删除记录后删除文件」「清理孤立文件」在同一把锁下进行，删除文件前重新确认库中没有记录，
// 因此不会删掉刚被重新上传的同一份文件。
var blobStripes [64]sync.Mutex

func lockBlob(sha string) func() {
	index, _ := strconv.ParseUint(sha[:2], 16, 8)
	stripe := &blobStripes[index%uint64(len(blobStripes))]
	stripe.Lock()
	return stripe.Unlock
}

// errFileGone 表示引用的全局文件在这期间被删除，客户端重新查询后上传即可。
func errFileGone() error {
	return conflict(resp.FileDeleted, "文件已被删除，请重新上传")
}

func recordUserFile(ctx context.Context, file *model.UserFile, blob *model.Blob) error {
	return recordChange(ctx, file.UserID, hub.OpUpsert, EntityUserFile, file.ID, dro.NewUserFile(file, blob))
}

func findBlob(ctx context.Context, id int64) (*model.Blob, error) {
	blobs, err := repository.Blob.ListByIDs(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	blob, ok := blobs[id]
	if !ok {
		return nil, errFileGone()
	}
	return blob, nil
}

// referenceBlob 为用户新增一次对全局文件的引用（新建文件附件时调用），返回用户文件。
// 用户还没有这份文件时新建用户文件记录，全局引用数加一；已有时用户引用数加一。名称更新为这次引用的附件名称。
func referenceBlob(ctx context.Context, userID int64, blob *model.Blob, name string, now time.Time) (*model.UserFile, error) {
	file, err := repository.UserFile.FindByBlob(ctx, userID, blob.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if _, err = repository.Blob.AddReference(ctx, blob.ID, 1); errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errFileGone()
		} else if err != nil {
			return nil, err
		}
		file = &model.UserFile{UserID: userID, BlobID: blob.ID, Name: name, ReferenceCount: 1, FirstReferencedAt: now}
		if err = repository.UserFile.Create(ctx, file); err != nil {
			return nil, err
		}
		return file, recordUserFile(ctx, file, blob)
	}
	if err != nil {
		return nil, err
	}
	return file, changeUserFileReference(ctx, file, 1, name, now)
}

// changeUserFileReference 把用户引用数加 delta。降到 0 时记下归零时间；增加时清空归零时间。
// name 不为空时更新为最近一次引用它的附件名称。
func changeUserFileReference(ctx context.Context, file *model.UserFile, delta int64, name string, now time.Time) error {
	file.ReferenceCount += delta
	if file.ReferenceCount < 0 {
		file.ReferenceCount = 0
	}
	switch {
	case file.ReferenceCount == 0:
		file.ZeroAt = &now
	case delta > 0:
		file.ZeroAt = nil
	}
	if name != "" {
		file.Name = name
	}
	values := map[string]any{"reference_count": file.ReferenceCount, "zero_at": file.ZeroAt, "name": file.Name}
	if err := repository.UserFile.Update(ctx, file.UserID, file.ID, values); err != nil {
		return err
	}
	blob, err := findBlob(ctx, file.BlobID)
	if err != nil {
		return err
	}
	return recordUserFile(ctx, file, blob)
}

// releaseBlob 把全局引用数减一。降到 0 时删除全局记录，并在事务提交后删除存储中的文件。
// 必须在 write 中调用。
func releaseBlob(ctx context.Context, blobID int64) error {
	blob, err := repository.Blob.AddReference(ctx, blobID, -1)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil || blob.ReferenceCount > 0 {
		return err
	}
	if err = repository.Blob.Delete(ctx, blob.ID); err != nil {
		return err
	}
	sha := blob.SHA256
	return afterCommit(ctx, func() { removeBlobFiles(context.Background(), sha) })
}

// removeBlobFiles 在库中没有该 sha256 的记录时删除原文件和缩略图。删除失败只记录日志，不影响数据。
func removeBlobFiles(ctx context.Context, sha string) {
	unlock := lockBlob(sha)
	defer unlock()
	if _, err := repository.Blob.FindBySHA256(ctx, sha); err == nil {
		return // 已被重新上传
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		glog.Error("检查全局文件 %s 失败，保留文件: %v", sha, err)
		return
	}
	if err := storage.Get().RemoveBlob(sha); err != nil {
		glog.Error("删除全局文件 %s 失败: %v", sha, err)
	}
}

// List 返回用户的全部文件和读取时的同步序号，用于文件管理。客户端把列表视为该序号时的状态，之后按推送更新。
func (s *fileService) List(ctx context.Context, userID int64) (*dro.UserFileList, error) {
	result := &dro.UserFileList{Files: []dro.UserFile{}}
	err := repository.ReadTransaction(ctx, func(ctx context.Context) error {
		user, err := User.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		files, err := repository.UserFile.List(ctx, userID)
		if err != nil {
			return err
		}
		blobIDs := make([]int64, len(files))
		for i, file := range files {
			blobIDs[i] = file.BlobID
		}
		blobs, err := repository.Blob.ListByIDs(ctx, blobIDs)
		if err != nil {
			return err
		}
		for _, file := range files {
			if blob, ok := blobs[file.BlobID]; ok {
				result.Files = append(result.Files, dro.NewUserFile(file, blob))
			}
		}
		result.Revision = user.Revision
		return nil
	})
	return result, err
}

// Delete 删除用户的文件记录。只有引用数为 0 的文件可以删除；删除后全局引用数减一。
func (s *fileService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		file, err := repository.UserFile.FindByID(ctx, userID, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound(resp.FileNotFound, "文件不存在")
		}
		if err != nil {
			return err
		}
		if file.ReferenceCount > 0 {
			return badRequest(resp.FileInUse, "文件仍被附件引用，不能删除")
		}
		if err = repository.UserFile.Delete(ctx, userID, id); err != nil {
			return err
		}
		if err = recordChange(ctx, userID, hub.OpDelete, EntityUserFile, id, nil); err != nil {
			return err
		}
		return releaseBlob(ctx, file.BlobID)
	})
}

// Open 返回用户文件引用的全局文件，用于在文件管理中查看内容和缩略图。
func (s *fileService) Open(ctx context.Context, userID, id int64) (*model.UserFile, *model.Blob, error) {
	file, err := repository.UserFile.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, notFound(resp.FileNotFound, "文件不存在")
	}
	if err != nil {
		return nil, nil, err
	}
	blob, err := findBlob(ctx, file.BlobID)
	if err != nil {
		return nil, nil, notFound(resp.FileNotFound, "文件不存在")
	}
	return file, blob, nil
}

// deleteUserFiles 删除用户的附件、用户文件和未完成的上传，对应的全局引用数减一。删除用户时调用。
func deleteUserFiles(ctx context.Context, userID int64) error {
	files, err := repository.UserFile.List(ctx, userID)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err = releaseBlob(ctx, file.BlobID); err != nil {
			return err
		}
	}
	if err = repository.Attachment.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err = repository.UserFile.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	sessions, err := repository.UploadSession.ListIDsByUser(ctx, userID)
	if err != nil {
		return err
	}
	if err = repository.UploadSession.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	return afterCommit(ctx, func() {
		for _, id := range sessions {
			if err := storage.Get().RemoveUpload(id); err != nil {
				glog.Error("删除上传临时文件 %d 失败: %v", id, err)
			}
		}
	})
}
