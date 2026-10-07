import {MenuButton} from '../../layout/MenuButton'
import {useState} from 'react'
import * as adminApi from '../../api/admin'
import {errorMessage} from '../../api/client'
import type {AdminUser} from '../../api/types'
import {useRemoteData} from '../../hooks/useRemoteData'
import {useMe} from '../../session/context'
import {useSettings} from '../../sync/hooks'
import {formatDateTime} from '../../utils/datetime'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {CreateUserDialog} from './CreateUserDialog'
import {ResetPasswordDialog} from './ResetPasswordDialog'
import './UsersPage.css'
import {useT} from '../../i18n'

/** 正在确认的操作：停用或删除某个账号。 */
type PendingAction = {kind: 'disable' | 'delete'; user: AdminUser}

/**
 * 用户管理，只对管理员开放。管理员创建账号时只填写用户名和密码；
 * 停用、重置密码会让该账号的全部设备重新登录，删除会同时删除该账号的全部数据。
 */
export function UsersPage() {
  const t = useT()
  const me = useMe()
  const settings = useSettings()
  const {data: users, error: loadError, reload} = useRemoteData(adminApi.listUsers, [] as AdminUser[])
  const [actionError, setActionError] = useState('')
  const [creating, setCreating] = useState(false)
  const [resetting, setResetting] = useState<AdminUser | null>(null)
  const [pending, setPending] = useState<PendingAction | null>(null)
  const error = actionError || loadError

  async function run(operation: () => Promise<unknown>) {
    try {
      await operation()
      setActionError('')
      reload()
    } catch (err) {
      setActionError(errorMessage(err))
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('nav.users')}</h1>
        <span className="grow" />
        <button type="button" className="btn primary" onClick={() => setCreating(true)}>
          {t('admin.createUser')}
        </button>
      </div>
      {error && <p className="form-error">{error}</p>}
      <div className="card-section user-table-wrap">
        <table className="user-table">
          <thead>
            <tr>
              <th>{t('common.username')}</th>
              <th>{t('common.nickname')}</th>
              <th>{t('admin.role')}</th>
              <th>{t('admin.status')}</th>
              <th>{t('common.createdAt')}</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {users.map((user) => {
              const self = user.id === me.account.id
              return (
                <tr key={user.id}>
                  <td>{user.username}</td>
                  <td>{user.nickname || user.username}</td>
                  <td>{user.role === 'admin' ? t('admin.roleAdmin') : t('admin.roleUser')}</td>
                  <td>{user.disabled_at ? <span className="tag danger">{t('admin.disabled')}</span> : <span className="tag">{t('common.ok')}</span>}</td>
                  <td>{formatDateTime(user.created_at, settings.timezone)}</td>
                  <td>
                    <div className="user-actions">
                      {self ? (
                        <span className="hint">{t('admin.currentAccount')}</span>
                      ) : (
                        <>
                          <button type="button" className="btn sm" onClick={() => setResetting(user)}>
                            {t('admin.resetPassword')}
                          </button>
                          {user.disabled_at ? (
                            <button type="button" className="btn sm" onClick={() => void run(() => adminApi.enableUser(user.id))}>
                              {t('admin.enable')}
                            </button>
                          ) : (
                            <button type="button" className="btn sm" onClick={() => setPending({kind: 'disable', user})}>
                              {t('admin.disable')}
                            </button>
                          )}
                          <button type="button" className="btn sm danger" onClick={() => setPending({kind: 'delete', user})}>
                            {t('common.delete')}
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <CreateUserDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => {
          setCreating(false)
          reload()
        }}
      />
      <ResetPasswordDialog user={resetting} onClose={() => setResetting(null)} />
      <ConfirmDialog
        open={pending !== null}
        title={pending?.kind === 'delete' ? t('admin.deleteUserTitle', {name: pending.user.username}) : t('admin.disableUserTitle', {name: pending?.user.username ?? ''})}
        description={
          pending?.kind === 'delete'
            ? t('admin.deleteDesc')
            : t('admin.disableDesc')
        }
        confirmText={pending?.kind === 'delete' ? t('common.delete') : t('admin.disable')}
        danger
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          if (!pending) return
          const {kind, user} = pending
          setPending(null)
          await run(() => (kind === 'delete' ? adminApi.deleteUser(user.id) : adminApi.disableUser(user.id)))
        }}
      />
    </div>
  )
}
