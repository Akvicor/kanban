import {useState, type FormEvent} from 'react'
import {errorMessage} from '../../api/client'
import {changePassword} from '../../api/user'
import {useT} from '../../i18n'

/** 密码长度要求，与后端 common/passwd 一致。 */
const MIN_LENGTH = 8

/** 修改密码。成功后这台设备保持登录，其他设备需要重新登录。 */
export function PasswordSection() {
  const t = useT()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setDone(false)
    if (next.length < MIN_LENGTH) {
      setError(t('settings.newPasswordMin', {count: MIN_LENGTH}))
      return
    }
    if (next !== confirm) {
      setError(t('settings.passwordMismatch'))
      return
    }
    setSaving(true)
    setError('')
    try {
      await changePassword(current, next)
      setCurrent('')
      setNext('')
      setConfirm('')
      setDone(true)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="card-section">
      <h2>{t('settings.changePassword')}</h2>
      <p className="hint">{t('settings.changePasswordHint')}</p>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('settings.currentPassword')}</span>
          <input className="input" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} required />
        </label>
        <label className="field">
          <span>{t('common.newPassword')}</span>
          <input className="input" type="password" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)} required />
        </label>
        <label className="field">
          <span>{t('settings.confirmPassword')}</span>
          <input className="input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
        </label>
        {error && <p className="form-error">{error}</p>}
        {done && <p className="form-ok">{t('settings.passwordChanged')}</p>}
        <div className="form-actions">
          <button type="submit" className="btn primary" disabled={saving}>
            {t('settings.changePassword')}
          </button>
        </div>
      </form>
    </section>
  )
}
