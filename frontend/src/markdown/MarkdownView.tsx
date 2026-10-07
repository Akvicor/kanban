import '@diplodoc/transform/dist/css/yfm.css'
import '@diplodoc/transform/dist/js/yfm.js'
import {useEffect, useMemo, useRef, useState, type MouseEvent} from 'react'
import {useNavigate} from 'react-router-dom'
import {INTERNAL_LINK_CLASS} from './links'
import {renderMarkdown} from './render'
import './markdown.css'
import {useT} from '../i18n'

/** 超过这个高度的内容先折叠，点击「展开」查看全部。 */
const COLLAPSED_HEIGHT = 800

/**
 * 渲染 Markdown 描述。折叠块、代码复制按钮等交互由 YFM 运行时脚本处理；
 * 指向本站的链接在当前页面内跳转，其他链接在新窗口打开。
 */
export default function MarkdownView({source}: {source: string}) {
  const t = useT()
  const navigate = useNavigate()
  const html = useMemo(() => renderMarkdown(source), [source])
  const content = useRef<HTMLDivElement>(null)
  const [overflowing, setOverflowing] = useState(false)
  const [expanded, setExpanded] = useState(false)

  useEffect(() => {
    const element = content.current
    if (!element) return
    const observer = new ResizeObserver(() => setOverflowing(element.scrollHeight > COLLAPSED_HEIGHT))
    observer.observe(element)
    return () => observer.disconnect()
  }, [])

  function onClick(event: MouseEvent<HTMLDivElement>) {
    const link = (event.target as HTMLElement).closest<HTMLAnchorElement>(`a.${INTERNAL_LINK_CLASS}`)
    if (!link || event.ctrlKey || event.metaKey || event.shiftKey) return
    event.preventDefault()
    const url = new URL(link.href)
    navigate(`${url.pathname}${url.search}${url.hash}`)
  }

  const collapsed = overflowing && !expanded
  return (
    <div className="markdown-view">
      <div className={collapsed ? 'markdown-clip collapsed' : 'markdown-clip'} style={collapsed ? {maxHeight: COLLAPSED_HEIGHT} : undefined}>
        <div ref={content} className="yfm markdown" onClick={onClick} dangerouslySetInnerHTML={{__html: html}} />
      </div>
      {overflowing && (
        <button type="button" className="btn sm ghost markdown-toggle" onClick={() => setExpanded(!expanded)}>
          {expanded ? t('common.collapse') : t('markdown.expandAll')}
        </button>
      )}
    </div>
  )
}
