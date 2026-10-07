import {lazy, Suspense} from 'react'
import {Dialog} from '../../components/Dialog'
import type {PreviewKind} from '../kind'
import './Preview.css'
import {useT} from '../../i18n'

// 文本高亮、Markdown 渲染和 CSV 解析体积较大，打开对应预览时才加载。
const TextViewer = lazy(() => import('./TextViewer'))
const CsvViewer = lazy(() => import('./CsvViewer'))

export interface PreviewTarget {
  kind: Exclude<PreviewKind, 'image'>
  url: string
  name: string
  size: number
}

/** 非图片附件的预览：PDF 用浏览器内置查看器，音视频用原生播放器，CSV、文本和 Markdown 在页面中显示。 */
export function PreviewDialog({target, onClose}: {target: PreviewTarget; onClose: () => void}) {
  const t = useT()
  const {kind, url, name, size} = target
  let body
  switch (kind) {
    case 'pdf':
      body = <iframe className="pdf-preview" src={url} title={name} />
      break
    case 'audio':
      body = <audio className="media-preview" src={url} controls autoPlay />
      break
    case 'video':
      body = <video className="media-preview" src={url} controls autoPlay />
      break
    case 'csv':
      body = <CsvViewer url={url} size={size} />
      break
    case 'markdown':
    case 'text':
      body = <TextViewer url={url} name={name} size={size} markdown={kind === 'markdown'} />
      break
    default:
      body = <p className="hint">{t('attachment.noPreview')}</p>
  }
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={name} className="preview-dialog">
      <div className="preview-body">
        <Suspense fallback={<p className="hint">{t('common.loading')}</p>}>{body}</Suspense>
      </div>
      <div className="form-actions">
        <a className="btn" href={url} download={name}>
          {t('common.download')}
        </a>
        <button type="button" className="btn" onClick={onClose}>
          {t('common.close')}
        </button>
      </div>
    </Dialog>
  )
}
