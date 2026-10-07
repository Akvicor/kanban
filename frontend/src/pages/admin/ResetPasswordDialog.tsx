import {useState, type FormEvent} from 'react'
import * as adminApi from '../../api/admin'
import {errorMessage} from '../../api/client'
import type {AdminUser} from '../../api/types'
import {Dialog} from '../../components/Dialog'
import {useT} from '../../i18n'

interface ResetPasswordDialogProps {
  /** 要重置密码的账号，为 null 时关闭弹窗。 */
  user: AdminUser | null
  onClose: () => void
}

/** 重置账号密码。重置后该账号的全部设备需要重新登录。 */
export function ResetPasswordDialog({user, onClose}: ResetPasswordDialogProps) {
  const t = useT()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  function close() {
    setPassword('')
    setError('')
    onClose()
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!user) return
    setSaving(true)
    setError('')
    try {
      await adminApi.resetPassword(user.id, password)
      close()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open={user !== null}
      onOpenChange={(next) => !next && close()}
      title={t('admin.resetPasswordTitle', {name: user?.username ?? ''})}
      description={t('admin.resetPasswordDesc')}
    >
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.newPassword')}</span>
          <input className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} required autoFocus />
        </label>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={close}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving}>
            {t('admin.resetPassword')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
