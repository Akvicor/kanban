import {useMemo, useState, type ReactNode} from 'react'
import {useNavigate} from 'react-router-dom'
import * as directoryApi from '../api/directory'
import {errorMessage, type WriteResult} from '../api/client'
import type {Board} from '../api/types'
import {ConfirmDialog} from '../components/ConfirmDialog'
import {useToast} from '../components/Toast'
import {useDirectoryData, useSettings, useSyncStore} from '../sync/hooks'
import {EntityType, type SyncStore} from '../sync/store'
import {DirectoryActionsContext, type DirTarget, type DirectoryActions} from './actions'
import {NameDialog} from './NameDialog'
import {PlaceDialog} from './PlaceDialog'
import {parentOptions} from './tree'
import {useT} from '../i18n'

/** 当前打开的弹窗。 */
type Pending =
  | {type: 'create'; kind: 'folder' | 'board'; parentId: number | null}
  | {type: 'rename'; target: DirTarget}
  | {type: 'move'; target: DirTarget}
  | {type: 'archive'; target: DirTarget}
  | {type: 'restore'; board: Board}

function targetName(target: DirTarget): string {
  return target.kind === 'folder' ? target.folder.name : target.board.name
}

function targetParent(target: DirTarget): number | null {
  return target.kind === 'folder' ? target.folder.parent_id : target.board.folder_id
}

/** 用写入响应立即更新本地的文件夹或看板。 */
function applyResult(store: SyncStore, kind: DirTarget['kind'], result: WriteResult<{id: number}>) {
  store.applyLocal(kind === 'folder' ? EntityType.Folder : EntityType.Board, result.data.id, result.data, result.revision)
}

/** 把文件夹或看板放到目标父级的第 index 位，index 为 null 时排在末尾。 */
async function placeTarget(store: SyncStore, target: DirTarget, parentId: number | null, index: number | null) {
  const result =
    target.kind === 'folder'
      ? await directoryApi.moveFolder(target.folder.id, parentId, index)
      : await directoryApi.moveBoard(target.board.id, parentId, index)
  applyResult(store, target.kind, result)
}

/**
 * 提供目录操作，并渲染操作需要的弹窗和失败提示。写入成功后立即用响应更新本地同步数据，
 * 受影响的其他文件夹和看板（例如重新排列的同级、归档时一并归档的下级）通过同步推送到达。
 */
export function DirectoryActionsProvider({children}: {children: ReactNode}) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const {folders, boards} = useDirectoryData()
  const navigate = useNavigate()
  const {show: showToast, element: toastElement} = useToast()
  const [pending, setPending] = useState<Pending | null>(null)
  const close = () => setPending(null)
  const mainBoardId = settings.main_board_id

  const actions = useMemo<DirectoryActions>(() => {
    const run = (operation: () => Promise<void>) => {
      operation().catch((err: unknown) => showToast(errorMessage(err)))
    }
    return {
      createBoard: (folderId) => setPending({type: 'create', kind: 'board', parentId: folderId}),
      createFolder: (parentId) => setPending({type: 'create', kind: 'folder', parentId}),
      rename: (target) => setPending({type: 'rename', target}),
      move: (target) => setPending({type: 'move', target}),
      archive: (target) => setPending({type: 'archive', target}),
      restore: (board) => setPending({type: 'restore', board}),
      toggleMain: (board) =>
        run(async () => {
          const result = await directoryApi.setMainBoard(mainBoardId === board.id ? null : board.id)
          store.applyLocal(EntityType.Settings, 0, result.data, result.revision)
        }),
      place: (target, parentId, index) => run(() => placeTarget(store, target, parentId, index)),
    }
  }, [store, mainBoardId, showToast])

  let dialog: ReactNode = null
  if (pending?.type === 'create') {
    const {kind, parentId} = pending
    dialog = (
      <NameDialog
        title={kind === 'board' ? t('directory.createBoard') : t('directory.createFolder')}
        initialName=""
        confirmText={t('common.create')}
        onClose={close}
        onSubmit={async (name) => {
          if (kind === 'board') {
            const result = await directoryApi.createBoard(parentId, name)
            applyResult(store, 'board', result)
            navigate(`/board/${result.data.id}`)
          } else {
            applyResult(store, 'folder', await directoryApi.createFolder(parentId, name))
          }
        }}
      />
    )
  } else if (pending?.type === 'rename') {
    const {target} = pending
    dialog = (
      <NameDialog
        title={target.kind === 'folder' ? t('directory.renameFolder') : t('directory.renameBoard')}
        initialName={targetName(target)}
        confirmText={t('common.save')}
        onClose={close}
        onSubmit={async (name) => {
          const result =
            target.kind === 'folder'
              ? await directoryApi.renameFolder(target.folder.id, name)
              : await directoryApi.renameBoard(target.board.id, name)
          applyResult(store, target.kind, result)
        }}
      />
    )
  } else if (pending?.type === 'move') {
    const {target} = pending
    const moving = target.kind === 'folder' ? {kind: target.kind, id: target.folder.id} : {kind: target.kind, id: target.board.id}
    dialog = (
      <PlaceDialog
        title={t('directory.moveTitle', {name: targetName(target)})}
        description={t('directory.choosePlace')}
        confirmText={t('common.move')}
        options={parentOptions(folders, boards, moving)}
        initialParentId={targetParent(target)}
        onClose={close}
        onSubmit={(parentId, atStart) => placeTarget(store, target, parentId, atStart ? 0 : null)}
      />
    )
  } else if (pending?.type === 'restore') {
    const {board} = pending
    dialog = (
      <PlaceDialog
        title={t('directory.restoreTitle', {name: board.name})}
        description={t('directory.restoreBoardDesc')}
        confirmText={t('common.restore')}
        options={parentOptions(folders, boards)}
        initialParentId={board.folder_id}
        onClose={close}
        onSubmit={async (parentId, atStart) => {
          applyResult(store, 'board', await directoryApi.restoreBoard(board.id, parentId, atStart ? 0 : null))
        }}
      />
    )
  } else if (pending?.type === 'archive') {
    const {target} = pending
    dialog = (
      <ConfirmDialog
        open
        title={t('directory.archiveTitle', {name: targetName(target)})}
        description={
          target.kind === 'folder'
            ? t('directory.archiveFolderDesc')
            : t('directory.archiveBoardDesc')
        }
        confirmText={t('common.archive')}
        danger
        onCancel={close}
        onConfirm={() => {
          close()
          const request = target.kind === 'folder' ? directoryApi.archiveFolder(target.folder.id) : directoryApi.archiveBoard(target.board.id)
          request.catch((err: unknown) => showToast(errorMessage(err)))
        }}
      />
    )
  }

  return (
    <DirectoryActionsContext.Provider value={actions}>
      {children}
      {dialog}
      {toastElement}
    </DirectoryActionsContext.Provider>
  )
}
