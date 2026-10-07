import {useRef, useState, type ReactNode} from 'react'
import {attachmentUrl, deleteAttachment, renameAttachment, setCardCover} from '../api/file'
import type {Attachment, Card} from '../api/types'
import {useCardWrite} from '../card/detail/useCardWrite'
import {ConfirmDialog} from '../components/ConfirmDialog'
import {NameDialog} from '../directory/NameDialog'
import {useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {formatSize} from '../utils/size'
import {AttachmentItem} from './AttachmentItem'
import {LinkDialog} from './LinkDialog'
import {previewKind} from './kind'
import {openGallery} from './preview/gallery'
import {PreviewDialog, type PreviewTarget} from './preview/PreviewDialog'
import {uploadQueue, useUploads, type UploadItem} from '../upload/queue'
import {attachmentCreated} from './useAttachmentInput'
import './Attachment.css'
import {t, useT} from '../i18n'

/** 默认显示的附件数，其余点「显示全部」展开。 */
const COLLAPSED_COUNT = 4

type Pending = {type: 'link'} | {type: 'rename'; attachment: Attachment} | {type: 'delete'; attachment: Attachment} | {type: 'preview'; target: PreviewTarget}

/** 上传阶段在界面上的名称。 */
function phaseText(phase: UploadItem['phase']): string {
  return t(phase === 'waiting' ? 'upload.waiting' : phase === 'hashing' ? 'upload.hashing' : phase === 'uploading' ? 'upload.uploading' : 'upload.saving')
}

/**
 * 卡片详情中的附件：按添加时间从新到旧排列，默认显示前 4 个。
 * 可以上传文件（多选）、添加链接，点击图片进入画廊，链接在新窗口打开，其他文件在弹窗中预览。
 * 卡片只读时只能查看和下载。
 */
export function AttachmentSection({card, attachments, readOnly, timeZone}: {card: Card; attachments: Attachment[]; readOnly: boolean; timeZone: string}) {
  const t = useT()
  const store = useSyncStore()
  const write = useCardWrite()
  const uploads = useUploads(card.id)
  const fileInput = useRef<HTMLInputElement>(null)
  const [expanded, setExpanded] = useState(false)
  const [pending, setPending] = useState<Pending | null>(null)
  const own = attachments.filter((attachment) => attachment.card_id === card.id).sort((a, b) => b.id - a.id)
  const visible = expanded ? own : own.slice(0, COLLAPSED_COUNT)
  const images = own.filter((attachment) => attachment.file?.image)
  const close = () => setPending(null)

  function open(attachment: Attachment) {
    if (attachment.type === 'link') {
      window.open(attachment.url, '_blank', 'noopener,noreferrer')
      return
    }
    const file = attachment.file
    if (!file) return
    const kind = previewKind(file, attachment.name)
    if (kind === 'image') {
      openGallery(
        images.map((image) => ({src: attachmentUrl(image.id), width: image.file!.width, height: image.file!.height, name: image.name})),
        images.indexOf(attachment),
      )
      return
    }
    setPending({type: 'preview', target: {kind, url: attachmentUrl(attachment.id), name: attachment.name, size: file.size}})
  }

  let dialog: ReactNode = null
  if (pending?.type === 'link') {
    dialog = <LinkDialog cardId={card.id} onClose={close} />
  } else if (pending?.type === 'rename') {
    const {attachment} = pending
    dialog = (
      <NameDialog
        title={t('attachment.rename')}
        initialName={attachment.name}
        confirmText={t('common.save')}
        maxLength={128}
        onClose={close}
        onSubmit={async (name) => {
          if ((await write(EntityType.Attachment, () => renameAttachment(attachment.id, name))) !== 'ok') throw new Error(t('attachment.renameFailed'))
        }}
      />
    )
  } else if (pending?.type === 'delete') {
    const {attachment} = pending
    dialog = (
      <ConfirmDialog
        open
        title={t('attachment.deleteTitle', {name: attachment.name})}
        description={attachment.file ? t('attachment.deleteDescFile') : t('attachment.deleteDesc')}
        confirmText={t('common.delete')}
        danger
        onCancel={close}
        onConfirm={() => {
          close()
          void write(EntityType.Attachment, () => deleteAttachment(attachment.id), attachment.id)
        }}
      />
    )
  } else if (pending?.type === 'preview') {
    dialog = <PreviewDialog target={pending.target} onClose={close} />
  }

  return (
    <section className="section">
      <h5>
        {t('attachment.section')}
        {own.length > 0 && <span className="extra">{own.length}</span>}
      </h5>
      {own.length === 0 && uploads.length === 0 && <p className="hint">{t(readOnly ? 'attachment.none' : 'attachment.noneDropHint')}</p>}
      {uploads.length > 0 && (
        <ul className="uploads">
          {uploads.map((item) => (
            <li key={item.id} className={item.error ? 'upload failed' : 'upload'}>
              <span className="upload-name">{item.name}</span>
              <span className="upload-state">{item.error || `${phaseText(item.phase)} ${Math.round(item.ratio * 100)}% · ${formatSize(item.size)}`}</span>
              {!item.error && (
                <span className="progress" aria-hidden>
                  <span style={{width: `${Math.round(item.ratio * 100)}%`}} />
                </span>
              )}
              {item.error && (
                <button type="button" className="icon-btn sm" aria-label={t('common.remove')} onClick={() => uploadQueue.dismiss(item.id)}>
                  ×
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      <ul className="attachments">
        {visible.map((attachment) => (
          <AttachmentItem
            key={attachment.id}
            attachment={attachment}
            isCover={card.cover_attachment_id === attachment.id}
            readOnly={readOnly}
            timeZone={timeZone}
            onOpen={() => open(attachment)}
            onToggleCover={() =>
              void write(EntityType.Card, () => setCardCover(card.id, card.cover_attachment_id === attachment.id ? null : attachment.id))
            }
            onRename={() => setPending({type: 'rename', attachment})}
            onDelete={() => setPending({type: 'delete', attachment})}
          />
        ))}
      </ul>
      {own.length > COLLAPSED_COUNT && (
        <button type="button" className="btn sm ghost" onClick={() => setExpanded(!expanded)}>
          {expanded ? t('common.collapse') : t('attachment.showAll', {count: own.length})}
        </button>
      )}
      {!readOnly && (
        <div className="attachment-actions">
          <button type="button" className="btn sm" onClick={() => fileInput.current?.click()}>
            {t('upload.file')}
          </button>
          <button type="button" className="btn sm" onClick={() => setPending({type: 'link'})}>
            {t('attachment.addLink')}
          </button>
          <input
            ref={fileInput}
            type="file"
            multiple
            hidden
            aria-label={t('upload.choose')}
            onChange={(event) => {
              const files = Array.from(event.target.files ?? [])
              event.target.value = ''
              if (files.length > 0) uploadQueue.add(card.id, files, attachmentCreated(store))
            }}
          />
        </div>
      )}
      {dialog}
    </section>
  )
}
