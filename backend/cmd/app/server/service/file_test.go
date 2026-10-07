package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// useStorage 让当前测试使用临时目录作为文件存储。
func useStorage(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	storage.Set(store)
	t.Cleanup(func() { storage.Set(nil) })
	return store
}

func sum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// pngData 生成一张 w×h 的不透明 PNG。
func pngData(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range w * h {
		img.Set(i%w, i/w, color.RGBA{uint8(i), 100, 200, 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// upload 按上传协议完整上传 data（分两片），返回 sha256。
func upload(t *testing.T, userID int64, data []byte) string {
	t.Helper()
	ctx := context.Background()
	sha := sum(data)
	state, err := Upload.Prepare(ctx, userID, sha, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if state.Exists {
		return sha
	}
	half := int64(len(data) / 2)
	if _, err = Upload.Chunk(ctx, userID, state.SessionID, 0, bytes.NewReader(data[:half])); err != nil {
		t.Fatal(err)
	}
	if state, err = Upload.Chunk(ctx, userID, state.SessionID, half, bytes.NewReader(data[half:])); err != nil || !state.Done {
		t.Fatalf("上传完成 state = %+v, err = %v", state, err)
	}
	return sha
}

func blobOf(t *testing.T, sha string) *model.Blob {
	t.Helper()
	blob, err := repository.Blob.FindBySHA256(context.Background(), sha)
	if err != nil {
		t.Fatalf("全局文件 %s: %v", sha, err)
	}
	return blob
}

func userFileOf(t *testing.T, userID, blobID int64) *model.UserFile {
	t.Helper()
	file, err := repository.UserFile.FindByBlob(context.Background(), userID, blobID)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func TestUploadResumeAndVerify(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		store := useStorage(t)
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		data := []byte(strings.Repeat("看板文件内容", 1000))
		sha := sum(data)

		state, err := Upload.Prepare(ctx, alice.ID, sha, int64(len(data)))
		if err != nil || state.Exists || state.Received != 0 {
			t.Fatalf("新上传 state = %+v, err = %v", state, err)
		}
		if _, err = Upload.Chunk(ctx, alice.ID, state.SessionID, 0, bytes.NewReader(data[:100])); err != nil {
			t.Fatal(err)
		}
		// 中断后再次 prepare，续用同一会话并从已收到的位置继续。
		resumed, err := Upload.Prepare(ctx, alice.ID, sha, int64(len(data)))
		if err != nil || resumed.SessionID != state.SessionID || resumed.Received != 100 {
			t.Fatalf("续传 state = %+v, err = %v", resumed, err)
		}
		if _, err = Upload.Chunk(ctx, alice.ID, state.SessionID, 50, bytes.NewReader(data[50:])); errorKind(err) != KindConflict {
			t.Fatalf("偏移不对 err = %v", err)
		}
		done, err := Upload.Chunk(ctx, alice.ID, state.SessionID, 100, bytes.NewReader(data[100:]))
		if err != nil || !done.Done {
			t.Fatalf("完成 state = %+v, err = %v", done, err)
		}
		blob := blobOf(t, sha)
		if blob.Size != int64(len(data)) || blob.MimeType != "text/plain" || blob.IsImage() || blob.ReferenceCount != 0 {
			t.Fatalf("全局文件 = %+v", blob)
		}
		if content, err := os.ReadFile(store.BlobPath(sha)); err != nil || !bytes.Equal(content, data) {
			t.Fatalf("存储中的内容不一致: %v", err)
		}
		// 重试最后一个分片时直接返回完成。
		if again, err := Upload.Chunk(ctx, alice.ID, done.SessionID, 100, bytes.NewReader(data[100:])); err != nil || !again.Done {
			t.Fatalf("重试已完成的分片 state = %+v, err = %v", again, err)
		}
		if state, _ = Upload.Prepare(ctx, alice.ID, sha, int64(len(data))); !state.Exists {
			t.Fatal("已上传的文件应返回已存在")
		}

		// 内容与声明的 sha256 不一致时拒绝，会话被删除。
		fake := strings.Repeat("0", 64)
		state, _ = Upload.Prepare(ctx, alice.ID, fake, 3)
		if _, err = Upload.Chunk(ctx, alice.ID, state.SessionID, 0, strings.NewReader("abc")); errorKind(err) != KindBadRequest {
			t.Fatalf("sha256 不一致 err = %v", err)
		}
		if _, err = repository.UploadSession.FindByID(ctx, alice.ID, state.SessionID); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("校验失败后会话仍在: %v", err)
		}
		if _, err = repository.Blob.FindBySHA256(ctx, fake); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal("校验失败不应建立全局文件")
		}

		// 过期会话被清理，临时文件一并删除。
		state, _ = Upload.Prepare(ctx, alice.ID, strings.Repeat("1", 64), 10)
		if _, err = Upload.Chunk(ctx, alice.ID, state.SessionID, 0, strings.NewReader("12345")); err != nil {
			t.Fatal(err)
		}
		if err = Upload.ExpireSessions(ctx, time.Now().Add(25*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if size, _ := store.UploadSize(state.SessionID); size != 0 {
			t.Fatal("过期会话的临时文件没有删除")
		}
		if _, err = Upload.Chunk(ctx, alice.ID, state.SessionID, 5, strings.NewReader("67890")); errorKind(err) != KindNotFound {
			t.Fatalf("过期会话 err = %v", err)
		}
	})
}

func TestConcurrentUploadsOfSameFile(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		store := useStorage(t)
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		data := []byte("两个人同时上传的文件")
		sha := sum(data)
		// 两人都在对方完成前开始上传，各自得到会话。
		first, _ := Upload.Prepare(ctx, alice.ID, sha, int64(len(data)))
		second, _ := Upload.Prepare(ctx, bob.ID, sha, int64(len(data)))
		if first.SessionID == second.SessionID {
			t.Fatal("不同用户应使用各自的会话")
		}
		for _, s := range []struct {
			userID  int64
			session int64
		}{{alice.ID, first.SessionID}, {bob.ID, second.SessionID}} {
			if state, err := Upload.Chunk(ctx, s.userID, s.session, 0, bytes.NewReader(data)); err != nil || !state.Done {
				t.Fatalf("完成上传 state = %+v, err = %v", state, err)
			}
		}
		// 后完成的一方丢弃自己的内容，直接使用已有的全局文件。
		if blob := blobOf(t, sha); blob.Size != int64(len(data)) {
			t.Fatalf("全局文件 = %+v", blob)
		}
		if size, _ := store.UploadSize(second.SessionID); size != 0 {
			t.Fatal("重复上传的临时文件没有删除")
		}
	})
}

func TestAttachmentReferenceCounts(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		store := useStorage(t)
		e := newCardEnv(t, "待办")
		bob := mustUser(t, "bob")
		card := e.create("待办", "卡片", listsort.Tail)
		data := []byte("共享的文件")
		sha := upload(t, e.user.ID, data)

		first, err := Attachment.CreateFile(e.ctx, e.user.ID, card.ID, sha, "报告.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Attachment.CreateFile(e.ctx, e.user.ID, card.ID, sha, "报告副本.txt"); err != nil {
			t.Fatal(err)
		}
		blob := blobOf(t, sha)
		file := userFileOf(t, e.user.ID, blob.ID)
		if blob.ReferenceCount != 1 || file.ReferenceCount != 2 || file.Name != "报告副本.txt" {
			t.Fatalf("引用数 blob = %d, file = %+v", blob.ReferenceCount, file)
		}

		// 另一个用户只知道 sha256 时不能引用，也看不出系统中已有这份文件。
		bobBoard := mustBoard(t, bob.ID, "个人")
		bobList := mustList(t, bob.ID, firstPanel(t, bob.ID, bobBoard.ID).ID, "收件箱")
		bobCard, err := Card.Create(e.ctx, bob.ID, bobList.ID, "鲍勃的卡片", listsort.Tail)
		if err != nil {
			t.Fatal(err)
		}
		if state, _ := Upload.Prepare(e.ctx, bob.ID, sha, int64(len(data))); state.Exists || state.SessionID == 0 {
			t.Fatalf("没有持有的文件应要求上传: %+v", state)
		}
		if _, err = Attachment.CreateFile(e.ctx, bob.ID, bobCard.ID, sha, "bob.txt"); errorKind(err) != KindConflict {
			t.Fatalf("只凭 sha256 引用他人文件 err = %v", err)
		}
		missing := strings.Repeat("2", 64)
		if _, err = Attachment.CreateFile(e.ctx, bob.ID, bobCard.ID, missing, "bob.txt"); errorKind(err) != KindConflict {
			t.Fatalf("引用不存在的文件 err = %v", err)
		}

		// 完整上传一次后可以引用；存储仍只有一份，全局引用数加一。上传凭证使用一次后失效。
		upload(t, bob.ID, data)
		if _, err = Attachment.CreateFile(e.ctx, bob.ID, bobCard.ID, sha, "bob.txt"); err != nil {
			t.Fatal(err)
		}
		if blob = blobOf(t, sha); blob.ReferenceCount != 2 {
			t.Fatalf("两个用户引用后全局引用数 = %d", blob.ReferenceCount)
		}
		if held, _ := repository.UploadSession.HasCompleted(e.ctx, bob.ID, sha); held {
			t.Fatal("引用后上传凭证应被消耗")
		}
		// 已有用户文件后再次上传同一文件秒传。
		if state, _ := Upload.Prepare(e.ctx, bob.ID, sha, int64(len(data))); !state.Exists {
			t.Fatal("已持有的文件不需要上传")
		}

		// 复制卡片带出两个附件，用户引用数加二。
		copied, err := Card.Copy(e.ctx, e.user.ID, card.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if file = userFileOf(t, e.user.ID, blob.ID); file.ReferenceCount != 4 {
			t.Fatalf("复制后用户引用数 = %d", file.ReferenceCount)
		}
		copiedAttachments, _ := repository.Attachment.ListByCards(e.ctx, e.user.ID, []int64{copied.ID})
		if len(copiedAttachments) != 2 {
			t.Fatalf("副本附件数 = %d", len(copiedAttachments))
		}

		// 删除全部附件后用户引用数为 0，记下归零时间；仍有引用时不能在文件管理中删除。
		if err = File.Delete(e.ctx, e.user.ID, file.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("有引用时删除 err = %v", err)
		}
		all, _ := repository.Attachment.ListByCards(e.ctx, e.user.ID, []int64{card.ID, copied.ID})
		for _, attachment := range all {
			if err = Attachment.Delete(e.ctx, e.user.ID, attachment.ID); err != nil {
				t.Fatal(err)
			}
		}
		if file = userFileOf(t, e.user.ID, blob.ID); file.ReferenceCount != 0 || file.ZeroAt == nil {
			t.Fatalf("删除附件后用户文件 = %+v", file)
		}
		if _, err = Attachment.Rename(e.ctx, e.user.ID, first.ID, "x"); errorKind(err) != KindNotFound {
			t.Fatalf("已删除的附件 err = %v", err)
		}

		// 用户在文件管理中删除后全局引用数减一；还有鲍勃引用，文件保留。
		if err = File.Delete(e.ctx, e.user.ID, file.ID); err != nil {
			t.Fatal(err)
		}
		if blob = blobOf(t, sha); blob.ReferenceCount != 1 {
			t.Fatalf("删除用户文件后全局引用数 = %d", blob.ReferenceCount)
		}

		// 删除鲍勃：他的用户文件随之删除，全局引用数降到 0，全局记录和存储中的文件被删除。
		admin, _ := repository.User.FindByUsername(e.ctx, "admin")
		if err = User.Delete(e.ctx, admin.ID, bob.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = repository.Blob.FindBySHA256(e.ctx, sha); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("全局引用数归零后记录仍在: %v", err)
		}
		if _, err = os.Stat(store.BlobPath(sha)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("全局引用数归零后文件仍在: %v", err)
		}
	})
}

func TestImageAttachmentCover(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		store := useStorage(t)
		e := newCardEnv(t, "待办")
		card := e.create("待办", "卡片", listsort.Tail)
		imageSHA := upload(t, e.user.ID, pngData(t, 800, 400))
		textSHA := upload(t, e.user.ID, []byte("说明文字"))

		blob := blobOf(t, imageSHA)
		if !blob.IsImage() || blob.ImageWidth != 800 || blob.ImageHeight != 400 || blob.MimeType != "image/png" {
			t.Fatalf("图片的全局文件 = %+v", blob)
		}
		for _, size := range []int{360, 720} {
			if _, err := os.Stat(store.ThumbnailPath(imageSHA, size, blob.ThumbnailExt)); err != nil {
				t.Fatalf("缺少 %d 缩略图: %v", size, err)
			}
		}

		text, err := Attachment.CreateFile(e.ctx, e.user.ID, card.ID, textSHA, "说明.txt")
		if err != nil {
			t.Fatal(err)
		}
		if e.reload(card.ID).CoverAttachmentID != nil {
			t.Fatal("非图片不应自动成为封面")
		}
		picture, err := Attachment.CreateFile(e.ctx, e.user.ID, card.ID, imageSHA, "图.png")
		if err != nil {
			t.Fatal(err)
		}
		if cover := e.reload(card.ID).CoverAttachmentID; cover == nil || *cover != picture.ID {
			t.Fatal("卡片没有封面时，新图片应自动成为封面")
		}
		if _, err = Attachment.SetCover(e.ctx, e.user.ID, card.ID, &text.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("非图片设为封面 err = %v", err)
		}

		// 复制卡片时封面指向副本中对应的附件。
		copied, err := Card.Copy(e.ctx, e.user.ID, card.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		copiedCover := e.reload(copied.ID).CoverAttachmentID
		if copiedCover == nil || *copiedCover == picture.ID {
			t.Fatalf("副本封面 = %v", copiedCover)
		}
		if attachment, _ := repository.Attachment.FindByID(e.ctx, e.user.ID, *copiedCover); attachment.CardID != copied.ID {
			t.Fatal("副本封面不属于副本")
		}

		// 删除作为封面的附件后，卡片不再有封面。
		if err = Attachment.Delete(e.ctx, e.user.ID, picture.ID); err != nil {
			t.Fatal(err)
		}
		if e.reload(card.ID).CoverAttachmentID != nil {
			t.Fatal("删除封面附件后封面仍在")
		}

		// 归档中的卡片不能添加附件，但已有的附件可以读取。
		if err = Card.Archive(e.ctx, e.user.ID, card.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = Attachment.CreateFile(e.ctx, e.user.ID, card.ID, imageSHA, "x.png"); errorKind(err) != KindBadRequest {
			t.Fatalf("归档卡片添加附件 err = %v", err)
		}
		if _, opened, err := Attachment.Open(e.ctx, e.user.ID, text.ID); err != nil || opened.SHA256 != textSHA {
			t.Fatalf("读取归档卡片的附件 err = %v", err)
		}
	})
}

func TestLinkAttachment(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		useStorage(t)
		e := newCardEnv(t, "待办")
		card := e.create("待办", "卡片", listsort.Tail)
		for _, bad := range []string{"", "ftp://example.com", "javascript:alert(1)", "https://" + strings.Repeat("a", 2050)} {
			if _, err := Attachment.CreateLink(e.ctx, e.user.ID, card.ID, bad, ""); errorKind(err) != KindBadRequest {
				t.Fatalf("网址 %q err = %v", bad, err)
			}
		}
		link, err := Attachment.CreateLink(e.ctx, e.user.ID, card.ID, "https://example.com/docs", "")
		if err != nil || link.Name != "https://example.com/docs" || link.Type != model.AttachmentLink {
			t.Fatalf("链接附件 = %+v, err = %v", link, err)
		}
		// 抓到图标后保存；网址已不同时忽略。
		if err = applyFavicon(e.ctx, e.user.ID, link.ID, "https://other.example", "data:image/png;base64,AA=="); err != nil {
			t.Fatal(err)
		}
		if err = applyFavicon(e.ctx, e.user.ID, link.ID, link.URL, "data:image/png;base64,AA=="); err != nil {
			t.Fatal(err)
		}
		saved, _ := repository.Attachment.FindByID(e.ctx, e.user.ID, link.ID)
		if saved.Favicon != "data:image/png;base64,AA==" {
			t.Fatalf("站点图标 = %q", saved.Favicon)
		}
		views, err := cardBundle(e.ctx, e.user.ID, []*model.Card{card})
		if err != nil || len(views.Attachments) != 1 || views.Attachments[0].Favicon == "" {
			t.Fatalf("卡片内容中的附件 = %+v, err = %v", views.Attachments, err)
		}
	})
}

func TestCleanOrphanFiles(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		store := useStorage(t)
		mustAdmin(t)
		alice := mustUser(t, "alice")
		kept := upload(t, alice.ID, []byte("保留"))
		orphan := strings.Repeat("ab", 32)
		if err := os.MkdirAll(store.BlobPath(orphan)[:len(store.BlobPath(orphan))-65], 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(store.BlobPath(orphan), []byte("孤立"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := store.WriteThumbnail(orphan, 360, "jpg", []byte("x")); err != nil {
			t.Fatal(err)
		}
		if _, err := store.AppendUpload(987654, 0, strings.NewReader("孤立的上传"), 100); err != nil {
			t.Fatal(err)
		}
		if err := CleanOrphanFiles(context.Background()); err != nil {
			t.Fatal(err)
		}
		entries, _ := store.List()
		if len(entries.Blobs) != 1 || entries.Blobs[0] != kept || len(entries.Thumbnails) != 0 || len(entries.Uploads) != 0 {
			t.Fatalf("清理后的存储 = %+v", entries)
		}
	})
}
