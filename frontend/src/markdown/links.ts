import type MarkdownIt from 'markdown-it'
import type Token from 'markdown-it/lib/token'

/** 指向本站页面的链接的类名。点击时在当前页面内跳转，见 MarkdownView。 */
export const INTERNAL_LINK_CLASS = 'md-internal-link'

/**
 * 链接的打开方式：指向本站的链接在当前页面内跳转，链接文字是完整地址时只显示路径；
 * 其他链接在新窗口打开，不带来源页信息。
 */
function processLink(token: Token, next: Token | undefined) {
  const href = token.attrGet('href')
  if (!href) return
  let url: URL
  try {
    url = new URL(href, window.location.href)
  } catch {
    return
  }
  if (url.origin === window.location.origin) {
    token.attrJoin('class', INTERNAL_LINK_CLASS)
    if (next?.type === 'text' && next.content === href) {
      next.content = `${url.pathname}${url.search}${url.hash}`
    }
  } else {
    token.attrSet('target', '_blank')
    token.attrSet('rel', 'noopener noreferrer')
  }
}

export function linkPlugin(md: MarkdownIt) {
  md.core.ruler.push('kanban_links', (state) => {
    for (const block of state.tokens) {
      block.children?.forEach((token, index, children) => {
        if (token.type === 'link_open') processLink(token, children[index + 1])
      })
    }
    return true
  })
}
