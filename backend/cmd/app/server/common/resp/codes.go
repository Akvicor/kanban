package resp

// 细粒度业务错误码：接口错误以整数码返回，前端按码取对应语言的文案（src/i18n/errors.*.ts）。
// 0–7 是协议级结果码（见 model.go），行为相关的取值（4 未登录、6 编辑冲突携带数据）保持不变；
// 其余有具体文案的业务错误都用这里的码。前端缺少任何一码的文案即测试失败，两边必须同步修改。
//
// 分段：10xx 认证与会话、11xx 用户与设置、12xx 目录（文件夹/看板）、13xx 面板、14xx 列表、
// 15xx 卡片、16xx 任务、17xx 标签与优先级、18xx 附件/文件/上传、19xx 渠道/设备/通用。
const (
	// 认证与会话
	LoginInvalid        Code = 1001 // 用户名或密码不正确，或账号已停用
	SessionExpired      Code = 1002 // 登录已失效，请重新登录
	NotLoggedIn         Code = 1003 // 请先登录
	AccountDisabled     Code = 1004 // 账号已停用
	AdminRequired       Code = 1005 // 需要管理员权限
	TooManyLoginAttempt Code = 1006 // 登录尝试过于频繁，请稍后再试

	// 用户与设置
	UsernameLength       Code = 1101 // 用户名长度为 1 到 32 个字符
	UsernameWhitespace   Code = 1102 // 用户名不能包含空白字符
	NicknameLength       Code = 1103 // 昵称长度为 1 到 32 个字符
	NicknameControl      Code = 1104 // 昵称不能包含控制字符
	UsernameTaken        Code = 1105 // 用户名已被使用
	PasswordCharset      Code = 1106 // 密码包含无效字符
	PasswordTooShort     Code = 1107 // 密码至少 8 个字符
	PasswordTooLong      Code = 1108 // 密码不能超过 72 字节
	CurrentPasswordWrong Code = 1109 // 当前密码不正确
	TimezoneInvalid      Code = 1110 // 时区不正确
	PaletteInvalid       Code = 1111 // 配色不正确
	LocaleInvalid        Code = 1112 // 语言不正确
	EditorModeInvalid    Code = 1113 // 编辑器模式不正确
	TemplateCharset      Code = 1114 // 正文模板包含无效字符
	TemplateTooLong      Code = 1115 // 正文模板不能超过 4000 个字符
	ShortcutConflict     Code = 1116 // 快捷键已被其他操作使用
	ShortcutInvalid      Code = 1117 // 快捷键不正确
	InitialAdminRequired Code = 1118 // 库中还没有用户，需要提供初始管理员的用户名和密码
	SelfActionForbidden  Code = 1119 // 不能对自己的账号执行此操作
	UserNotFound         Code = 1120 // 用户不存在
	PanModifierInvalid   Code = 1121 // 拖动平移修饰键不正确

	// 目录：文件夹与看板
	NameEmpty            Code = 1201 // 名称不能为空
	NameTooLong          Code = 1202 // 名称不能超过 100 个字符
	FolderDepthLimit     Code = 1203 // 文件夹最多 16 层
	FolderDepthExceeded  Code = 1204 // 移动后文件夹会超过 16 层
	FolderSelfMove       Code = 1205 // 不能把文件夹移到它自己或它的下级中
	FolderArchived       Code = 1206 // 文件夹已归档
	TargetFolderArchived Code = 1207 // 目标文件夹已归档
	FolderNotFound       Code = 1208 // 文件夹不存在
	TargetFolderNotFound Code = 1209 // 目标文件夹不存在
	BoardArchived        Code = 1210 // 看板已归档
	BoardNotArchived     Code = 1211 // 看板不在看板归档中
	BoardNotFound        Code = 1212 // 看板不存在

	// 面板
	PanelNotFound       Code = 1301 // 面板不存在
	PanelArchived       Code = 1302 // 面板已归档
	PanelNotArchived    Code = 1303 // 面板不在归档中
	PanelWrongBoard     Code = 1304 // 面板不属于这个看板
	PanelAlreadyInBoard Code = 1305 // 面板已经在这个看板中
	PanelBoardArchived  Code = 1306 // 面板所在的看板已归档

	// 列表
	ListNotFound          Code = 1401 // 列表不存在
	ListArchived          Code = 1402 // 列表已归档
	ListNotArchived       Code = 1403 // 列表不在列表归档中
	ListSortInvalid       Code = 1404 // 排序方式不正确
	ListTimeActionInvalid Code = 1405 // 时间动作不正确
	ListRuleLabelOutside  Code = 1406 // 操作配置只能使用本面板的标签

	// 卡片
	CardNotFound             Code = 1501 // 卡片不存在
	CardArchived             Code = 1502 // 卡片已归档
	CardNotArchived          Code = 1503 // 卡片不在卡片归档中
	CardTitleEmpty           Code = 1504 // 标题不能为空
	CardTitleTooLong         Code = 1505 // 标题不能超过 500 个字符
	CardDescriptionTooLong   Code = 1506 // 描述不能超过 1048576 个字符
	CardCopyCrossPanel       Code = 1507 // 卡片只能在所属面板内复制
	CardMoveCrossPanel       Code = 1508 // 卡片只能在所属面板的列表之间移动
	CardRestoreWrongPanel    Code = 1509 // 只能恢复到卡片所属面板的列表
	CardPriorityOutsidePanel Code = 1510 // 优先级挡位不属于卡片所在的面板
	CardLabelOutsidePanel    Code = 1511 // 标签不属于卡片所在的面板
	CardTimerActionInvalid   Code = 1512 // 定时器操作不正确
	CardElapsedNegative      Code = 1513 // 累计时间不能为负数
	CardButtonInvalid        Code = 1514 // 创建按钮不正确
	CardLinkTargetInvalid    Code = 1515 // 关联目标必须是一个看板或一个面板
	CardLinkExists           Code = 1516 // 已经关联了这个目标
	CardLinkNotFound         Code = 1517 // 关联不存在

	// 任务
	TaskNotFound        Code = 1601 // 任务不存在
	TaskTitleEmpty      Code = 1602 // 任务标题不能为空
	TaskTitleTooLong    Code = 1603 // 任务标题不能超过 1000 个字符
	TaskDepthLimit      Code = 1604 // 任务最多三层
	TaskParentWrongCard Code = 1605 // 父任务不属于这张卡片
	TaskSelfMove        Code = 1606 // 不能把任务移到它自己或它的下级下面
	TaskBatchLimit      Code = 1607 // 一次最多新建 200 个任务

	// 标签与优先级
	LabelNotFound            Code = 1701 // 标签不存在
	LabelNameTooLong         Code = 1702 // 标签名称不能超过 50 个字符
	PriorityLevelNotFound    Code = 1703 // 优先级挡位不存在
	PriorityLevelNameEmpty   Code = 1704 // 挡位名称不能为空
	PriorityLevelNameTooLong Code = 1705 // 挡位名称不能超过 20 个字符

	// 附件、文件与上传
	AttachmentNotFound      Code = 1801 // 附件不存在
	AttachmentNoFile        Code = 1802 // 附件没有文件
	AttachmentNameEmpty     Code = 1803 // 附件名称不能为空
	AttachmentNameTooLong   Code = 1804 // 附件名称不能超过 128 个字符
	AttachmentCoverNotImage Code = 1805 // 只能把这张卡片上的图片附件设为封面
	AttachmentLinkEmpty     Code = 1806 // 网址不能为空，且不能超过 2048 个字符
	AttachmentLinkInvalid   Code = 1807 // 网址必须以 http:// 或 https:// 开头
	FileNotFound            Code = 1808 // 文件不存在
	FileInUse               Code = 1809 // 文件仍被附件引用，不能删除
	FileDeleted             Code = 1810 // 文件已被删除，请重新上传
	UploadSha256Invalid     Code = 1811 // sha256 格式不正确
	UploadOffsetInvalid     Code = 1812 // 分片偏移不正确
	UploadTooLarge          Code = 1813 // 单个文件不能超过 4GiB
	UploadChecksumMismatch  Code = 1814 // 文件校验失败（sha256 不一致），请重新上传
	UploadOffsetConflict    Code = 1815 // 分片偏移与服务端已收到的大小不一致，请重新查询后继续
	UploadExpired           Code = 1816 // 上传已过期，请重新上传

	// 通知渠道、设备与通用
	ChannelNameInvalid   Code = 1901 // 渠道名字为 1 到 50 个字符
	ChannelAPIInvalid    Code = 1902 // API 必须是 http:// 或 https:// 开头、不带 ? 和 # 的地址
	ChannelSecretEmpty   Code = 1903 // Token 和 Sign 不能为空
	ChannelSecretTooLong Code = 1904 // Token 和 Sign 不能超过 500 个字符
	ChannelFormatInvalid Code = 1905 // 内容格式只能是文本或 Markdown
	ChannelTestFailed    Code = 1906 // 测试发送失败
	ChannelNotFound      Code = 1907 // 通知渠道不存在
	DeviceNotFound       Code = 1908 // 设备不存在
	ColorInvalid         Code = 1909 // 颜色必须是 #RRGGBB 格式
	PositionInvalid      Code = 1910 // 插入位置不正确
	MalformedRequest     Code = 1911 // 请求格式不正确
)
