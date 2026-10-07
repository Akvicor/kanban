package service

import (
	"context"
	"encoding/base64"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/favicon"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/safehttp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Akvicor/glog"
	"gorm.io/gorm"
)

const (
	// attachmentNameMaxLength 是附件名称的最大字符数，与参考项目一致。
	attachmentNameMaxLength = 128
	// linkURLMaxLength 是链接附件网址的最大字符数。
	linkURLMaxLength = 2048
	// faviconTimeout 是抓取一个站点图标的总时限（首页加图标，最多三次请求）。
	faviconTimeout = 20 * time.Second
	// faviconRequestTimeout 是抓取图标时单次请求的时限。
	faviconRequestTimeout = 4 * time.Second
)

// Attachment 是卡片附件的服务：新建文件附件和链接附件、重命名、删除、设为或取消封面，以及按附件读取文件。
// 卡片在归档中时附件只能查看和下载。
var Attachment = new(attachmentService)

type attachmentService struct{}

// faviconClient 抓取站点图标时使用的客户端，只允许连接公网地址。
var faviconClient = safehttp.New(safehttp.PublicOnly, faviconRequestTimeout)

func findAttachment(ctx context.Context, userID, id int64) (*model.Attachment, error) {
	attachment, err := repository.Attachment.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.AttachmentNotFound, "附件不存在")
	}
	return attachment, err
}

func validateAttachmentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", badRequest(resp.AttachmentNameEmpty, "附件名称不能为空")
	}
	if utf8.RuneCountInString(name) > attachmentNameMaxLength {
		return "", badRequest(resp.AttachmentNameTooLong, "附件名称不能超过 128 个字符")
	}
	return name, nil
}

// truncateName 把文字截断为附件名称允许的长度。
func truncateName(name string) string {
	if utf8.RuneCountInString(name) <= attachmentNameMaxLength {
		return name
	}
	return string([]rune(name)[:attachmentNameMaxLength])
}

func validateLinkURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || utf8.RuneCountInString(raw) > linkURLMaxLength {
		return "", badRequest(resp.AttachmentLinkEmpty, "网址不能为空，且不能超过 2048 个字符")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", badRequest(resp.AttachmentLinkInvalid, "网址必须以 http:// 或 https:// 开头")
	}
	return raw, nil
}

// attachmentViews 生成附件信息，文件附件带上它引用的文件信息。
func attachmentViews(ctx context.Context, userID int64, attachments []*model.Attachment) ([]dro.Attachment, error) {
	var fileIDs []int64
	for _, attachment := range attachments {
		if attachment.UserFileID != nil {
			fileIDs = append(fileIDs, *attachment.UserFileID)
		}
	}
	files, err := repository.UserFile.ListByIDs(ctx, userID, fileIDs)
	if err != nil {
		return nil, err
	}
	blobIDs := make([]int64, 0, len(files))
	for _, file := range files {
		blobIDs = append(blobIDs, file.BlobID)
	}
	blobs, err := repository.Blob.ListByIDs(ctx, blobIDs)
	if err != nil {
		return nil, err
	}
	views := make([]dro.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		var file *model.UserFile
		var blob *model.Blob
		if attachment.UserFileID != nil {
			file = files[*attachment.UserFileID]
			if file != nil {
				blob = blobs[file.BlobID]
			}
		}
		views = append(views, dro.NewAttachment(attachment, file, blob))
	}
	return views, nil
}

func attachmentView(ctx context.Context, attachment *model.Attachment) (dro.Attachment, error) {
	views, err := attachmentViews(ctx, attachment.UserID, []*model.Attachment{attachment})
	if err != nil {
		return dro.Attachment{}, err
	}
	return views[0], nil
}

func recordAttachment(ctx context.Context, attachment *model.Attachment) error {
	view, err := attachmentView(ctx, attachment)
	if err != nil {
		return err
	}
	return recordChange(ctx, attachment.UserID, hub.OpUpsert, EntityAttachment, attachment.ID, view)
}

