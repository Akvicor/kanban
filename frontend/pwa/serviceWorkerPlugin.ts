import {createHash} from 'node:crypto'
import {readFileSync} from 'node:fs'
import type {Plugin} from 'vite'

/** Service Worker 模板，占位符见模板文件。 */
const TEMPLATE = new URL('./sw.template.js', import.meta.url)

/**
 * 构建时生成站点根目录下的 sw.js（放在根目录，作用范围才能覆盖全部页面）：
 * - 预缓存列表取 index.html 引用的 /static/ 资源，即打开页面时必须加载的入口脚本和样式。
 * - 版本号取全部产物文件名和 index.html 内容的哈希。产物文件名带内容哈希，内容变化时版本随之变化，
 *   浏览器据此安装新的 Service Worker 并清理旧缓存。
 */
export function serviceWorker(): Plugin {
  return {
    name: 'kanban-service-worker',
    apply: 'build',
    enforce: 'post',
    generateBundle(_options, bundle) {
      const html = bundle['index.html']
      if (!html || html.type !== 'asset') {
        this.error('生成 Service Worker 时找不到 index.html')
      }
      const page = typeof html.source === 'string' ? html.source : new TextDecoder().decode(html.source)
      const precache = [...new Set([...page.matchAll(/(?:src|href)="(\/static\/[^"]+)"/g)].map((match) => match[1]))]
      const version = createHash('sha256')
        .update(Object.keys(bundle).sort().join('\n'))
        .update(page)
        .digest('hex')
        .slice(0, 16)
      const source = readFileSync(TEMPLATE, 'utf8').replace('__SW_VERSION__', version).replace('__SW_PRECACHE__', JSON.stringify(precache))
      this.emitFile({type: 'asset', fileName: 'sw.js', source})
    },
  }
}
