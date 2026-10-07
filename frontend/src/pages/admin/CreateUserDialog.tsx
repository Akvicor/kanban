import {useState, type FormEvent} from 'react'
import * as adminApi from '../../api/admin'
import {errorMessage} from '../../api/client'
import {Dialog} from '../../components/Dialog'
import {useT} from '../../i18n'

interface CreateUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: () => void
}

/** 新建普通用户。只填写用户名和密码，其余设置由用户自己修改。 */
export function CreateUserDialog({open, onOpenChange, onCreated}: CreateUserDialogProps) {
  const t = useT()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  function reset() {
    setUsername('')
    setPassword('')
    setError('')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      await adminApi.createUser(username.trim(), password)
      reset()
      onCreated()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset()
        onOpenChange(next)
      }}
      title={t('admin.createUser')}
      description={t('admin.createUserDesc')}
    >
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.username')}</span>
          <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus />
        </label>
        <label className="field">
          <span>{t('common.password')}</span>
          <input className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </label>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving}>
            {t('common.create')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
