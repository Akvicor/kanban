package dto

// PrepareUpload 是开始或继续上传的请求：文件的 sha256（64 位小写十六进制）和大小。
type PrepareUpload struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// CreateFileAttachment 是用已上传的文件新建文件附件的请求。
type CreateFileAttachment struct {
	CardID int64  `json:"card_id"`
	SHA256 string `json:"sha256"`
	Name   string `json:"name"`
}

// CreateLinkAttachment 是新建链接附件的请求，Name 为空时取网址。
type CreateLinkAttachment struct {
	CardID int64  `json:"card_id"`
	URL    string `json:"url"`
	Name   string `json:"name"`
}

// RenameAttachment 是修改附件名称的请求。
type RenameAttachment struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// SetCardCover 是设置卡片封面的请求，AttachmentID 为空表示取消封面。
type SetCardCover struct {
	ID           int64  `json:"id"`
	AttachmentID *int64 `json:"attachment_id"`
}
