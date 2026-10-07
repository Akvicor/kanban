import {useState, type FormEvent} from 'react'
import {errorMessage} from '../api/client'
import {Dialog} from '../components/Dialog'
import {useT} from '../i18n'

interface NameDialogProps {
  title: string
  initialName: string
  confirmText: string
  /** 名称的最大长度，缺省 100（文件夹、看板、面板、列表）。 */
  maxLength?: number
  /** 保存名称，失败时抛出错误，错误信息显示在弹窗中。 */
  onSubmit: (name: string) => Promise<void>
  onClose: () => void
}

/** 输入名称的弹窗，用于新建和重命名文件夹、看板。挂载时即打开，关闭时由调用方卸载。 */
export function NameDialog({title, initialName, confirmText, maxLength = 100, onSubmit, onClose}: NameDialogProps) {
  const t = useT()
  const [name, setName] = useState(initialName)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      await onSubmit(name.trim())
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={title}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.name')}</span>
          <input className="input" value={name} maxLength={maxLength} onChange={(e) => setName(e.target.value)} required autoFocus />
        </label>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving || name.trim() === ''}>
            {confirmText}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
