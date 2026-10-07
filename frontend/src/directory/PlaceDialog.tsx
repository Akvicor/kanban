import {useState, type FormEvent} from 'react'
import {errorMessage} from '../api/client'
import {Dialog} from '../components/Dialog'
import {Select} from '../components/Select'
import type {ParentOption} from './tree'
import {useT} from '../i18n'

interface PlaceDialogProps {
  title: string
  description: string
  confirmText: string
  options: ParentOption[]
  initialParentId: number | null
  /** 保存位置。atStart 为 true 时放在同级最前面，否则放在末尾。失败时抛出错误。 */
  onSubmit: (parentId: number | null, atStart: boolean) => Promise<void>
  onClose: () => void
}

/** 选择目标父级和同级位置的弹窗，用于「移动到…」和从看板归档恢复。挂载时即打开。 */
export function PlaceDialog({title, description, confirmText, options, initialParentId, onSubmit, onClose}: PlaceDialogProps) {
  const t = useT()
  const initial = options.find((option) => option.id === initialParentId && !option.disabledReason) ?? options[0]
  const [parentId, setParentId] = useState<number | null>(initial.id)
  const [atStart, setAtStart] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      await onSubmit(parentId, atStart)
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
        <label className="field">
          <span>{t('common.position')}</span>
          <Select
            label={t('common.position')}
            value={parentId === null ? '' : String(parentId)}
            options={options.map((option) => ({
              value: option.id === null ? '' : String(option.id),
              label: `${'\u3000'.repeat(Math.max(option.depth - 1, 0))}${option.name}${option.disabledReason ? `（${option.disabledReason}）` : ''}`,
              disabled: option.disabledReason !== null,
            }))}
            onChange={(value) => setParentId(value === '' ? null : Number(value))}
          />
        </label>
        <div className="radio-row">
          <label>
            <input type="radio" checked={!atStart} onChange={() => setAtStart(false)} /> {t('directory.placeLast')}
          </label>
          <label>
            <input type="radio" checked={atStart} onChange={() => setAtStart(true)} /> {t('directory.placeFirst')}
          </label>
        </div>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving}>
            {confirmText}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
