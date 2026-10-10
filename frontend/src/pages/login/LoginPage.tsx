import {useState, type FormEvent} from 'react'
import {Navigate, useLocation, useNavigate} from 'react-router-dom'
import {errorMessage} from '../../api/client'
import {displayName} from '../../session/account'
import {canLeavePage} from '../../session/accounts/actions'
import {AccountsDialog} from '../../session/accounts/AccountsDialog'
import {currentAccount} from '../../session/accounts/storage'
import {useAccounts} from '../../session/accounts/useAccounts'
import {useSession} from '../../session/context'
import './LoginPage.css'
import {useT} from '../../i18n'

/** 进入登录页时附带的路由状态：登录后回到的地址；addAccount 表示已登录时添加另一个账号。 */
interface LoginState {
  from?: string
  addAccount?: boolean
}

/**
 * 登录页。没有自行注册，账号由管理员创建。三种情况：
 * - 未登录：普通登录。
 * - 当前账号登录已失效：提示重新登录并预填用户名；本设备还有其他有效账号时可以切换过去。
 * - 已登录时从账号弹窗进入：登录新账号，成功后切换过去并刷新页面，可以取消返回。
 */
export function LoginPage() {
  const t = useT()
  const {status, me, login} = useSession()
  const location = useLocation()
  const navigate = useNavigate()
  const accounts = useAccounts()
  const state = (location.state as LoginState | null) ?? {}
  const adding = status === 'authenticated' && state.addAccount === true
  const current = currentAccount(accounts)
  const expired = status !== 'authenticated' && current?.token === null ? current : null
  const [expiredUserId] = useState(() => expired?.userId ?? null)
  const [username, setUsername] = useState(() => expired?.username ?? '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [switching, setSwitching] = useState(false)
  const canSwitch = expired !== null && accounts.accounts.some((account) => account.userId !== accounts.current && account.token !== null)

  if (status === 'authenticated' && !adding) {
    // 重新登录时换成了另一个账号，原地址属于失效的账号，回到主页。
    const sameAccount = expiredUserId === null || me?.account.id === expiredUserId
    const from = state.from
    return <Navigate to={sameAccount && from && from !== '/login' ? from : '/'} replace />
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    // 添加账号成功后会刷新页面，有上传或请求进行中时不登录。
    if (adding && !canLeavePage()) {
      setError(t('account.busy'))
      return
    }
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
          {adding ? t('login.addTitle') : t('common.board')}
        </div>
        {expired && (
          <p className="login-notice" role="status">
            {t('login.expired', {name: displayName(expired)})}
          </p>
        )}
        <label className="field">
          <span>{t('common.username')}</span>
          <input
            className="input"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoFocus={!expired}
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
            autoFocus={expired !== null}
          />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="btn primary" type="submit" disabled={submitting || status === 'loading'}>
          {submitting ? t('login.submitting') : t('login.submit')}
        </button>
        {adding && (
          <button type="button" className="btn ghost" disabled={submitting} onClick={() => navigate(state.from ?? '/', {replace: true})}>
            {t('common.cancel')}
          </button>
        )}
        {canSwitch && (
          <button type="button" className="btn ghost" onClick={() => setSwitching(true)}>
            {t('login.switchOther')}
          </button>
        )}
      </form>
      {canSwitch && <AccountsDialog open={switching} onOpenChange={setSwitching} switchOnly />}
    </div>
  )
}
