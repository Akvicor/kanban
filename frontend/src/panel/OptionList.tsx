import {useState, type FormEvent} from 'react'
import {ColorPicker} from '../components/ColorPicker'
import {t, useT} from '../i18n'

/** 标签或优先级挡位：名称、颜色和顺序。 */
export interface OptionItem {
  id: number
  name: string
  color: string
}

interface OptionListProps {
  items: OptionItem[]
  /** 名称是否必填。标签可以只有颜色，挡位必须有名称。 */
  nameRequired: boolean
  nameMaxLength: number
  /** 新建项的默认颜色。 */
  defaultColor: string
  placeholder: string
  onCreate: (name: string, color: string) => Promise<void>
  onUpdate: (id: number, name: string, color: string) => Promise<void>
  onMove: (id: number, index: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
}

/**
 * 可编辑的标签或挡位列表。颜色用取色器任意选择；名称在失去焦点或按回车时保存；
 * 上下箭头调整顺序；删除需要再点一次确认，删除后不能恢复。
 */
export function OptionList({items, nameRequired, nameMaxLength, defaultColor, placeholder, onCreate, onUpdate, onMove, onDelete}: OptionListProps) {
  const t = useT()
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(defaultColor)
  const [confirming, setConfirming] = useState<number | null>(null)

  async function create(event: FormEvent) {
    event.preventDefault()
    if (nameRequired && newName.trim() === '') return
    await onCreate(newName.trim(), newColor)
    setNewName('')
  }

  return (
    <div className="option-list">
      {items.map((item, index) => (
        <div key={item.id} className="option-row">
          <ColorPicker value={item.color} label={t('common.color')} onChange={(color) => void onUpdate(item.id, item.name, color)} />
          <NameInput
            key={`${item.id}:${item.name}`}
            value={item.name}
            required={nameRequired}
            maxLength={nameMaxLength}
            placeholder={placeholder}
            onSave={(name) => onUpdate(item.id, name, item.color)}
          />
          <button type="button" className="icon-btn sm" aria-label={t('common.moveUp')} disabled={index === 0} onClick={() => void onMove(item.id, index - 1)}>
            ↑
          </button>
          <button
            type="button"
            className="icon-btn sm"
            aria-label={t('common.moveDown')}
            disabled={index === items.length - 1}
            onClick={() => void onMove(item.id, index + 1)}
          >
            ↓
          </button>
          {confirming === item.id ? (
            <button
              type="button"
              className="btn sm danger"
              onClick={() => {
                setConfirming(null)
                void onDelete(item.id)
              }}
              onBlur={() => setConfirming(null)}
              autoFocus
            >
              {t('common.confirmDelete')}
            </button>
          ) : (
            <button type="button" className="btn sm ghost" onClick={() => setConfirming(item.id)}>
              {t('common.delete')}
            </button>
          )}
        </div>
      ))}
      <form className="option-row new" onSubmit={create}>
        <ColorPicker value={newColor} label={t('panel.newItemColor')} onChange={setNewColor} />
        <input
          className="input"
          value={newName}
          maxLength={nameMaxLength}
          placeholder={placeholder}
          aria-label={t('panel.newItemName')}
          onChange={(e) => setNewName(e.target.value)}
        />
        <button type="submit" className="btn sm" disabled={nameRequired && newName.trim() === ''}>
          {t('common.add')}
        </button>
      </form>
    </div>
  )
}

/** 名称输入框：失去焦点或按回车时，名称有变化才保存；保存失败时恢复原名称。 */
function NameInput({value, required, maxLength, placeholder, onSave}: {
  value: string
  required: boolean
  maxLength: number
  placeholder: string
  onSave: (name: string) => Promise<void>
}) {
  const [draft, setDraft] = useState(value)

  function commit() {
    const name = draft.trim()
    if (name === value || (required && name === '')) {
      setDraft(value)
      return
    }
    onSave(name).catch(() => setDraft(value))
  }

  return (
    <input
      className="input"
      value={draft}
      maxLength={maxLength}
      placeholder={placeholder}
      aria-label={t('common.name')}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          e.currentTarget.blur()
        }
      }}
    />
  )
}
