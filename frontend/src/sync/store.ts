import type {
  Account,
  Attachment,
  Board,
  Card,
  CardAction,
  CardBundle,
  CardLink,
  Device,
  Folder,
  Label,
  List,
  NotifyChannel,
  NotifyDelivery,
  Panel,
  PanelContent,
  PriorityLevel,
  Settings,
  Snapshot,
  SyncEvent,
  Task,
  UserFile,
  UserFileList,
} from '../api/types'

/** 同步的实体类型，与后端 service 中的 Entity* 常量一致。 */
export const EntityType = {
  Settings: 'settings',
  Account: 'account',
  Device: 'device',
  Folder: 'folder',
  Board: 'board',
  Panel: 'panel',
  Label: 'label',
  PriorityLevel: 'priority_level',
  List: 'list',
  Card: 'card',
  Task: 'task',
  CardAction: 'card_action',
  CardLink: 'card_link',
  Attachment: 'attachment',
  UserFile: 'user_file',
  NotifyChannel: 'notify_channel',
  NotifyDelivery: 'notify_delivery',
} as const

/**
 * 当前用户的同步数据。folders、boards、panels 包含已归档的，目录和各级归档由它们得出。
 * 列表和卡片按面板分开加载：loadedPanels 中的面板已经加载完整，其他面板只有加载后收到推送的部分。
 * 卡片归档和列表归档中的卡片在查看时另外加载。附件和通知发送记录随卡片一起加载；通知渠道随快照下发。
 * 用户文件在打开文件管理时加载，userFilesLoaded 表示已经加载完整。
 */
export interface SyncState {
  settings: Settings
  /** 当前用户的账号。快照或账号变更到达后才有，之前用登录时的账号。 */
  account: Account | null
  /** 按最后活跃时间倒序。 */
  devices: Device[]
  folders: Folder[]
  boards: Board[]
  panels: Panel[]
  labels: Label[]
  priorityLevels: PriorityLevel[]
  lists: List[]
  cards: Card[]
  tasks: Task[]
  cardActions: CardAction[]
  cardLinks: CardLink[]
  attachments: Attachment[]
  userFiles: UserFile[]
  notifyChannels: NotifyChannel[]
  notifyDeliveries: NotifyDelivery[]
  loadedPanels: ReadonlySet<number>
  userFilesLoaded: boolean
}

/**
 * 按 ID 存放的实体类型与 SyncState 中对应字段的关系。
 * snapshot 是全量快照中对应的字段；为 null 的类型不在快照中，按面板或按需加载。
 */
const COLLECTIONS = {
  [EntityType.Device]: {field: 'devices', snapshot: 'devices'},
  [EntityType.Folder]: {field: 'folders', snapshot: 'folders'},
  [EntityType.Board]: {field: 'boards', snapshot: 'boards'},
  [EntityType.Panel]: {field: 'panels', snapshot: 'panels'},
  [EntityType.Label]: {field: 'labels', snapshot: 'labels'},
  [EntityType.PriorityLevel]: {field: 'priorityLevels', snapshot: 'priority_levels'},
  [EntityType.List]: {field: 'lists', snapshot: null},
  [EntityType.Card]: {field: 'cards', snapshot: null},
  [EntityType.Task]: {field: 'tasks', snapshot: null},
  [EntityType.CardAction]: {field: 'cardActions', snapshot: null},
  [EntityType.CardLink]: {field: 'cardLinks', snapshot: null},
  [EntityType.Attachment]: {field: 'attachments', snapshot: null},
  [EntityType.UserFile]: {field: 'userFiles', snapshot: null},
  [EntityType.NotifyChannel]: {field: 'notifyChannels', snapshot: 'notify_channels'},
  [EntityType.NotifyDelivery]: {field: 'notifyDeliveries', snapshot: null},
} as const

type CollectionType = keyof typeof COLLECTIONS
type CollectionField = (typeof COLLECTIONS)[CollectionType]['field']
type Entity = {id: number}

