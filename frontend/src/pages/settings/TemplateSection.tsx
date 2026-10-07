import {useState, type FormEvent} from 'react'
import {useT} from '../../i18n'
import {useSettings} from '../../sync/hooks'
import {useSaveSettings} from './useSaveSettings'

/** 正文模板中可用的占位符，与后端通知正文的替换规则一致。 */
const PLACEHOLDERS = [
  ['title', 'common.title'],
  ['description', 'common.description'],
  ['board', 'notify.fieldBoard'],
  ['panel', 'notify.fieldPanel'],
  ['list', 'notify.fieldList'],
  ['priority', 'common.priority'],
  ['labels', 'common.label'],
  ['remind_at', 'common.remindAt'],
  ['due_at', 'common.dueAt'],
  ['created_at', 'common.createdAt'],
  ['started_at', 'common.startedAt'],
  ['completed_at', 'common.completedAt'],
  ['tasks', 'notify.fieldTasks'],
] as const

interface Draft {
  remind: string
  due: string
}

/**
 * 提醒和截止通知的正文模板。
 *
 * 模板为空表示使用系统默认模板（随界面语言变化），此时编辑框显示当前语言的默认文本作预览；
 * 保存任何修改即成为自定义模板，不再随语言变化。「恢复默认」清空模板，重新跟随语言。
 * 两份模板一起编辑，点击保存后生效；开始编辑后保留草稿，直到保存或撤销。
 */
export function TemplateSection() {
  const t = useT()
  const settings = useSettings()
  const {save, saving, error} = useSaveSettings()
  const [draft, setDraft] = useState<Draft | null>(null)
  const [saved, setSaved] = useState(false)
  const defaults: Draft = {remind: t('template.defaultRemind'), due: t('template.defaultDue')}
  const current = draft ?? {remind: settings.remind_template, due: settings.due_template}
  const changed = current.remind !== settings.remind_template || current.due !== settings.due_template
  const customized = settings.remind_template !== '' || settings.due_template !== '' || draft !== null

  function edit(next: Partial<Draft>) {
    setDraft({...current, ...next})
    setSaved(false)
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (await save({remind_template: current.remind, due_template: current.due})) {
      setDraft(null)
      setSaved(true)
    }
  }

  async function restore() {
    if (await save({remind_template: '', due_template: ''})) {
      setDraft(null)
      setSaved(true)
    }
  }

  return (
    <section className="card-section">
      <h2>{t('settings.templates')}</h2>
      <p className="hint">{t('settings.templatesHint')}</p>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.remind')}</span>
          <textarea
            className="textarea"
            value={current.remind || defaults.remind}
            onChange={(e) => edit({remind: e.target.value})}
          />
        </label>
        <label className="field">
          <span>{t('common.due')}</span>
          <textarea className="textarea" value={current.due || defaults.due} onChange={(e) => edit({due: e.target.value})} />
        </label>
        <div className="placeholder-list">
          {PLACEHOLDERS.map(([field, labelKey]) => (
            <span key={field} className="tag muted" title={t(labelKey)}>
              {`{{${field}}}`} {t(labelKey)}
            </span>
          ))}
        </div>
        {error && <p className="form-error">{error}</p>}
        {saved && !changed && <p className="form-ok">{t('common.saved')}</p>}
        <div className="form-actions">
          <button type="button" className="btn" disabled={!changed || saving} onClick={() => setDraft(null)}>
            {t('settings.undoChanges')}
          </button>
          {customized && (
            <button type="button" className="btn" disabled={saving} onClick={() => void restore()}>
              {t('settings.restoreDefault')}
            </button>
          )}
          <button type="submit" className="btn primary" disabled={!changed || saving}>
            {t('common.save')}
          </button>
        </div>
      </form>
    </section>
  )
}
