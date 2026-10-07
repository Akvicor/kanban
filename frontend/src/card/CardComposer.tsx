import {useState, type FormEvent, type KeyboardEvent} from 'react'
import './CardComposer.css'
import {useT} from '../i18n'

interface CardComposerProps {
  /** 新建一张卡片。失败时自行提示并抛出错误，输入框保留内容。 */
  onCreate: (title: string) => Promise<void>
  onClose: () => void
}

/** 列首或列尾的新建卡片输入框。回车创建后保持打开以便继续输入，Esc 或失去焦点且为空时关闭。 */
export function CardComposer({onCreate, onClose}: CardComposerProps) {
  const t = useT()
  const [title, setTitle] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(event?: FormEvent) {
    event?.preventDefault()
    const value = title.trim()
    if (!value || saving) return
    setSaving(true)
    try {
      await onCreate(value)
      setTitle('')
    } catch {
      // 失败由 onCreate 提示，保留输入以便重试。
    } finally {
      setSaving(false)
    }
  }

  function onKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      void submit()
    } else if (event.key === 'Escape') {
      onClose()
    }
  }

  return (
    <form className="composer" onSubmit={submit}>
      <textarea
        className="textarea composer-input"
        value={title}
        placeholder={t('card.composerHint')}
        aria-label={t('card.composerAria')}
        rows={2}
        autoFocus
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={onKeyDown}
        onBlur={() => title.trim() === '' && onClose()}
      />
      <div className="composer-actions">
        <button type="submit" className="btn sm primary" disabled={saving || title.trim() === ''}>
          {t('card.add')}
        </button>
        <button type="button" className="btn sm ghost" onMouseDown={(e) => e.preventDefault()} onClick={onClose}>
          {t('common.cancel')}
        </button>
      </div>
    </form>
  )
}
