import {createContext, useContext} from 'react'
import type {Board, Folder} from '../api/types'
import {t} from '../i18n'

/** 目录中可以操作的一项。 */
export type DirTarget = {kind: 'folder'; folder: Folder} | {kind: 'board'; board: Board}

/** 目录操作。需要输入或确认的操作会打开弹窗；失败时显示提示。 */
export interface DirectoryActions {
  createBoard: (folderId: number | null) => void
  createFolder: (parentId: number | null) => void
  rename: (target: DirTarget) => void
  move: (target: DirTarget) => void
  archive: (target: DirTarget) => void
  restore: (board: Board) => void
  toggleMain: (board: Board) => void
  /** 拖动后把一项放到目标位置，index 为 null 时排在末尾。 */
  place: (target: DirTarget, parentId: number | null, index: number | null) => void
}

export const DirectoryActionsContext = createContext<DirectoryActions | null>(null)

export function useDirectoryActions(): DirectoryActions {
  const actions = useContext(DirectoryActionsContext)
  if (!actions) {
    throw new Error(t('error.useDirectoryActions'))
  }
  return actions
}
