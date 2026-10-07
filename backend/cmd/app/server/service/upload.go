package service

import (
	"context"
	"errors"
	"io"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/filetype"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/thumbnail"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"os"
	"time"

	"github.com/Akvicor/glog"
	"gorm.io/gorm"
)

const (
	// MaxFileSize 是单个文件的大小上限。
	MaxFileSize = 4 << 30
	// MaxChunkSize 是单个分片的大小上限。
	MaxChunkSize = 16 << 20
	// uploadExpiry 是上传会话没有更新时保留的时长：未完成的上传没有继续，或已完成的上传凭证没有被使用。
	uploadExpiry = 24 * time.Hour
	// orphanBatch 是清理孤立文件时每次查询的数量。
	orphanBatch = 500
)

// Upload 是分片上传的服务：
//  1. Prepare：客户端给出 sha256 和大小。当前用户已持有这份文件时直接返回「已存在」；
//     否则返回该用户同一文件的未完成会话（没有则新建），以及服务端已收到的字节数。
//  2. Chunk：按顺序上传分片，偏移必须等于已收到的字节数。收齐后服务端计算 sha256，
//     一致时识别类型、生成缩略图、放入存储并建立全局文件记录（系统已有同一份文件时丢弃这次的内容），
//     会话标记为已完成，作为该用户持有这份内容的凭证；不一致时删除会话并报错。
//  3. 之后客户端用 sha256 新建文件附件，见 Attachment.CreateFile。
//
// 存储按 sha256 全局去重，但引用一份文件之前必须证明持有它的内容：已有引用它的用户文件，或完整上传过一次。
// 只知道 sha256 不能引用别人的文件，也不能探测某份文件是否存在于系统中。
//
// 上传会话不是同步实体，不记录变更。
var Upload = new(uploadService)

type uploadService struct{}

func findSession(ctx context.Context, userID, id int64) (*model.UploadSession, error) {
	session, err := repository.UploadSession.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.UploadExpired, "上传已过期，请重新上传")
	}
	return session, err
}

// heldBlob 返回用户持有的全局文件：全局文件存在，且用户已有引用它的用户文件，或持有该 sha256 的上传凭证。
// viaProof 表示是凭上传凭证持有的。用户没有持有或文件不存在时 blob 为 nil，两种情况不做区分。
func heldBlob(ctx context.Context, userID int64, sha string) (blob *model.Blob, viaProof bool, err error) {
	blob, err = repository.Blob.FindBySHA256(ctx, sha)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if _, err = repository.UserFile.FindByBlob(ctx, userID, blob.ID); err == nil {
		return blob, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	held, err := repository.UploadSession.HasCompleted(ctx, userID, sha)
	if err != nil || !held {
		return nil, false, err
	}
	return blob, true, nil
}

// Prepare 开始或继续上传一份文件。
func (s *uploadService) Prepare(ctx context.Context, userID int64, sha string, size int64) (dro.UploadState, error) {
	if !storage.ValidSHA256(sha) {
		return dro.UploadState{}, badRequest(resp.UploadSha256Invalid, "sha256 格式不正确")
	}
	if size < 0 || size > MaxFileSize {
		return dro.UploadState{}, badRequest(resp.UploadTooLarge, "单个文件不能超过 4GiB")
	}
	if blob, _, err := heldBlob(ctx, userID, sha); err != nil {
		return dro.UploadState{}, err
	} else if blob != nil {
		return dro.UploadState{Exists: true}, nil
	}

	var state dro.UploadState
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		now := nowUTC()
		session, err := repository.UploadSession.FindResumable(ctx, userID, sha, size)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			session = &model.UploadSession{UserID: userID, SHA256: sha, Size: size, CreatedAt: now, UpdatedAt: now}
			if err = repository.UploadSession.Create(ctx, session); err != nil {
				return err
			}
			state = dro.UploadState{SessionID: session.ID}
			return nil
		}
		if err != nil {
			return err
		}
		received, err := storage.Get().UploadSize(session.ID)
		if err != nil {
			return err
		}
		state = dro.UploadState{SessionID: session.ID, Received: received}
		return repository.UploadSession.Update(ctx, userID, session.ID, map[string]any{"received": received, "updated_at": now})
	})
	return state, err
}

// Chunk 把一个分片追加到上传会话。offset 必须等于已收到的字节数；收齐后完成上传，返回 Done。
// 会话已完成时直接返回 Done，便于客户端重试最后一个分片。
func (s *uploadService) Chunk(ctx context.Context, userID, sessionID, offset int64, data io.Reader) (dro.UploadState, error) {
	session, err := findSession(ctx, userID, sessionID)
	if err != nil {
		return dro.UploadState{}, err
	}
	if session.Completed {
		return dro.UploadState{SessionID: session.ID, Received: session.Size, Done: true}, nil
	}
	if offset < 0 || offset > session.Size {
		return dro.UploadState{}, badRequest(resp.UploadOffsetInvalid, "分片偏移不正确")
	}
	// 同一份文件的分片追加和收齐后的处理串行进行，避免并发请求交错写入或重复处理。
	unlock := lockBlob(session.SHA256)
	defer unlock()
	received, err := storage.Get().AppendUpload(session.ID, offset, data, min(MaxChunkSize, session.Size-offset))
	if errors.Is(err, storage.ErrOffsetMismatch) {
		return dro.UploadState{}, conflict(resp.UploadOffsetConflict, "分片偏移与服务端已收到的大小不一致，请重新查询后继续")
	}
	if err != nil {
		return dro.UploadState{}, err
	}
	if err = repository.UploadSession.Update(ctx, userID, session.ID, map[string]any{"received": received, "updated_at": nowUTC()}); err != nil {
		return dro.UploadState{}, err
	}
	state := dro.UploadState{SessionID: session.ID, Received: received}
	if received < session.Size {
		return state, nil
	}
	if err = s.finish(ctx, session); err != nil {
		return dro.UploadState{}, err
	}
	state.Done = true
	return state, nil
}

