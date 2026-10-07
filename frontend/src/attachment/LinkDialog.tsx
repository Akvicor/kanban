import {useState, type FormEvent} from 'react'
import {createLinkAttachment} from '../api/file'
import {useCardWrite} from '../card/detail/useCardWrite'
import {Dialog} from '../components/Dialog'
import {EntityType} from '../sync/store'
import {useT} from '../i18n'

/** 添加链接附件：网址必填，名称可以不填（取网址）。站点图标在后台抓取。挂载时即打开。 */
export function LinkDialog({cardId, initialUrl = '', onClose}: {cardId: number; initialUrl?: string; onClose: () => void}) {
  const t = useT()
  const write = useCardWrite()
  const [url, setUrl] = useState(initialUrl)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    const outcome = await write(EntityType.Attachment, () => createLinkAttachment(cardId, url.trim(), name.trim()))
    setSaving(false)
    if (outcome === 'ok') onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('attachment.addLink')}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('attachment.url')}</span>
          <input className="input" type="url" value={url} maxLength={2048} placeholder="https://" required autoFocus onChange={(e) => setUrl(e.target.value)} />
        </label>
        <label className="field">
          <span>{t('attachment.nameOptional')}</span>
          <input className="input" value={name} maxLength={128} placeholder={t('attachment.linkNameHint')} onChange={(e) => setName(e.target.value)} />
        </label>
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={saving}>
            {t('common.add')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
