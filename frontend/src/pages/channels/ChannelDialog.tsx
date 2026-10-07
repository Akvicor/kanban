import {useState, type FormEvent} from 'react'
import {errorMessage} from '../../api/client'
import {createChannel, updateChannel, type ChannelInput} from '../../api/notify'
import type {NotifyChannel, NotifyFormat} from '../../api/types'
import {Dialog} from '../../components/Dialog'
import {useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {useT} from '../../i18n'

/**
 * 新建或编辑通知渠道。channel 为 null 时新建，五项都必填；编辑时 Token、Sign 留空表示保留原值。
 * 挂载时即打开。
 */
export function ChannelDialog({channel, onClose}: {channel: NotifyChannel | null; onClose: () => void}) {
  const t = useT()
  const store = useSyncStore()
  const [input, setInput] = useState<ChannelInput>({
    name: channel?.name ?? '',
    api: channel?.api ?? '',
    token: '',
    sign: '',
    format: channel?.format ?? 'markdown',
  })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const set = (changes: Partial<ChannelInput>) => setInput((current) => ({...current, ...changes}))

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const result = channel ? await updateChannel(channel.id, input) : await createChannel(input)
      store.applyLocal(EntityType.NotifyChannel, result.data.id, result.data, result.revision)
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  const keep = channel ? t('channels.leaveBlankKeep') : ''
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={channel ? t('channels.edit') : t('channels.create')} description={t('channels.gmsgHint')}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('channels.name')}</span>
          <input className="input" value={input.name} maxLength={50} required autoFocus onChange={(e) => set({name: e.target.value})} />
        </label>
        <label className="field">
          <span>API</span>
          <input className="input" value={input.api} maxLength={2048} placeholder="https://msg.example.com" required onChange={(e) => set({api: e.target.value})} />
        </label>
        <label className="field">
          <span>Token</span>
          <input
            className="input"
            type="password"
            autoComplete="off"
            value={input.token}
            maxLength={500}
            required={!channel}
            placeholder={channel?.token_set ? t('channels.tokenSet', {hint: keep}) : keep}
            onChange={(e) => set({token: e.target.value})}
          />
        </label>
        <label className="field">
          <span>Sign</span>
          <input
            className="input"
            type="password"
            autoComplete="off"
            value={input.sign}
            maxLength={500}
            required={!channel}
            placeholder={channel?.sign_set ? t('channels.tokenSet', {hint: keep}) : keep}
            onChange={(e) => set({sign: e.target.value})}
          />
        </label>
        <div className="field">
          <span>{t('channels.format')}</span>
          <div className="radio-row">
            {(['markdown', 'text'] as NotifyFormat[]).map((format) => (
              <label key={format}>
                <input type="radio" name="format" checked={input.format === format} onChange={() => set({format})} />
                {format === 'markdown' ? 'Markdown' : t('common.text')}
              </label>
            ))}
          </div>
        </div>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving}>
            {t('common.save')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
