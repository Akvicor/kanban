import {useEffect, useMemo, useState, useSyncExternalStore} from 'react'
import {errorMessage} from '../api/client'
import {fetchPanelContent} from '../api/list'
import {fetchUserFiles} from '../api/file'
import type {
  Attachment,
  Board,
  Card,
  CardAction,
  CardLink,
  Device,
  Folder,
  Label,
  List,
  NotifyChannel,
  NotifyDelivery,
  Panel,
  PriorityLevel,
  Settings,
  Task,
  UserFile,
} from '../api/types'
import {useSession} from '../session/context'
import type {SyncStore} from './store'
import {t} from '../i18n'

/** 返回当前用户的同步数据。只能在需要登录的页面中使用。 */
export function useSyncStore(): SyncStore {
  const {sync} = useSession()
  if (!sync) {
    throw new Error(t('session.notSignedIn'))
  }
  return sync
}

/** 当前用户的个人设置，其他设备修改后自动更新。 */
export function useSettings(): Settings {
  const store = useSyncStore()
  return useSyncExternalStore(store.subscribe, () => store.getState().settings)
}

/** 当前用户的登录设备，最近活跃的在前。 */
export function useDevices(): Device[] {
  const store = useSyncStore()
  return useSyncExternalStore(store.subscribe, () => store.getState().devices)
}

/** 当前用户的全部面板（包括已归档的）、标签和优先级挡位。 */
export function usePanelData(): {panels: Panel[]; labels: Label[]; priorityLevels: PriorityLevel[]} {
  const store = useSyncStore()
  const panels = useSyncExternalStore(store.subscribe, () => store.getState().panels)
  const labels = useSyncExternalStore(store.subscribe, () => store.getState().labels)
  const priorityLevels = useSyncExternalStore(store.subscribe, () => store.getState().priorityLevels)
  return useMemo(() => ({panels, labels, priorityLevels}), [panels, labels, priorityLevels])
}

/** 正在加载内容的面板，避免同一面板重复请求。 */
const loadingPanels = new WeakMap<SyncStore, Set<number>>()

/**
 * 面板的列表（包括列表归档中的）。面板内容没有加载过时，发起一次加载；loaded 为 false 时数据还不完整。
 * 加载失败时返回错误信息，再次打开面板时重试。
 */
export function usePanelLists(panelId: number): {lists: List[]; loaded: boolean; error: string} {
  const store = useSyncStore()
  const lists = useSyncExternalStore(store.subscribe, () => store.getState().lists)
  const loaded = useSyncExternalStore(store.subscribe, () => store.getState().loadedPanels.has(panelId))
  const [failure, setFailure] = useState<{panelId: number; message: string} | null>(null)

  useEffect(() => {
    if (loaded) return
    const loading = loadingPanels.get(store) ?? new Set<number>()
    loadingPanels.set(store, loading)
    if (loading.has(panelId)) return
    loading.add(panelId)
    fetchPanelContent(panelId)
      .then((content) => store.applyPanelContent(content))
      .catch((err: unknown) => setFailure({panelId, message: errorMessage(err)}))
      .finally(() => loading.delete(panelId))
  }, [store, panelId, loaded])

  const panelLists = useMemo(() => lists.filter((list) => list.panel_id === panelId), [lists, panelId])
  return {lists: panelLists, loaded, error: failure?.panelId === panelId && !loaded ? failure.message : ''}
}

/** 已加载的卡片、任务、操作记录、关联、附件和通知发送记录。 */
export function useCardData(): {
  cards: Card[]
  tasks: Task[]
  cardActions: CardAction[]
  cardLinks: CardLink[]
  attachments: Attachment[]
  notifyDeliveries: NotifyDelivery[]
} {
  const store = useSyncStore()
  const cards = useSyncExternalStore(store.subscribe, () => store.getState().cards)
  const tasks = useSyncExternalStore(store.subscribe, () => store.getState().tasks)
  const cardActions = useSyncExternalStore(store.subscribe, () => store.getState().cardActions)
  const cardLinks = useSyncExternalStore(store.subscribe, () => store.getState().cardLinks)
  const attachments = useSyncExternalStore(store.subscribe, () => store.getState().attachments)
  const notifyDeliveries = useSyncExternalStore(store.subscribe, () => store.getState().notifyDeliveries)
  return useMemo(
    () => ({cards, tasks, cardActions, cardLinks, attachments, notifyDeliveries}),
    [cards, tasks, cardActions, cardLinks, attachments, notifyDeliveries],
  )
}

/** 当前用户的通知渠道，按创建顺序。 */
export function useNotifyChannels(): NotifyChannel[] {
  const store = useSyncStore()
  const channels = useSyncExternalStore(store.subscribe, () => store.getState().notifyChannels)
  return useMemo(() => [...channels].sort((a, b) => a.id - b.id), [channels])
}

/**
 * 用户的全部文件，用于文件管理。没有加载过时发起一次加载，之后随推送更新；收到全量快照后重新加载。
 */
export function useUserFiles(): {files: UserFile[]; loaded: boolean; error: string} {
  const store = useSyncStore()
  const files = useSyncExternalStore(store.subscribe, () => store.getState().userFiles)
  const loaded = useSyncExternalStore(store.subscribe, () => store.getState().userFilesLoaded)
  const [error, setError] = useState('')

  useEffect(() => {
    if (loaded) return
    let cancelled = false
    fetchUserFiles()
      .then((list) => !cancelled && store.applyUserFiles(list))
      .catch((err: unknown) => !cancelled && setError(errorMessage(err)))
    return () => {
      cancelled = true
    }
  }, [store, loaded])

  return {files, loaded, error: loaded ? '' : error}
}

/** 当前用户的全部文件夹和看板，包括已归档的。 */
export function useDirectoryData(): {folders: Folder[]; boards: Board[]} {
  const store = useSyncStore()
  const folders = useSyncExternalStore(store.subscribe, () => store.getState().folders)
  const boards = useSyncExternalStore(store.subscribe, () => store.getState().boards)
  return useMemo(() => ({folders, boards}), [folders, boards])
}