// CreateFile 用已上传的文件（按 sha256）新建文件附件。卡片没有封面且这是一张图片时，自动设为封面。
// 用户必须持有这份文件（见 heldBlob）：凭上传凭证引用时消耗凭证；没有持有时与文件不存在返回同样的错误。
func (s *attachmentService) CreateFile(ctx context.Context, userID, cardID int64, sha, name string) (*model.Attachment, error) {
	name, err := validateAttachmentName(name)
	if err != nil {
		return nil, err
	}
	var attachment *model.Attachment
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		card, _, err := usableCard(ctx, userID, cardID)
		if err != nil {
			return err
		}
		blob, viaProof, err := heldBlob(ctx, userID, sha)
		if err != nil {
			return err
		}
		if blob == nil {
			return errFileGone()
		}
		now := nowUTC()
		file, err := referenceBlob(ctx, userID, blob, name, now)
		if err != nil {
			return err
		}
		if viaProof {
			if err = repository.UploadSession.DeleteCompleted(ctx, userID, sha); err != nil {
				return err
			}
		}
		attachment = &model.Attachment{UserID: userID, CardID: card.ID, Type: model.AttachmentFile, Name: name, UserFileID: &file.ID, CreatedAt: now}
		if err = repository.Attachment.Create(ctx, attachment); err != nil {
			return err
		}
		if err = recordAttachment(ctx, attachment); err != nil {
			return err
		}
		if card.CoverAttachmentID == nil && blob.IsImage() {
			return setCover(ctx, card, &attachment.ID)
		}
		return nil
	})
	return attachment, err
}

// CreateLink 新建链接附件，名称为空时取网址。提交后在后台抓取站点图标，抓到后更新附件并推送。
func (s *attachmentService) CreateLink(ctx context.Context, userID, cardID int64, rawURL, name string) (*model.Attachment, error) {
	link, err := validateLinkURL(rawURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = truncateName(link)
	}
	if name, err = validateAttachmentName(name); err != nil {
		return nil, err
	}
	var attachment *model.Attachment
	err = write(ctx, func(ctx context.Context) error {
		card, _, err := usableCard(ctx, userID, cardID)
		if err != nil {
			return err
		}
		attachment = &model.Attachment{UserID: userID, CardID: card.ID, Type: model.AttachmentLink, Name: name, URL: link, CreatedAt: nowUTC()}
		if err = repository.Attachment.Create(ctx, attachment); err != nil {
			return err
		}
		if err = recordAttachment(ctx, attachment); err != nil {
			return err
		}
		id := attachment.ID
		return afterCommit(ctx, func() { go fetchFavicon(userID, id, link) })
	})
	return attachment, err
}

// fetchFavicon 抓取站点图标并保存到链接附件。抓取失败只记录调试日志。
// 它在独立的 goroutine 中运行，解码的是外部网站提供的任意数据：这里捕获 panic 并记录，
// 附件保持没有图标的状态，避免单个畸形图标让整个进程退出。
func fetchFavicon(userID, attachmentID int64, link string) {
	defer func() {
		if r := recover(); r != nil {
			glog.Error("抓取附件 %d 的站点图标时发生异常: %v", attachmentID, r)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), faviconTimeout)
	defer cancel()
	data, err := favicon.Fetch(ctx, faviconClient, link)
	if err != nil {
		glog.Debug("抓取站点图标 %s 失败: %v", link, err)
		return
	}
	if err = applyFavicon(ctx, userID, attachmentID, link, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(data)); err != nil {
		glog.Warning("保存站点图标失败: %v", err)
	}
}

// applyFavicon 保存抓取到的站点图标。附件已删除或网址已不同时不做任何事。
func applyFavicon(ctx context.Context, userID, attachmentID int64, link, dataURL string) error {
	return write(ctx, func(ctx context.Context) error {
		attachment, err := repository.Attachment.FindByID(ctx, userID, attachmentID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && attachment.URL != link) {
			return nil
		}
		if err != nil {
			return err
		}
		attachment.Favicon = dataURL
		if err = repository.Attachment.Update(ctx, userID, attachmentID, map[string]any{"favicon": dataURL}); err != nil {
			return err
		}
		return recordAttachment(ctx, attachment)
	})
}

// usableAttachment 查找可以修改的附件：附件所在卡片可以修改。返回附件和卡片。
func usableAttachment(ctx context.Context, userID, id int64) (*model.Attachment, *model.Card, error) {
	attachment, err := findAttachment(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	card, _, err := usableCard(ctx, userID, attachment.CardID)
	if err != nil {
		return nil, nil, err
	}
	return attachment, card, nil
}

// Rename 修改附件名称。文件附件的用户文件名称同步为新名称。
func (s *attachmentService) Rename(ctx context.Context, userID, id int64, name string) (*model.Attachment, error) {
	name, err := validateAttachmentName(name)
	if err != nil {
		return nil, err
	}
	var attachment *model.Attachment
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if attachment, _, err = usableAttachment(ctx, userID, id); err != nil {
			return err
		}
		attachment.Name = name
		if err = repository.Attachment.Update(ctx, userID, id, map[string]any{"name": name}); err != nil {
			return err
		}
		if attachment.UserFileID != nil {
			file, err := repository.UserFile.FindByID(ctx, userID, *attachment.UserFileID)
			if err != nil {
				return err
			}
			if err = changeUserFileReference(ctx, file, 0, name, nowUTC()); err != nil {
				return err
			}
		}
		return recordAttachment(ctx, attachment)
	})
	return attachment, err
}

