import transform from '@diplodoc/transform'
import {t} from '../i18n'
import checkbox from '@diplodoc/transform/lib/plugins/checkbox'
import code from '@diplodoc/transform/lib/plugins/code'
import cut from '@diplodoc/transform/lib/plugins/cut'
import deflist from '@diplodoc/transform/lib/plugins/deflist'
import imsize from '@diplodoc/transform/lib/plugins/imsize'
import monospace from '@diplodoc/transform/lib/plugins/monospace'
import notes from '@diplodoc/transform/lib/plugins/notes'
import sup from '@diplodoc/transform/lib/plugins/sup'
import table from '@diplodoc/transform/lib/plugins/table'
import {defaultOptions as defaultSanitizeOptions} from '@diplodoc/transform/lib/sanitize'
import {emojiDefs} from '@gravity-ui/markdown-editor/_/bundle/emoji.js'
import color from '@gravity-ui/markdown-editor/markdown-it/color.js'
import emoji from '@gravity-ui/markdown-editor/markdown-it/emoji.js'
import ins from '@gravity-ui/markdown-editor/markdown-it/ins.js'
import mark from '@gravity-ui/markdown-editor/markdown-it/mark.js'
import sub from '@gravity-ui/markdown-editor/markdown-it/sub.js'
import type MarkdownIt from 'markdown-it'
import {linkPlugin} from './links'

/**
 * 卡片描述的 Markdown 渲染（YFM，与编辑器使用同一套语法）。
 *
 * 支持的语法：CommonMark 和 GFM 表格、删除线、自动链接，换行即换行；
 * 下划线 ++、高亮 ==、上标 ^、下标 ~、等宽 ##、文字颜色 {red}(文字)、表情 :smile:；
 * 图片尺寸 =宽x高、多行表格 #|、提示块 {% note %}、折叠块 {% cut %}、定义列表、任务复选框 [ ]；
 * 代码块按语言高亮并带复制按钮。
 *
 * 原始 HTML 不生效，渲染结果再经过白名单清理，图片地址允许 http、https 和 data（粘贴的图片以 base64 嵌入）。
 */

/** 文字颜色的类名前缀，与编辑器一致，样式见 markdown.css。 */
const COLOR_CLASS_NAME = 'yfm-colorify'

const plugins = [
  ins,
  mark,
  sub,
  (md: MarkdownIt) => md.use(emoji, {defs: emojiDefs}),
  color,
  sup,
  monospace,
  code,
  (md: MarkdownIt) => md.use(imsize, {enableInlineStyling: true}),
  table,
  (md: MarkdownIt) => md.use(notes, {notesAutotitle: false, log: console}),
  cut,
  deflist,
  checkbox,
  linkPlugin,
]

const sanitizeOptions = {
  ...defaultSanitizeOptions,
  allowedSchemesByTag: {img: ['http', 'https', 'data']},
}

/** 把 Markdown 渲染为清理过的 HTML。渲染出错时返回错误说明，不抛出。 */
export function renderMarkdown(source: string): string {
  try {
    return transform(source, {
      plugins,
      breaks: true,
      linkify: true,
      sanitizeOptions,
      defaultClassName: COLOR_CLASS_NAME,
    }).result.html
  } catch (error) {
    const div = document.createElement('div')
    div.textContent = t('markdown.renderError', {error: String(error)})
    return div.innerHTML
  }
}
