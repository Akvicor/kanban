import {useState, type CSSProperties, type FormEvent} from 'react'
import {updateListRules} from '../api/list'
import {errorMessage} from '../api/client'
import type {Label, List, ListOp, ListRules, OpRules, TimeAction} from '../api/types'
import {Dialog} from '../components/Dialog'
import {Select} from '../components/Select'
import {panelLabels} from '../panel/model'
import {usePanelData, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {opLabels, timeActions} from './model'
import {t, useT} from '../i18n'

const OPS: ListOp[] = ['create', 'enter', 'exit']

const OP_HINTS: Record<ListOp, string> = {
  create: t('list.ruleCreate'),
  enter: t('list.ruleEnter'),
  exit: t('list.ruleExit'),
}

/**
 * 列表的操作配置：创建、移入、移出三种操作分别可以增加或删除标签，并决定开始时间和完成时间的动作。
 * 一次操作里先删除标签，再增加标签。挂载时即打开。
 */
export function ListRulesDialog({list, onClose}: {list: List; onClose: () => void}) {
  const t = useT()
  const store = useSyncStore()
  const {labels} = usePanelData()
  const options = panelLabels(labels, list.panel_id)
  const [rules, setRules] = useState<ListRules>(list.rules)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const update = (op: ListOp, changes: Partial<OpRules>) => setRules((current) => ({...current, [op]: {...current[op], ...changes}}))

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const result = await updateListRules(list.id, rules)
      store.applyLocal(EntityType.List, list.id, result.data, result.revision)
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={t('list.rulesTitle', {name: list.name})}
      description={t('list.rulesDesc')}
    >
      <form className="form list-rules" onSubmit={submit}>
        {OPS.map((op) => (
          <section key={op} className="rule-op">
            <h3>{opLabels()[op]}</h3>
            <p className="hint">{OP_HINTS[op]}</p>
            <LabelPicker
              title={t('list.addLabels')}
              labels={options}
              selected={rules[op].add_labels}
              onChange={(ids) => update(op, {add_labels: ids})}
            />
            <LabelPicker
              title={t('list.removeLabels')}
              labels={options}
              selected={rules[op].remove_labels}
              onChange={(ids) => update(op, {remove_labels: ids})}
            />
            <div className="field-row">
              <TimeSelect title={t('common.startedAt')} value={rules[op].start} onChange={(value) => update(op, {start: value})} />
              <TimeSelect title={t('common.completedAt')} value={rules[op].complete} onChange={(value) => update(op, {complete: value})} />
            </div>
          </section>
        ))}
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

function LabelPicker({title, labels, selected, onChange}: {
  title: string
  labels: Label[]
  selected: number[]
  onChange: (ids: number[]) => void
}) {
  return (
    <div className="field">
      <span>{title}</span>
      {labels.length === 0 ? (
        <span className="hint">{t('panel.noLabelsHint')}</span>
      ) : (
        <div className="label-picker">
          {labels.map((label) => {
            const on = selected.includes(label.id)
            return (
              <button
                key={label.id}
                type="button"
                className={on ? 'label-chip on' : 'label-chip'}
                style={{'--c': label.color} as CSSProperties}
                aria-pressed={on}
                onClick={() => onChange(on ? selected.filter((id) => id !== label.id) : [...selected, label.id])}
              >
                {label.name || t('list.unnamed')}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

function TimeSelect({title, value, onChange}: {title: string; value: TimeAction; onChange: (value: TimeAction) => void}) {
  return (
    <label className="field">
      <span>{title}</span>
      <Select
        label={title}
        value={value}
        options={timeActions().map((action) => ({value: action.value, label: action.label}))}
        onChange={(next) => onChange(next as TimeAction)}
      />
    </label>
  )
}
