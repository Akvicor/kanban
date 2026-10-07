package model

import "time"

// Blob 是全局文件：同一 sha256 的内容在存储中只保存一份。它属于系统，不属于任何用户，因此没有 user_id。
// ReferenceCount 是引用它的用户数，即指向它的 UserFile 记录数；降到 0 时删除记录和存储中的文件。
type Blob struct {
	ID             int64  `gorm:"column:id;primaryKey"`
	SHA256         string `gorm:"column:sha256;size:64;not null;uniqueIndex"`
	Size           int64  `gorm:"column:size;not null"`
	MimeType       string `gorm:"column:mime_type;size:128;not null"` // 按文件头识别的类型
	ReferenceCount int64  `gorm:"column:reference_count;not null"`
	// 图片信息：ThumbnailExt 不为空表示成功生成了缩略图，这份文件算图片；宽高已按 EXIF 方向矫正。
	ThumbnailExt string    `gorm:"column:thumbnail_ext;size:8;not null"`
	ImageWidth   int       `gorm:"column:image_width;not null"`
	ImageHeight  int       `gorm:"column:image_height;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*Blob) TableName() string {
	return "blobs"
}

// IsImage 判断文件是否算图片（成功生成了缩略图）。
func (b *Blob) IsImage() bool {
	return b.ThumbnailExt != ""
}

// UserFile 是用户在文件管理中看到的文件，每个用户、每个全局文件一条。
// ReferenceCount 是该用户引用它的附件数；降到 0 时记下 ZeroAt，不自动删除，由用户在文件管理中手动删除。
// Name 是最近一次引用它的附件名称，用于在文件管理中标识文件。
type UserFile struct {
	ID                int64      `gorm:"column:id;primaryKey"`
	UserID            int64      `gorm:"column:user_id;not null;uniqueIndex:idx_user_file_blob"`
	BlobID            int64      `gorm:"column:blob_id;not null;uniqueIndex:idx_user_file_blob;index"`
	Name              string     `gorm:"column:name;type:text;not null"`
	ReferenceCount    int64      `gorm:"column:reference_count;not null"`
	ZeroAt            *time.Time `gorm:"column:zero_at"` // 引用归零时间，再次被引用时清空
	FirstReferencedAt time.Time  `gorm:"column:first_referenced_at;not null"`
}

// TableName 返回表名。
func (*UserFile) TableName() string {
	return "user_files"
}

// UploadSession 是一次分片上传。未完成时，已收到的内容保存在存储的 uploads 目录中，以文件大小为准；
// Received 记录最近一次确认的大小。
// 收齐并校验 sha256 后 Completed 为 true，临时文件已移入存储或丢弃，会话保留为该用户持有这份内容的凭证：
// 用户凭它引用全局文件（见 Attachment.CreateFile），引用后删除。会话长时间没有更新时删除。
type UploadSession struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	SHA256    string    `gorm:"column:sha256;size:64;not null"`
	Size      int64     `gorm:"column:size;not null"`
	Received  int64     `gorm:"column:received;not null"`
	Completed bool      `gorm:"column:completed;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;index"` // 最后一次收到分片或完成上传的时间，用于清理过期会话
}

// TableName 返回表名。
func (*UploadSession) TableName() string {
	return "upload_sessions"
}

// 附件类型。
const (
	AttachmentFile = "file" // 文件附件，引用用户文件
	AttachmentLink = "link" // 链接附件，保存网址和站点图标
)

// Attachment 是卡片上的附件。文件附件通过 UserFileID 引用用户文件；链接附件保存网址，
// Favicon 是抓取到的站点图标（32×32 PNG 的 data URL），没有抓到时为空。附件按添加顺序（ID）排列。
type Attachment struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	UserID     int64     `gorm:"column:user_id;not null;index"`
	CardID     int64     `gorm:"column:card_id;not null;index"`
	Type       string    `gorm:"column:type;size:8;not null"`
	Name       string    `gorm:"column:name;type:text;not null"`
	UserFileID *int64    `gorm:"column:user_file_id;index"`
	URL        string    `gorm:"column:url;type:text;not null"`
	Favicon    string    `gorm:"column:favicon;type:text;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*Attachment) TableName() string {
	return "attachments"
}
