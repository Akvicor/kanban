package model

// All 返回需要迁移的全部表模型。migrate 按这个列表创建和升级表结构。
// 表之间的关联不建数据库外键，由服务层维护。
func All() []any {
	return []any{
		&User{},
		&Device{},
		&ChangeLog{},
		&Folder{},
		&Board{},
		&Panel{},
		&Label{},
		&PriorityLevel{},
		&List{},
		&ListLabelRule{},
		&ListTimeRule{},
		&Card{},
		&CardLabel{},
		&Task{},
		&CardAction{},
		&CardLink{},
		&Blob{},
		&UserFile{},
		&UploadSession{},
		&Attachment{},
		&NotifyChannel{},
		&CardNotifyChannel{},
		&NotifyDelivery{},
	}
}
