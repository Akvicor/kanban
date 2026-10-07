import {useEffect, useRef, useState, type DragEvent, type FormEvent, type ReactNode} from 'react'
import type {WriteResult} from '../api/client'
import {createLinkAttachment} from '../api/file'
import type {Attachment} from '../api/types'
import type {CardWrite} from '../card/detail/useCardWrite'
import {Dialog} from '../components/Dialog'
import {useSyncStore} from '../sync/hooks'
import {EntityType, type SyncStore} from '../sync/store'
import {uploadQueue} from '../upload/queue'
import {useT} from '../i18n'

/** 上传完成并新建附件后，用响应立即更新本地数据。 */
export function attachmentCreated(store: SyncStore) {
  return (result: WriteResult<Attachment>) => store.applyLocal(EntityType.Attachment, result.data.id, result.data, result.revision)
}

const URL_PATTERN = /^https?:\/\/\S+$/i

/** 粘贴不处理的情况：焦点在输入控件或编辑器中，或有弹窗、画廊打开。 */
function pasteHandledElsewhere(event: ClipboardEvent): boolean {
  const target = event.target instanceof Element ? event.target : null
  if (target?.closest('input, textarea, select, [contenteditable="true"], [contenteditable=""]')) return true
  return document.querySelector('.dialog, .pswp') !== null
}

/**
 * 卡片详情中添加附件的其他方式：把文件拖到卡片详情上，以及在卡片详情中粘贴。
 * 粘贴时，剪贴板里有文件就上传；内容是网址时建链接附件；其他文字弹窗输入文件名，存为 .txt 附件。
 * write 是卡片详情的写入函数（hook 在提供写入上下文的组件中使用，因此由调用方传入）。
 * 返回拖放需要的事件处理、是否正在拖入文件，以及粘贴文字时的弹窗。卡片只读时都不生效。
 */
export function useAttachmentInput(cardId: number | null, readOnly: boolean, write: CardWrite) {
  const store = useSyncStore()
  const [dragging, setDragging] = useState(false)
  const [pastedText, setPastedText] = useState<string | null>(null)
  const depth = useRef(0)
  const enabled = cardId !== null && !readOnly

  useEffect(() => {
    if (!enabled) return
    function onPaste(event: ClipboardEvent) {
      if (pasteHandledElsewhere(event) || !event.clipboardData || cardId === null) return
      const files = Array.from(event.clipboardData.files)
      if (files.length > 0) {
        event.preventDefault()
        uploadQueue.add(cardId, files, attachmentCreated(store))
        return
      }
      const text = event.clipboardData.getData('text/plain')
      if (!text.trim()) return
      event.preventDefault()
      if (URL_PATTERN.test(text.trim())) {
        void write(EntityType.Attachment, () => createLinkAttachment(cardId, text.trim(), ''))
      } else {
        setPastedText(text)
      }
    }
    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [enabled, cardId, store, write])

  const hasFiles = (event: DragEvent) => event.dataTransfer.types.includes('Files')
  const dropProps = enabled
    ? {
        onDragEnter: (event: DragEvent) => {
          if (!hasFiles(event)) return
          event.preventDefault()
          depth.current++
          setDragging(true)
        },
        onDragOver: (event: DragEvent) => {
          if (hasFiles(event)) event.preventDefault()
        },
        onDragLeave: (event: DragEvent) => {
          if (!hasFiles(event)) return
          depth.current = Math.max(0, depth.current - 1)
          if (depth.current === 0) setDragging(false)
        },
        onDrop: (event: DragEvent) => {
          if (!hasFiles(event)) return
          event.preventDefault()
          depth.current = 0
          setDragging(false)
          const files = Array.from(event.dataTransfer.files)
          if (files.length > 0 && cardId !== null) uploadQueue.add(cardId, files, attachmentCreated(store))
        },
      }
    : {}

  let dialog: ReactNode = null
  if (pastedText !== null && cardId !== null) {
    dialog = (
      <TextFileDialog
        text={pastedText}
        onClose={() => setPastedText(null)}
        onSubmit={(file) => uploadQueue.add(cardId, [file], attachmentCreated(store))}
      />
    )
  }
  return {dropProps, dragging: enabled && dragging, dialog}
}

/** 把粘贴的文字存为 .txt 附件：输入文件名，不带扩展名时补上 .txt。 */
function TextFileDialog({text, onSubmit, onClose}: {text: string; onSubmit: (file: File) => void; onClose: () => void}) {
  const t = useT()
  const [name, setName] = useState(t('upload.pastedText'))

  function submit(event: FormEvent) {
    event.preventDefault()
    const base = name.trim() || t('upload.pastedText')
    const fileName = /\.txt$/i.test(base) ? base : `${base}.txt`
    onSubmit(new File([text], fileName, {type: 'text/plain'}))
    onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('upload.saveTextTitle')} description={t('upload.saveTextDesc')}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('upload.fileName')}</span>
          <input className="input" value={name} maxLength={124} autoFocus onChange={(e) => setName(e.target.value)} />
        </label>
        <pre className="pasted-text">{text.length > 2000 ? `${text.slice(0, 2000)}…` : text}</pre>
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary">
            {t('common.save')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
