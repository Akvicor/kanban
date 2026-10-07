import {useState, type FormEvent} from 'react'
import {errorMessage} from '../api/client'
import {Dialog} from '../components/Dialog'
import {Select} from '../components/Select'
import {boardOptions} from '../directory/tree'
import {useDirectoryData} from '../sync/hooks'
import {useT} from '../i18n'

interface BoardSelectDialogProps {
  title: string
  description: string
  confirmText: string
  /** 默认选中的看板；不在目录中时默认选中第一个。 */
  initialBoardId: number | null
  /** 不能选择的看板，例如面板当前所在的看板。 */
  excludeBoardId?: number
  /** 保存选择，失败时抛出错误，错误信息显示在弹窗中。 */
  onSubmit: (boardId: number) => Promise<void>
  onClose: () => void
}

/** 选择一个目录中的看板，用于把面板移到其他看板和恢复面板。挂载时即打开。 */
export function BoardSelectDialog({title, description, confirmText, initialBoardId, excludeBoardId, onSubmit, onClose}: BoardSelectDialogProps) {
  const t = useT()
  const {folders, boards} = useDirectoryData()
  const options = boardOptions(folders, boards).filter((option) => option.board.id !== excludeBoardId)
  const initial = options.find((option) => option.board.id === initialBoardId) ?? options[0]
  const [boardId, setBoardId] = useState<number | null>(initial?.board.id ?? null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (boardId === null) return
    setSaving(true)
    setError('')
    try {
      await onSubmit(boardId)
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={title} description={description}>
      <form className="form" onSubmit={submit}>
        {options.length === 0 ? (
          <p className="hint">{t('board.noneToPick')}</p>
        ) : (
          <label className="field">
            <span>{t('common.board')}</span>
            <Select
              label={t('common.board')}
              value={boardId === null ? '' : String(boardId)}
              options={options.map((option) => ({value: String(option.board.id), label: option.label}))}
              onChange={(value) => setBoardId(Number(value))}
            />
          </label>
        )}
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving || boardId === null}>
            {confirmText}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
