import {useEffect, useState} from 'react'
import {actionLabel, MAX_KEYS_PER_ACTION} from '../../shortcuts/actions'
import {findOwner, formatKey, isApplePlatform, keyFromEvent} from '../../shortcuts/keys'
import {Select} from '../../components/Select'
import {useMe} from '../../session/context'
import {useSettings} from '../../sync/hooks'
import {useSaveSettings} from './useSaveSettings'
import {useT} from '../../i18n'

/**
 * 快捷键设置。点击「添加」后按下要绑定的按键即保存；同一按键不能绑定两个操作，冲突时不保存。
 * 快捷键作用于鼠标悬停的卡片或列。
 */
export function ShortcutSection() {
  const t = useT()
  const me = useMe()
  const {save, saving, error} = useSaveSettings()
  const [capturing, setCapturing] = useState<string | null>(null)
  const [conflict, setConflict] = useState('')
  const apple = isApplePlatform()
  const settings = useSettings()
  const bindings = settings.shortcuts

  useEffect(() => {
    if (!capturing) {
      return
    }
    function onKeyDown(event: KeyboardEvent) {
      event.preventDefault()
      event.stopPropagation()
      if (event.code === 'Escape') {
        setCapturing(null)
        return
      }
      const key = keyFromEvent(event)
      if (!key || !capturing) {
        return
      }
      const keys = bindings[capturing] ?? []
      const owner = findOwner(bindings, key)
      if (owner === capturing) {
        setCapturing(null)
        return
      }
      if (owner) {
        setConflict(t('settings.shortcutConflict', {key: formatKey(key, apple), action: actionLabel(owner)}))
        return
      }
      setCapturing(null)
      void save({shortcuts: {[capturing]: [...keys, key]}})
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [capturing, bindings, apple, save, t])

  function startCapture(action: string) {
    setConflict('')
    setCapturing(action)
  }

  function removeKey(action: string, key: string) {
    setConflict('')
    void save({shortcuts: {[action]: (bindings[action] ?? []).filter((k) => k !== key)}})
  }

  return (
    <section className="card-section">
      <div className="section-title-row">
        <h2>{t('shortcut.title')}</h2>
        <button
          type="button"
          className="btn sm"
          disabled={saving}
          onClick={() => {
            setConflict('')
            void save({shortcuts: me.shortcut_defaults})
          }}
        >
          {t('shortcut.resetAll')}
        </button>
      </div>
      <p className="hint">{t('settings.shortcutHint')}</p>
      <div className="form">
        <label className="field">
          <span>{t('settings.panModifier')}</span>
          <Select
            label={t('settings.panModifier')}
            value={settings.pan_modifier}
            disabled={saving}
            options={[
              {value: '', label: t('settings.panModifierOff')},
              {value: 'ctrl', label: 'Ctrl'},
              {value: 'alt', label: 'Alt'},
              {value: 'shift', label: 'Shift'},
            ]}
            onChange={(value) => void save({pan_modifier: value})}
          />
        </label>
        <p className="hint">{t('settings.panModifierHint')}</p>
      </div>
      <table className="shortcut-table">
        <tbody>
          {me.shortcut_actions.map((action) => {
            const keys = bindings[action] ?? []
            return (
              <tr key={action}>
                <th scope="row">{actionLabel(action)}</th>
                <td>
                  <div className="shortcut-keys">
                    {keys.map((key) => (
                      <span key={key} className="kbd">
                        {formatKey(key, apple)}
                        <button
                          type="button"
                          className="kbd-remove"
                          aria-label={t('settings.shortcutRemove', {key: formatKey(key, apple)})}
                          disabled={saving}
                          onClick={() => removeKey(action, key)}
                        >
                          ×
                        </button>
                      </span>
                    ))}
                    {keys.length === 0 && <span className="hint">{t('shortcut.unbound')}</span>}
                    {capturing === action ? (
                      <span className="capture">{t('shortcut.recording')}</span>
                    ) : (
                      keys.length < MAX_KEYS_PER_ACTION && (
                        <button type="button" className="btn sm ghost" disabled={saving} onClick={() => startCapture(action)}>
                          {t('common.add')}
                        </button>
                      )
                    )}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {(conflict || error) && <p className="form-error">{conflict || error}</p>}
    </section>
  )
}
