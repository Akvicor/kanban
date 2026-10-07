import {attachmentThumbnailUrl, attachmentUrl} from '../api/file'
import type {Attachment} from '../api/types'
import {Menu, MenuItem, MenuSeparator} from '../components/Menu'
import {formatDateTime} from '../utils/datetime'
import {formatSize} from '../utils/size'
import {extensionOf} from './kind'
import {useT} from '../i18n'

interface AttachmentItemProps {
  attachment: Attachment
  isCover: boolean
  readOnly: boolean
  timeZone: string
  onOpen: () => void
  onToggleCover: () => void
  onRename: () => void
  onDelete: () => void
}

/** 用浏览器下载文件，文件名为附件名称。 */
export function download(url: string, name: string) {
  const link = document.createElement('a')
  link.href = url
  link.download = name
  document.body.appendChild(link)
  link.click()
  link.remove()
}

/**
 * 附件列表中的一项：缩略图（图片显示缩略图，链接显示站点图标，其他显示扩展名）、名称、大小和添加时间，
 * 以及下载、设为或取消封面、重命名、删除的菜单。
 */
export function AttachmentItem({attachment, isCover, readOnly, timeZone, onOpen, onToggleCover, onRename, onDelete}: AttachmentItemProps) {
  const t = useT()
  const {file} = attachment
  let thumb
  if (file?.image) {
    thumb = <img src={attachmentThumbnailUrl(attachment.id, 360)} alt="" loading="lazy" />
  } else if (attachment.type === 'link') {
    thumb = attachment.favicon ? <img className="favicon" src={attachment.favicon} alt="" /> : <span className="thumb-text">🔗</span>
  } else {
    thumb = <span className="thumb-text">{extensionOf(attachment.name).slice(0, 4).toUpperCase() || '—'}</span>
  }
  const meta = [file ? formatSize(file.size) : attachment.url.replace(/^https?:\/\//, ''), formatDateTime(attachment.created_at, timeZone)]

  return (
    <li className="attachment">
      <button type="button" className={isCover ? 'attachment-thumb cover' : 'attachment-thumb'} aria-label={t('common.openName', {name: attachment.name})} onClick={onOpen}>
        {thumb}
      </button>
      <div className="attachment-info">
        <button type="button" className="attachment-name" onClick={onOpen}>
          {attachment.name}
        </button>
        <span className="attachment-meta">{meta.join(' · ')}</span>
      </div>
      {isCover && <span className="tag">{t('attachment.cover')}</span>}
      <Menu
        trigger={
          <button type="button" className="icon-btn sm" aria-label={t('common.menuOf', {name: attachment.name})}>
            ⋯
          </button>
        }
      >
        {file && <MenuItem onSelect={() => download(attachmentUrl(attachment.id), attachment.name)}>{t('common.download')}</MenuItem>}
        {file?.image && !readOnly && <MenuItem onSelect={onToggleCover}>{isCover ? t('attachment.unsetCover') : t('attachment.setCover')}</MenuItem>}
        {!readOnly && <MenuItem onSelect={onRename}>{t('common.rename')}</MenuItem>}
        {!readOnly && (
          <>
            <MenuSeparator />
            <MenuItem danger onSelect={onDelete}>
              {t('common.delete')}
            </MenuItem>
          </>
        )}
        {attachment.type === 'link' && readOnly && <MenuItem onSelect={onOpen}>{t('attachment.openLink')}</MenuItem>}
      </Menu>
    </li>
  )
}
