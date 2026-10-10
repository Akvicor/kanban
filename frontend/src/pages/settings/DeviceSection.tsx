import {useState} from 'react'
import {errorMessage} from '../../api/client'
import type {Device} from '../../api/types'
import {revokeDevice} from '../../api/user'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {canLeavePage, logoutAccount} from '../../session/accounts/actions'
import {useMe} from '../../session/context'
import {useDevices, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import {useT} from '../../i18n'

/**
 * 已登录的设备，随同步实时更新。可以踢掉其他设备；踢掉当前设备等于退出当前账号（切到账号列表中的下一个账号）。
 * 两种操作都需要二次确认。
 */
export function DeviceSection() {
  const t = useT()
  const me = useMe()
  const settings = useSettings()
  const devices = useDevices()
  const store = useSyncStore()
  const [error, setError] = useState('')
  const [confirming, setConfirming] = useState<Device | null>(null)
  const confirmingCurrent = confirming?.id === me.device_id

  async function revoke(device: Device) {
    if (device.id === me.device_id) {
      if (!canLeavePage()) {
        setError(t('account.busy'))
        return
      }
      await logoutAccount(me.account.id)
      return
    }
    try {
      const {revision} = await revokeDevice(device.id)
      store.applyLocal(EntityType.Device, device.id, null, revision)
      setError('')
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  function confirm() {
    const device = confirming
    setConfirming(null)
    if (device) void revoke(device)
  }

  return (
    <section className="card-section">
      <h2>{t('settings.devices')}</h2>
      <p className="hint">{t('settings.devicesHint')}</p>
      <ul className="device-list">
        {devices.map((device) => {
          const current = device.id === me.device_id
          return (
            <li key={device.id}>
              <div className="device-info">
                <strong>
                  {device.name}
                  {current && <span className="tag">{t('settings.currentDevice')}</span>}
                </strong>
                <span className="hint">
                  {t('settings.lastActive', {active: formatDateTime(device.last_active_at, settings.timezone)})} · {t('settings.signedInAt', {created: formatDateTime(device.created_at, settings.timezone)})}
                </span>
              </div>
              <button type="button" className="btn sm danger" onClick={() => setConfirming(device)}>
                {current ? t('common.signOut') : t('settings.kick')}
              </button>
            </li>
          )
        })}
      </ul>
      {error && <p className="form-error">{error}</p>}
      <ConfirmDialog
        open={confirming !== null}
        title={confirmingCurrent ? t('account.signOutCurrent') : t('settings.kickTitle', {name: confirming?.name ?? ''})}
        description={confirmingCurrent ? t('account.signOutCurrentDesc') : t('settings.kickDesc')}
        confirmText={confirmingCurrent ? t('account.signOut') : t('settings.kick')}
        danger
        onConfirm={confirm}
        onCancel={() => setConfirming(null)}
      />
    </section>
  )
}