/** 设备排序与后端一致：最后活跃时间倒序，相同时 ID 倒序。其他集合的顺序由使用方决定。 */
function sortCollection(field: CollectionField, items: Entity[]): Entity[] {
  if (field === 'devices') {
    return [...(items as Device[])].sort((a, b) => b.last_active_at.localeCompare(a.last_active_at) || b.id - a.id)
  }
  return items
}

function isCollection(type: string): type is CollectionType {
  return type in COLLECTIONS
}

/**
 * 当前用户同步数据的本地副本，供 React 通过 useSyncExternalStore 订阅。
 *
 * 每个实体记下本地数据对应的同步序号。只有序号更大的数据才会覆盖本地数据，因此：
 * - 写入的 HTTP 响应先到时，立即按响应更新，之后推送中同一序号或更早的变更被忽略，不会闪回旧值；
 * - 推送先到时，响应带来的同一序号数据也不会重复覆盖；
 * - 按面板或按需加载的内容晚于推送到达时，也不会覆盖推送带来的较新数据。
 */
export class SyncStore {
  private state: SyncState
  private readonly versions = new Map<string, number>()
  private readonly listeners = new Set<() => void>()

  /** 用登录时拿到的个人设置初始化；其余数据在收到快照后才有。 */
  constructor(settings: Settings) {
    this.state = {
      settings,
      account: null,
      devices: [],
      folders: [],
      boards: [],
      panels: [],
      labels: [],
      priorityLevels: [],
      lists: [],
      cards: [],
      tasks: [],
      cardActions: [],
      cardLinks: [],
      attachments: [],
      userFiles: [],
      notifyChannels: [],
      notifyDeliveries: [],
      loadedPanels: new Set(),
      userFilesLoaded: false,
    }
  }

  getState = (): SyncState => this.state

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  /** 本地实体数据对应的同步序号，没有时为 0。修改标题和描述时作为 base_revision 发送，用于检测编辑冲突。 */
  versionOf(type: string, id: number): number {
    return this.versions.get(key(type, id)) ?? 0
  }

  /**
   * 用快照整体替换本地数据。按面板加载的数据可能漏掉了断线期间的变更，一并清空，
   * 正在查看的面板会重新加载。
   */
  applySnapshot(snapshot: Snapshot, revision: number): void {
    this.versions.clear()
    this.versions.set(key(EntityType.Settings, 0), revision)
    if (snapshot.account) this.versions.set(key(EntityType.Account, snapshot.account.id), revision)
    const collections: Record<string, Entity[]> = {}
    for (const [type, {field, snapshot: source}] of Object.entries(COLLECTIONS)) {
      const items = source === null ? [] : (snapshot[source] as Entity[])
      items.forEach((item) => this.versions.set(key(type, item.id), revision))
      collections[field] = sortCollection(field, items)
    }
    this.setState({
      ...(collections as unknown as SyncState),
      settings: snapshot.settings,
      account: snapshot.account ?? this.state.account,
      loadedPanels: new Set<number>(),
      userFilesLoaded: false,
    })
  }

  /**
   * 应用打开面板时加载的内容，把面板标记为已加载。
   * 本地已有、但内容中没有的列表和卡片，如果本地数据不比内容新，说明它在读取时已不在这个范围内，予以移除。
   * 卡片的范围是内容中不在列表归档里的列表；卡片归档中的卡片不受影响。
   */
  applyPanelContent(content: PanelContent): void {
    const {panel_id: panelId, revision} = content
    const activeListIds = new Set(content.lists.filter((list) => list.archived_at === null).map((list) => list.id))
    let next = this.state
    next = this.merge(next, EntityType.List, content.lists, revision, (list: List) => list.panel_id === panelId)
    next = this.merge(next, EntityType.Card, content.cards, revision, (card: Card) => card.list_id !== null && activeListIds.has(card.list_id))
    next = this.mergeBundleChildren(next, content, revision)
    this.setState({...next, loadedPanels: new Set([...this.state.loadedPanels, panelId])})
  }

