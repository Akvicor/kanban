import {useState, type FormEvent} from 'react'
import {Navigate, useLocation} from 'react-router-dom'
import {errorMessage} from '../../api/client'
import {useSession} from '../../session/context'
import './LoginPage.css'
import {useT} from '../../i18n'

/** 登录页。没有自行注册，账号由管理员创建。 */
export function LoginPage() {
  const t = useT()
  const {status, login} = useSession()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  if (status === 'authenticated') {
    const from = (location.state as {from?: string} | null)?.from
    return <Navigate to={from && from !== '/login' ? from : '/'} replace />
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await login(username.trim(), password)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="login-page">
      <form className="login-card form" onSubmit={submit}>
        <div className="brand">
          <img className="logo" src="/icon-192.png" alt="" />
          {t('common.board')}
        </div>
        <label className="field">
          <span>{t('common.username')}</span>
          <input
            className="input"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoFocus
          />
        </label>
        <label className="field">
          <span>{t('common.password')}</span>
          <input
            className="input"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="btn primary" type="submit" disabled={submitting || status === 'loading'}>
          {submitting ? t('login.submitting') : t('login.submit')}
        </button>
      </form>
    </div>
  )
}
