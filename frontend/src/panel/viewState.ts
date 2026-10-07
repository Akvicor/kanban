import {useCallback, useState} from 'react'
import {EMPTY_FILTER, type CardFilter} from '../card/model'

/** 面板的界面状态：搜索、筛选和「只看今日」。按面板保存在当前设备上，不同步到其他设备。 */
export interface PanelViewState {
  filter: CardFilter
  todayOnly: boolean
}

const DEFAULT_VIEW: PanelViewState = {filter: EMPTY_FILTER, todayOnly: false}

function storageKey(panelId: number): string {
  return `kanban.panel.${panelId}.view`
}

function read(panelId: number): PanelViewState {
  try {
    const value = JSON.parse(localStorage.getItem(storageKey(panelId)) ?? 'null') as Partial<PanelViewState> | null
    if (!value) return DEFAULT_VIEW
    return {filter: {...EMPTY_FILTER, ...value.filter}, todayOnly: value.todayOnly === true}
  } catch {
    return DEFAULT_VIEW
  }
}

/** 读取和修改面板的界面状态，修改后保存到当前设备。 */
export function usePanelView(panelId: number) {
  const [view, setView] = useState(() => read(panelId))

  const update = useCallback(
    (changes: Partial<PanelViewState>) => {
      setView((current) => {
        const next = {...current, ...changes}
        try {
          localStorage.setItem(storageKey(panelId), JSON.stringify(next))
        } catch {
          // 存储不可用时只在本次打开期间生效。
        }
        return next
      })
    },
    [panelId],
  )

  return {view, update}
}
