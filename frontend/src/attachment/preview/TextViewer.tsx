import hljs from 'highlight.js'
import {useEffect, useMemo, useState} from 'react'
import MarkdownView from '../../markdown/MarkdownView'
import {extensionOf, TEXT_PREVIEW_LIMIT} from '../kind'
import {fetchText, type TextResult} from './fetchText'
import {useT} from '../../i18n'

/** 扩展名与 highlight.js 语言名不同的情况。其余扩展名直接作为语言名或别名查找。 */
const LANGUAGE_BY_EXTENSION: Record<string, string> = {
  htm: 'xml', html: 'xml', svg: 'xml', vue: 'xml', jsonc: 'json', mjs: 'javascript', cjs: 'javascript',
  h: 'c', hpp: 'cpp', cc: 'cpp', mm: 'objectivec', kts: 'kotlin', zsh: 'bash', sh: 'bash', ps1: 'powershell',
  bat: 'dos', cmd: 'dos', conf: 'ini', cfg: 'ini', env: 'ini', toml: 'ini', properties: 'properties', tex: 'latex',
  patch: 'diff', ex: 'elixir', exs: 'elixir', erl: 'erlang', hs: 'haskell', clj: 'clojure', mod: 'go', log: 'plaintext',
  txt: 'plaintext', tsv: 'plaintext', srt: 'plaintext', vtt: 'plaintext',
}

/** 按扩展名高亮文本；找不到对应语言时自动识别。 */
function highlight(text: string, name: string): string {
  const extension = extensionOf(name)
  const language = LANGUAGE_BY_EXTENSION[extension] ?? extension
  if (language === 'plaintext') {
    return hljs.highlight(text, {language: 'plaintext'}).value
  }
  if (hljs.getLanguage(language)) {
    return hljs.highlight(text, {language, ignoreIllegals: true}).value
  }
  return hljs.highlightAuto(text).value
}

/**
 * 文本和 Markdown 文件的预览：不超过 256KB 且是有效的 UTF-8 时显示，代码按扩展名语法高亮，
 * Markdown 按描述的渲染规则显示。
 */
export default function TextViewer({url, name, size, markdown}: {url: string; name: string; size: number; markdown: boolean}) {
  const t = useT()
  const [result, setResult] = useState<TextResult | null>(null)

  useEffect(() => {
    let cancelled = false
    void fetchText(url, size, TEXT_PREVIEW_LIMIT).then((value) => !cancelled && setResult(value))
    return () => {
      cancelled = true
    }
  }, [url, size])

  const html = useMemo(() => (result?.ok && !markdown ? highlight(result.text, name) : ''), [result, markdown, name])

  if (!result) return <p className="hint">{t('common.loading')}</p>
  if (!result.ok) return <p className="hint">{result.reason}</p>
  if (markdown) return <MarkdownView source={result.text} />
  return (
    <div className="yfm text-preview">
      <pre>
        <code className="hljs" dangerouslySetInnerHTML={{__html: html}} />
      </pre>
    </div>
  )
}
