/** 后端接口的数据结构，字段与后端 app/dro 一致。时间都是 RFC 3339 字符串。 */

import type {PalettePreference} from '../theme/palette'

export type Role = 'admin' | 'user'

/** 快捷键绑定：操作 ID 到按键列表。按键格式见 src/shortcuts/keys.ts。 */
export type ShortcutBindings = Record<string, string[]>

export interface Account {
  id: number
  username: string
  /** 显示名。没有单独设置时与用户名相同。 */
  nickname?: string
  role: Role
}

export interface Settings {
  timezone: string
  palette: PalettePreference
  /** 界面与通知语言；空值表示跟随系统（按浏览器语言解析）。 */
  locale: string
  /** 主看板，没有设置时为 null。 */
  main_board_id: number | null
  open_main_board_on_home: boolean
  remind_template: string
  due_template: string
  /** 合并默认值后实际生效的完整绑定。 */
  shortcuts: ShortcutBindings
  /** 卡片描述编辑器上次使用的模式：所见即所得或 Markdown 源码。 */
  editor_mode: EditorMode
  /** 按住修饰键拖动平移看板使用的修饰键；空值表示关闭手势。 */
  pan_modifier: string
}

/** 卡片描述编辑器的模式，与后端 common/types/editormode 一致。 */
export type EditorMode = 'wysiwyg' | 'markup'

export interface Me {
  account: Account
  settings: Settings
  device_id: number
  shortcut_defaults: ShortcutBindings
  /** 全部快捷键操作，顺序即设置界面中的展示顺序。 */
  shortcut_actions: string[]
}

export interface LoginResult {
  token: string
  me: Me
}

/** 登录设备。与 Me.device_id 相同的是当前设备。 */
export interface Device {
  id: number
  name: string
  created_at: string
  last_active_at: string
}

export interface AdminUser {
  id: number
  username: string
  nickname?: string
  role: Role
  created_at: string
  disabled_at: string | null
}

/** 个人设置修改，缺省的字段不修改。shortcuts 只需包含要修改的操作。 */
export type SettingsChanges = Partial<Settings>

/** 目录中的文件夹。parent_id 为 null 表示位于根；archived_at 不为空表示在看板归档中。 */
export interface Folder {
  id: number
  parent_id: number | null
  name: string
  position: number
  archived_at: string | null
}

/** 目录中的看板。archived_at 不为空表示在看板归档中，此时 folder_id 是归档前所在的文件夹。 */
export interface Board {
  id: number
  folder_id: number | null
  name: string
  position: number
  /** 打开看板时进入的面板，为 null 时进入第一个面板。 */
  main_panel_id: number | null
  archived_at: string | null
  created_at: string
}

/** 看板中的面板。archived_at 不为空表示在面板归档中，此时 board_id 是归档前所在的看板。 */
export interface Panel {
  id: number
  board_id: number
  name: string
  position: number
  archived_at: string | null
}

/** 面板中的标签。name 可以为空，此时只显示颜色。 */
export interface Label {
  id: number
  panel_id: number
  name: string
  color: string
  position: number
}

/** 面板中的优先级挡位，position 越小优先级越高。 */
export interface PriorityLevel {
  id: number
  panel_id: number
  name: string
  color: string
  position: number
}

export type SortMode = 'manual' | 'title' | 'priority' | 'remind_at' | 'due_at' | 'created_at' | 'started_at' | 'completed_at'
export type SortDir = 'asc' | 'desc'
/** 列表的一端：列首或列尾。 */
export type ListEnd = 'head' | 'tail'
/** 列表操作对时间字段的动作，空字符串表示不影响。 */
export type TimeAction = '' | 'set' | 'update' | 'clear'
/** 卡片相对列表的操作：创建、移入、移出。 */
export type ListOp = 'create' | 'enter' | 'exit'

/** 一种操作的配置：增加和删除的标签，以及开始时间、完成时间的动作。 */
export interface OpRules {
  add_labels: number[]
  remove_labels: number[]
  start: TimeAction
  complete: TimeAction
}

export type ListRules = Record<ListOp, OpRules>

/** 面板中的列表，连同操作配置。archived_at 不为空表示在列表归档中。color 为空表示没有颜色。 */
export interface List {
  id: number
  panel_id: number
  name: string
  color: string
  position: number
  show_age: boolean
  sort_mode: SortMode
  sort_dir: SortDir
  head_add: ListEnd
  tail_add: ListEnd
  remind_off: boolean
  due_off: boolean
  rules: ListRules
  archived_at: string | null
}

/**
 * 卡片。在列表中时 list_id 和 position（隐藏序号）有值；在卡片归档中时二者为 null，archived_at 有值。
 * 时间都是 RFC 3339 字符串。
 */
export interface Card {
  id: number
  panel_id: number
  list_id: number | null
  position: number | null
  title: string
  description: string
  label_ids: number[]
  priority_level_id: number | null
  remind_at: string | null
  due_at: string | null
  remind_notify: boolean
  due_notify: boolean
  created_at: string
  started_at: string | null
  completed_at: string | null
  timer_seconds: number
  timer_started_at: string | null
  archived_at: string | null
  /** 作为封面的图片附件，null 表示没有封面。 */
  cover_attachment_id: number | null
  /** 卡片选择的通知渠道，提醒和截止共用。 */
  notify_channel_ids: number[]
}

