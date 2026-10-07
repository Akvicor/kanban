import {useState} from 'react'
import {errorMessage} from '../../api/client'
import type {Device} from '../../api/types'
import {revokeDevice} from '../../api/user'
import {useMe, useSession} from '../../session/context'
import {useDevices, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import {useT} from '../../i18n'

/** 已登录的设备，随同步实时更新。可以踢掉其他设备；踢掉当前设备等于退出登录。 */
export function DeviceSection() {
  const t = useT()
  const me = useMe()
  const settings = useSettings()
  const devices = useDevices()
  const store = useSyncStore()
  const {logout} = useSession()
  const [error, setError] = useState('')

  async function revoke(device: Device) {
    if (device.id === me.device_id) {
      await logout()
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
              <button type="button" className="btn sm danger" onClick={() => void revoke(device)}>
                {current ? t('common.signOut') : t('settings.kick')}
              </button>
            </li>
          )
        })}
      </ul>
      {error && <p className="form-error">{error}</p>}
    </section>
  )
}
