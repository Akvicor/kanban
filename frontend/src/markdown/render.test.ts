import {describe, expect, it} from 'vitest'
import {INTERNAL_LINK_CLASS} from './links'
import {renderMarkdown} from './render'

function render(source: string): HTMLElement {
  const root = document.createElement('div')
  root.innerHTML = renderMarkdown(source)
  return root
}

describe('renderMarkdown', () => {
  it('自动识别网址，站内链接在页面内跳转并只显示路径，站外链接在新窗口打开', () => {
    const root = render(`见 https://example.com/a 和 ${window.location.origin}/board/1`)
    const [external, internal] = [...root.querySelectorAll('a')]
    expect(external.getAttribute('href')).toBe('https://example.com/a')
    expect(external.getAttribute('target')).toBe('_blank')
    expect(external.getAttribute('rel')).toBe('noopener noreferrer')
    expect(internal.classList.contains(INTERNAL_LINK_CLASS)).toBe(true)
    expect(internal.getAttribute('target')).toBeNull()
    expect(internal.textContent).toBe('/board/1')
  })

  it('原始 HTML 不生效，粘贴的 base64 图片可以显示', () => {
    const root = render('<script>alert(1)</script><img src=x onerror=alert(1)>\n\n![图](data:image/png;base64,iVBORw0KGgo= =20x10)')
    expect(root.querySelector('script')).toBeNull()
    expect(root.querySelector('img[onerror]')).toBeNull()
    const image = root.querySelector('img')!
    expect(image.getAttribute('src')).toBe('data:image/png;base64,iVBORw0KGgo=')
    expect(image.getAttribute('width')).toBe('20')
  })

  it('支持提示块、折叠块、文字颜色和代码高亮', () => {
    const root = render('{% note info %}\n\n说明\n\n{% endnote %}\n\n{% cut "更多" %}\n\n内容\n\n{% endcut %}\n\n{blue}(蓝字)\n\n```js\nconst a = 1\n```')
    expect(root.querySelector('.yfm-note')).not.toBeNull()
    expect(root.querySelector('.yfm-cut')).not.toBeNull()
    expect(root.querySelector('.yfm-colorify--blue')?.textContent).toBe('蓝字')
    expect(root.querySelector('code.hljs .hljs-keyword')?.textContent).toBe('const')
  })
})
