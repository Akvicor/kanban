import {useCallback, useState} from 'react'
import {errorMessage} from '../../api/client'
import type {SettingsChanges} from '../../api/types'
import {updateSettings} from '../../api/user'
import {useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'

/** 保存一部分个人设置，成功后立即用服务端返回的完整设置更新本地同步数据。 */
export function useSaveSettings() {
  const store = useSyncStore()
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const save = useCallback(
    async (changes: SettingsChanges): Promise<boolean> => {
      setSaving(true)
      setError('')
      try {
        const {data, revision} = await updateSettings(changes)
        store.applyLocal(EntityType.Settings, 0, data, revision)
        return true
      } catch (err) {
        setError(errorMessage(err))
        return false
      } finally {
        setSaving(false)
      }
    },
    [store],
  )

  return {save, saving, error}
}
