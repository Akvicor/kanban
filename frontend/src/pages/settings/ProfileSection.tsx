import {useState, type FormEvent} from 'react'
import {errorMessage} from '../../api/client'
import {updateProfile} from '../../api/user'
import {displayName} from '../../session/account'
import {useMe} from '../../session/context'
import {useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {useT} from '../../i18n'

/** 修改用户名和昵称。用户名用于登录，昵称用于界面显示。 */
export function ProfileSection() {
  const t = useT()
  const me = useMe()
  const store = useSyncStore()
  const [username, setUsername] = useState(me.account.username)
  const [nickname, setNickname] = useState(displayName(me.account))
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setDone(false)
    try {
      const result = await updateProfile(username.trim(), nickname.trim())
      store.applyLocal(EntityType.Account, me.account.id, result.data, result.revision)
      setDone(true)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="card-section">
      <h2>{t('settings.account')}</h2>
      <p className="hint">{t('settings.profileHint')}</p>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.username')}</span>
          <input className="input" value={username} maxLength={32} autoComplete="username" required onChange={(e) => setUsername(e.target.value)} />
        </label>
        <label className="field">
          <span>{t('common.nickname')}</span>
          <input className="input" value={nickname} maxLength={32} required onChange={(e) => setNickname(e.target.value)} />
        </label>
        {error && <p className="form-error">{error}</p>}
        {done && <p className="form-ok">{t('common.saved')}</p>}
        <div className="form-actions">
          <button type="submit" className="btn primary" disabled={saving}>
            {t('common.save')}
          </button>
        </div>
      </form>
    </section>
  )
}