// Delete 删除附件，文件附件的用户引用数减一。删除的是封面时，卡片不再有封面。
func (s *attachmentService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		attachment, card, err := usableAttachment(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = repository.Attachment.Delete(ctx, userID, id); err != nil {
			return err
		}
		if err = recordChange(ctx, userID, hub.OpDelete, EntityAttachment, id, nil); err != nil {
			return err
		}
		if attachment.UserFileID != nil {
			file, err := repository.UserFile.FindByID(ctx, userID, *attachment.UserFileID)
			if err != nil {
				return err
			}
			if err = changeUserFileReference(ctx, file, -1, "", nowUTC()); err != nil {
				return err
			}
		}
		if card.CoverAttachmentID != nil && *card.CoverAttachmentID == id {
			return setCover(ctx, card, nil)
		}
		return nil
	})
}

// SetCover 设置或取消卡片封面。attachmentID 为 nil 时取消；否则必须是这张卡片上的图片文件附件。
func (s *attachmentService) SetCover(ctx context.Context, userID, cardID int64, attachmentID *int64) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, cardID); err != nil {
			return err
		}
		if attachmentID != nil {
			attachment, err := findAttachment(ctx, userID, *attachmentID)
			if err != nil {
				return err
			}
			if attachment.CardID != card.ID || attachment.UserFileID == nil {
				return badRequest(resp.AttachmentCoverNotImage, "只能把这张卡片上的图片附件设为封面")
			}
			views, err := attachmentViews(ctx, userID, []*model.Attachment{attachment})
			if err != nil {
				return err
			}
			if views[0].File == nil || !views[0].File.Image {
				return badRequest(resp.AttachmentCoverNotImage, "只能把这张卡片上的图片附件设为封面")
			}
		}
		return setCover(ctx, card, attachmentID)
	})
	return card, err
}

func setCover(ctx context.Context, card *model.Card, attachmentID *int64) error {
	card.CoverAttachmentID = attachmentID
	if err := repository.Card.Update(ctx, card.UserID, card.ID, map[string]any{"cover_attachment_id": attachmentID}); err != nil {
		return err
	}
	return recordCard(ctx, card)
}

// copyAttachments 把原卡片的附件复制到副本：新建附件记录，引用同一份文件，用户引用数相应增加。
// 返回原附件 ID 到副本附件 ID 的对应，用于映射封面。
func copyAttachments(ctx context.Context, userID, fromCardID, toCardID int64, now time.Time) (map[int64]int64, error) {
	attachments, err := repository.Attachment.ListByCards(ctx, userID, []int64{fromCardID})
	if err != nil {
		return nil, err
	}
	mapping := map[int64]int64{}
	for _, attachment := range attachments {
		if attachment.UserFileID != nil {
			file, err := repository.UserFile.FindByID(ctx, userID, *attachment.UserFileID)
			if err != nil {
				return nil, err
			}
			if err = changeUserFileReference(ctx, file, 1, attachment.Name, now); err != nil {
				return nil, err
			}
		}
		copied := &model.Attachment{
			UserID: userID, CardID: toCardID, Type: attachment.Type, Name: attachment.Name,
			UserFileID: attachment.UserFileID, URL: attachment.URL, Favicon: attachment.Favicon, CreatedAt: now,
		}
		if err = repository.Attachment.Create(ctx, copied); err != nil {
			return nil, err
		}
		if err = recordAttachment(ctx, copied); err != nil {
			return nil, err
		}
		mapping[attachment.ID] = copied.ID
	}
	return mapping, nil
}

// Open 返回文件附件和它引用的全局文件，用于下载和查看缩略图。归档中的卡片的附件也可以查看。
func (s *attachmentService) Open(ctx context.Context, userID, id int64) (*model.Attachment, *model.Blob, error) {
	attachment, err := findAttachment(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	if attachment.UserFileID == nil {
		return nil, nil, notFound(resp.AttachmentNoFile, "附件没有文件")
	}
	_, blob, err := File.Open(ctx, userID, *attachment.UserFileID)
	if err != nil {
		return nil, nil, err
	}
	return attachment, blob, nil
}

// View 返回附件信息，用于写入后的响应。
func (s *attachmentService) View(ctx context.Context, attachment *model.Attachment) (dro.Attachment, error) {
	return attachmentView(ctx, attachment)
}