/** 通知渠道的内容格式，对应 gmsg 的 text 和 markdown。 */
export type NotifyFormat = 'text' | 'markdown'

/** 通知渠道。Token 和 Sign 不回显，token_set、sign_set 表示是否已设置。 */
export interface NotifyChannel {
  id: number
  name: string
  api: string
  format: NotifyFormat
  token_set: boolean
  sign_set: boolean
  created_at: string
}

/** 通知类型：到达提醒时间、到达截止时间。 */
export type NotifyKind = 'remind' | 'due'

/**
 * 一张卡片的一种通知在一条渠道上的发送记录。target_at 是针对的时间值；sent_at 不为 null 表示已发送；
 * 否则 attempts 大于 0 表示发送失败，next_attempt_at 是下次重试时间，last_error 是最后一次错误。
 */
export interface NotifyDelivery {
  id: number
  card_id: number
  kind: NotifyKind
  channel_id: number
  target_at: string
  sent_at: string | null
  attempts: number
  next_attempt_at: string
  last_error: string
}

/**
 * 文件信息，来自服务端的全局文件记录。mime_type 按文件内容识别；
 * image 为 true 表示生成了缩略图，可以设为封面、进入画廊，width、height 是按方向矫正后的尺寸。
 */
export interface FileInfo {
  size: number
  mime_type: string
  image: boolean
  width: number
  height: number
}

/**
 * 卡片上的附件。type 为 file 时 file 有值，内容和缩略图通过文件接口按附件 ID 访问；
 * type 为 link 时 url 有值，favicon 是站点图标的 data URL，没有抓到时为空。
 */
export interface Attachment {
  id: number
  card_id: number
  type: 'file' | 'link'
  name: string
  file: (FileInfo & {user_file_id: number}) | null
  url: string
  favicon: string
  created_at: string
}

/** 文件管理中的一个文件。name 是最近一次引用它的附件名称；zero_at 是引用数归零的时间，被引用时为 null。 */
export interface UserFile extends FileInfo {
  id: number
  name: string
  reference_count: number
  zero_at: string | null
  first_referenced_at: string
}

/** 用户的全部文件，revision 是读取时的同步序号。 */
export interface UserFileList {
  revision: number
  files: UserFile[]
}

/**
 * 上传的状态：exists 为 true 表示系统中已有这份文件，不需要上传；否则从 received 继续上传到 session_id。
 * done 为 true 表示全部收齐并校验通过。
 */
export interface UploadState {
  exists: boolean
  session_id: number
  received: number
  done: boolean
}

/** 卡片中的任务，parent_id 为 null 表示第一层。 */
export interface Task {
  id: number
  card_id: number
  parent_id: number | null
  title: string
  done: boolean
  position: number
}

export type CardActionType = 'create' | 'move' | 'task_complete' | 'task_uncomplete' | 'archive' | 'restore'

/** 卡片的一条操作记录。data 的内容随类型不同，见后端 service 中的 addAction 调用，展示见 card/detail/ActionLog.tsx。 */
export interface CardAction {
  id: number
  card_id: number
  type: CardActionType
  data: Record<string, unknown>
  created_at: string
}

/** 卡片上的关联，board_id 和 panel_id 只有一个有值。 */
export interface CardLink {
  id: number
  card_id: number
  board_id: number | null
  panel_id: number | null
  position: number
}

/** 一组卡片连同任务、操作记录、关联和附件，revision 是读取时的同步序号。 */
export interface CardBundle {
  revision: number
  cards: Card[]
  tasks: Task[]
  card_actions: CardAction[]
  card_links: CardLink[]
  attachments: Attachment[]
  notify_deliveries: NotifyDelivery[]
}

/**
 * 打开面板时一次加载的内容：全部列表（包括列表归档中的），以及不在列表归档中的列表里的卡片
 * （不含卡片归档）连同任务、操作记录、关联和附件。revision 是读取时的同步序号。
 */
export interface PanelContent extends CardBundle {
  panel_id: number
  lists: List[]
}

/** 同步的全量快照，与后端 dro.Snapshot 一致。folders、boards、panels 包含已归档的。 */
export interface Snapshot {
  /** 当前用户的账号。旧快照可能没有这一项。 */
  account?: Account
  settings: Settings
  devices: Device[]
  folders: Folder[]
  boards: Board[]
  panels: Panel[]
  labels: Label[]
  priority_levels: PriorityLevel[]
  notify_channels: NotifyChannel[]
}

/** 同步推送的一条变更，与后端 hub.Event 一致。data 是实体变更后的完整数据，删除时为 null。 */
export interface SyncEvent {
  revision: number
  op: 'upsert' | 'delete'
  type: string
  id: number
  data: unknown
}

/** 同步连接上服务端发送的消息，见后端 app/ws/protocol.go。 */
export type SyncMessage =
  | {type: 'snapshot'; revision: number; snapshot: Snapshot}
  | {type: 'events'; events: SyncEvent[]}
  | {type: 'caught_up'; revision: number}