  /** 应用按需加载的一组卡片（卡片归档、列表归档中的卡片）。 */
  applyBundle(bundle: CardBundle): void {
    let next = this.merge(this.state, EntityType.Card, bundle.cards, bundle.revision)
    next = this.mergeBundleChildren(next, bundle, bundle.revision)
    this.setState(next)
  }

  /**
   * 应用打开文件管理时加载的用户文件。本地已有、但列表中没有且不比列表新的文件，说明已被删除，予以移除。
   */
  applyUserFiles(list: UserFileList): void {
    const next = this.merge(this.state, EntityType.UserFile, list.files, list.revision, () => true)
    this.setState({...next, userFilesLoaded: true})
  }

  /** 应用一条推送的变更。 */
  applyEvent(event: SyncEvent): void {
    this.apply(event.type, event.id, event.op === 'delete' ? null : event.data, event.revision)
  }

  /** 应用写入接口的响应。data 为 null 表示实体已删除；revision 为 0 表示这次写入没有产生变更。 */
  applyLocal(type: string, id: number, data: unknown, revision: number): void {
    if (revision > 0) {
      this.apply(type, id, data, revision)
    }
  }

  private mergeBundleChildren(state: SyncState, bundle: CardBundle, revision: number): SyncState {
    let next = this.merge(state, EntityType.Task, bundle.tasks, revision)
    next = this.merge(next, EntityType.CardAction, bundle.card_actions, revision)
    next = this.merge(next, EntityType.CardLink, bundle.card_links, revision)
    next = this.merge(next, EntityType.Attachment, bundle.attachments, revision)
    return this.merge(next, EntityType.NotifyDelivery, bundle.notify_deliveries, revision)
  }

  /**
   * 把读取于 revision 时的一组实体合并进本地数据，不覆盖更新的本地数据。
   * inScope 给出时，本地属于该范围、但不在 items 中且不比 revision 新的实体被移除。
   */
  private merge<T extends Entity>(state: SyncState, type: CollectionType, items: T[], revision: number, inScope?: (item: T) => boolean): SyncState {
    const {field} = COLLECTIONS[type]
    const incoming = new Set(items.map((item) => item.id))
    let current = (state[field] as unknown as T[]).filter(
      (item) => !inScope || !inScope(item) || incoming.has(item.id) || this.versionOf(type, item.id) > revision,
    )
    for (const item of items) {
      if (this.versionOf(type, item.id) >= revision) continue
      this.versions.set(key(type, item.id), revision)
      current = [...current.filter((existing) => existing.id !== item.id), item]
    }
    return {...state, [field]: sortCollection(field, current)}
  }

  private apply(type: string, id: number, data: unknown, revision: number): void {
    // 个人设置每个用户只有一份，用固定的 0 作为实体键。
    const entityKey = key(type, type === EntityType.Settings ? 0 : id)
    if ((this.versions.get(entityKey) ?? 0) >= revision) {
      return
    }
    if (type === EntityType.Settings) {
      this.versions.set(entityKey, revision)
      this.setState({...this.state, settings: data as Settings})
      return
    }
    if (type === EntityType.Account && data) {
      this.versions.set(entityKey, revision)
      this.setState({...this.state, account: data as Account})
      return
    }
    if (isCollection(type)) {
      this.versions.set(entityKey, revision)
      const {field} = COLLECTIONS[type]
      const others = (this.state[field] as Entity[]).filter((item) => item.id !== id)
      const next = data === null ? others : [...others, data as Entity]
      this.setState({...this.state, [field]: sortCollection(field, next)})
    }
    // 尚未接入的实体类型直接忽略，序号照常前进。
  }

  private setState(next: SyncState): void {
    this.state = next
    this.listeners.forEach((listener) => listener())
  }
}

function key(type: string, id: number): string {
  return `${type}:${id}`
}
