import {useState, type FormEvent} from 'react'
import {updateListSettings, type ListSettingsInput} from '../api/list'
import {errorMessage} from '../api/client'
import type {List, ListEnd} from '../api/types'
import {ColorPicker} from '../components/ColorPicker'
import {Dialog} from '../components/Dialog'
import {Select} from '../components/Select'
import {useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {sortModes} from './model'
import {t, useT} from '../i18n'

/** 两个创建按钮的插入位置选项。文案随界面语言变化，必须在渲染时构造。 */
function endOptions(): {value: ListEnd; label: string}[] {
  return [
    {value: 'head', label: t('list.createHead')},
    {value: 'tail', label: t('list.createTail')},
  ]
}

/** 列表的展示和通知设置：颜色、卡龄、排序、两个创建按钮的插入位置、通知开关。挂载时即打开。 */
export function ListSettingsDialog({list, onClose}: {list: List; onClose: () => void}) {
  const t = useT()
  const store = useSyncStore()
  const [form, setForm] = useState<ListSettingsInput>({
    color: list.color,
    show_age: list.show_age,
    sort_mode: list.sort_mode,
    sort_dir: list.sort_dir,
    head_add: list.head_add,
    tail_add: list.tail_add,
    remind_off: list.remind_off,
    due_off: list.due_off,
  })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const set = (changes: Partial<ListSettingsInput>) => setForm((current) => ({...current, ...changes}))

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const result = await updateListSettings(list.id, form)
      store.applyLocal(EntityType.List, list.id, result.data, result.revision)
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('list.settingsTitle', {name: list.name})}>
      <form className="form" onSubmit={submit}>
        <div className="field">
          <span>{t('common.color')}</span>
          <div className="color-row">
            <label className="switch">
              <input type="checkbox" checked={form.color !== ''} onChange={(e) => set({color: e.target.checked ? '#3E63DD' : ''})} />
              <span className="track" />
              {t('list.useColor')}
            </label>
            {form.color !== '' && <ColorPicker value={form.color} label={t('list.color')} onChange={(color) => set({color})} />}
          </div>
        </div>
        <label className="switch">
          <input type="checkbox" checked={form.show_age} onChange={(e) => set({show_age: e.target.checked})} />
          <span className="track" />
          {t('list.showAge')}
        </label>
        <div className="field-row">
          <label className="field">
            <span>{t('list.sortMode')}</span>
            <Select
              label={t('list.sortMode')}
              value={form.sort_mode}
              options={sortModes().map((mode) => ({value: mode.value, label: mode.label}))}
              onChange={(value) => set({sort_mode: value as ListSettingsInput['sort_mode']})}
            />
          </label>
          <label className="field">
            <span>{t('list.sortDir')}</span>
            <Select
              label={t('list.sortDir')}
              value={form.sort_dir}
              disabled={form.sort_mode === 'manual'}
              options={[
                {value: 'asc', label: t('list.sortAsc')},
                {value: 'desc', label: t('list.sortDesc')},
              ]}
              onChange={(value) => set({sort_dir: value as ListSettingsInput['sort_dir']})}
            />
          </label>
        </div>
        <p className="hint">{t('list.sortDirHint')}</p>
        <div className="field-row">
          <label className="field">
            <span>{t('list.headCreate')}</span>
            <Select
              label={t('list.headCreate')}
              value={form.head_add}
              options={endOptions().map((end) => ({value: end.value, label: end.label}))}
              onChange={(value) => set({head_add: value as ListEnd})}
            />
          </label>
          <label className="field">
            <span>{t('list.tailCreate')}</span>
            <Select
              label={t('list.tailCreate')}
              value={form.tail_add}
              options={endOptions().map((end) => ({value: end.value, label: end.label}))}
              onChange={(value) => set({tail_add: value as ListEnd})}
            />
          </label>
        </div>
        <label className="switch">
          <input type="checkbox" checked={form.remind_off} onChange={(e) => set({remind_off: e.target.checked})} />
          <span className="track" />
          {t('list.remindOff')}
        </label>
        <label className="switch">
          <input type="checkbox" checked={form.due_off} onChange={(e) => set({due_off: e.target.checked})} />
          <span className="track" />
          {t('list.dueOff')}
        </label>
        <p className="hint">{t('list.notifyHint')}</p>
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