// finish 校验收齐的文件，放入存储并建立全局文件记录，然后把会话标记为已完成，作为该用户持有这份内容的凭证。
// 系统已有同一份文件时丢弃这次的内容，同样标记为已完成。调用方持有该 sha256 的锁。
func (s *uploadService) finish(ctx context.Context, session *model.UploadSession) error {
	store := storage.Get()
	complete := func() error {
		return repository.UploadSession.Update(ctx, session.UserID, session.ID, map[string]any{"completed": true, "updated_at": nowUTC()})
	}
	sha, err := store.HashUpload(session.ID)
	if err != nil {
		return err
	}
	if sha != session.SHA256 {
		if err = repository.UploadSession.Delete(ctx, session.UserID, session.ID); err != nil {
			return err
		}
		if err = store.RemoveUpload(session.ID); err != nil {
			return err
		}
		return badRequest(resp.UploadChecksumMismatch, "文件校验失败（sha256 不一致），请重新上传")
	}

	if _, err = repository.Blob.FindBySHA256(ctx, sha); err == nil {
		// 别的上传已经建好了同一份文件
		if err = complete(); err != nil {
			return err
		}
		return store.RemoveUpload(session.ID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	blob, err := materialize(store, session)
	if err != nil {
		return err
	}
	if _, err = repository.Blob.CreateIfAbsent(ctx, blob); err != nil {
		return err
	}
	return complete()
}

// materialize 识别类型、生成缩略图，并把上传临时文件移入存储，返回待插入的全局文件记录。
// 缩略图生成失败时这份文件不算图片，不影响上传。
func materialize(store *storage.Store, session *model.UploadSession) (*model.Blob, error) {
	path := store.UploadPath(session.ID)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	header := make([]byte, filetype.SniffLength)
	n, _ := io.ReadFull(file, header)
	_ = file.Close()

	blob := &model.Blob{SHA256: session.SHA256, Size: session.Size, MimeType: filetype.Detect(header[:n]), CreatedAt: nowUTC()}
	result, err := thumbnail.Generate(path)
	switch {
	case err == nil:
		saved := true
		for size, data := range result.Thumbnails {
			if err := store.WriteThumbnail(session.SHA256, size, result.Ext, data); err != nil {
				glog.Error("保存缩略图 %s 失败: %v", session.SHA256, err)
				saved = false
			}
		}
		if saved {
			blob.ThumbnailExt, blob.ImageWidth, blob.ImageHeight = result.Ext, result.Width, result.Height
		}
	case !errors.Is(err, thumbnail.ErrNotImage):
		glog.Warning("生成缩略图 %s 失败: %v", session.SHA256, err)
	}
	if err = store.AdoptUpload(session.ID, session.SHA256); err != nil {
		return nil, err
	}
	return blob, nil
}

// ExpireSessions 删除最后活动时间早于 now - 24 小时的上传会话及其临时文件，包括未被使用的上传凭证。
func (s *uploadService) ExpireSessions(ctx context.Context, now time.Time) error {
	sessions, err := repository.UploadSession.ListExpired(ctx, now.Add(-uploadExpiry))
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if err = repository.UploadSession.Delete(ctx, session.UserID, session.ID); err != nil {
			return err
		}
		if err = storage.Get().RemoveUpload(session.ID); err != nil {
			glog.Error("删除过期上传 %d 失败: %v", session.ID, err)
		}
	}
	return nil
}

// CleanOrphanFiles 删除存储中没有对应记录的原文件、缩略图和上传临时文件，例如进程在提交后、删除文件前退出留下的文件。
func CleanOrphanFiles(ctx context.Context) error {
	entries, err := storage.Get().List()
	if err != nil {
		return err
	}
	shas := map[string]bool{}
	for _, sha := range append(entries.Blobs, entries.Thumbnails...) {
		shas[sha] = true
	}
	candidates := make([]string, 0, len(shas))
	for sha := range shas {
		candidates = append(candidates, sha)
	}
	for start := 0; start < len(candidates); start += orphanBatch {
		batch := candidates[start:min(start+orphanBatch, len(candidates))]
		exists, err := repository.Blob.ExistsSHA256(ctx, batch)
		if err != nil {
			return err
		}
		for _, sha := range batch {
			if !exists[sha] {
				removeBlobFiles(ctx, sha)
			}
		}
	}
	for start := 0; start < len(entries.Uploads); start += orphanBatch {
		batch := entries.Uploads[start:min(start+orphanBatch, len(entries.Uploads))]
		exists, err := repository.UploadSession.ExistingIDs(ctx, batch)
		if err != nil {
			return err
		}
		for _, id := range batch {
			if !exists[id] {
				if err := storage.Get().RemoveUpload(id); err != nil {
					glog.Error("删除孤立的上传临时文件 %d 失败: %v", id, err)
				}
			}
		}
	}
	return nil
}

// StartFileJobs 启动文件相关的后台任务：启动时清理孤立文件，之后每小时清理过期的上传会话。
func StartFileJobs(ctx context.Context) {
	go func() {
		if err := CleanOrphanFiles(ctx); err != nil {
			glog.Error("清理孤立文件失败: %v", err)
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if err := Upload.ExpireSessions(ctx, time.Now()); err != nil {
				glog.Error("清理过期上传失败: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
