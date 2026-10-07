package dro

import (
	"kanban/cmd/app/server/model"
	"time"
)

// FileInfo 是文件附件和用户文件共有的文件信息，来自全局文件记录。
// Image 为 true 表示成功生成了缩略图，可以设为封面、进入画廊；Width、Height 是按方向矫正后的尺寸。
type FileInfo struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Image    bool   `json:"image"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// NewFileInfo 从全局文件记录生成文件信息。
func NewFileInfo(blob *model.Blob) FileInfo {
	return FileInfo{Size: blob.Size, MimeType: blob.MimeType, Image: blob.IsImage(), Width: blob.ImageWidth, Height: blob.ImageHeight}
}

// AttachmentFile 是文件附件引用的文件。
type AttachmentFile struct {
	UserFileID int64 `json:"user_file_id"`
	FileInfo
}

// Attachment 是卡片上的附件。文件附件的 File 有值；链接附件的 URL 有值，Favicon 是站点图标的 data URL，没有时为空。
// 文件内容和缩略图通过文件接口按附件 ID 访问。
type Attachment struct {
	ID        int64           `json:"id"`
	CardID    int64           `json:"card_id"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	File      *AttachmentFile `json:"file"`
	URL       string          `json:"url"`
	Favicon   string          `json:"favicon"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewAttachment 生成附件信息。文件附件需要它引用的用户文件和全局文件，链接附件两者传 nil。
func NewAttachment(attachment *model.Attachment, file *model.UserFile, blob *model.Blob) Attachment {
	view := Attachment{
		ID: attachment.ID, CardID: attachment.CardID, Type: attachment.Type, Name: attachment.Name,
		URL: attachment.URL, Favicon: attachment.Favicon, CreatedAt: attachment.CreatedAt,
	}
	if file != nil && blob != nil {
		view.File = &AttachmentFile{UserFileID: file.ID, FileInfo: NewFileInfo(blob)}
	}
	return view
}

// UserFile 是文件管理中的一个文件。Name 是最近一次引用它的附件名称；ZeroAt 是引用数归零的时间，被引用时为空。
type UserFile struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	ReferenceCount    int64      `json:"reference_count"`
	ZeroAt            *time.Time `json:"zero_at"`
	FirstReferencedAt time.Time  `json:"first_referenced_at"`
	FileInfo
}

// NewUserFile 从用户文件和它引用的全局文件生成文件信息。
func NewUserFile(file *model.UserFile, blob *model.Blob) UserFile {
	return UserFile{
		ID: file.ID, Name: file.Name, ReferenceCount: file.ReferenceCount, ZeroAt: file.ZeroAt,
		FirstReferencedAt: file.FirstReferencedAt, FileInfo: NewFileInfo(blob),
	}
}

// UserFileList 是用户的全部文件，以及读取时的同步序号。
type UserFileList struct {
	Revision int64      `json:"revision"`
	Files    []UserFile `json:"files"`
}

// UploadState 是上传的状态：Exists 为 true 表示系统中已有这份文件，不需要上传；
// 否则 SessionID 是上传会话，Received 是服务端已收到的字节数，从这里继续上传。
// Done 为 true 表示全部收齐并校验通过，可以用 sha256 新建文件附件。
type UploadState struct {
	Exists    bool  `json:"exists"`
	SessionID int64 `json:"session_id"`
	Received  int64 `json:"received"`
	Done      bool  `json:"done"`
}
