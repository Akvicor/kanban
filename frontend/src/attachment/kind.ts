import type {FileInfo} from '../api/types'

/** 附件的预览方式。none 表示不能预览，只能下载。 */
export type PreviewKind = 'image' | 'pdf' | 'audio' | 'video' | 'csv' | 'markdown' | 'text' | 'none'

/** 文本和 Markdown 预览的大小上限，与参考项目一致。 */
export const TEXT_PREVIEW_LIMIT = 256 * 1024
/** CSV 预览的大小上限：整份文件需要读入页面解析。 */
export const CSV_PREVIEW_LIMIT = 10 * 1024 * 1024

/** 按扩展名当作文本预览的文件（服务端可能把它们识别为 application/octet-stream 以外的任何类型）。 */
const TEXT_EXTENSIONS = new Set([
  'txt', 'log', 'ini', 'conf', 'cfg', 'env', 'toml', 'yaml', 'yml', 'json', 'jsonc', 'xml', 'html', 'htm', 'css', 'scss', 'less',
  'js', 'mjs', 'cjs', 'jsx', 'ts', 'tsx', 'vue', 'svelte', 'go', 'mod', 'py', 'rb', 'php', 'java', 'kt', 'kts', 'scala', 'c', 'h',
  'cc', 'cpp', 'hpp', 'cs', 'rs', 'swift', 'm', 'mm', 'lua', 'pl', 'r', 'sql', 'sh', 'bash', 'zsh', 'fish', 'ps1', 'bat', 'cmd',
  'dockerfile', 'makefile', 'gradle', 'properties', 'tex', 'diff', 'patch', 'svg', 'tsv', 'proto', 'graphql', 'dart', 'ex', 'exs',
  'erl', 'hs', 'clj', 'vim', 'srt', 'vtt',
])

/** 文件名的扩展名（小写，不带点）。没有扩展名时，返回小写的完整文件名（如 makefile）。 */
export function extensionOf(name: string): string {
  const dot = name.lastIndexOf('.')
  return (dot > 0 ? name.slice(dot + 1) : name).toLowerCase()
}

/**
 * 判断附件的预览方式：生成了缩略图的是图片；PDF、音频、视频按服务端识别的类型；
 * CSV、Markdown 和代码等文本按扩展名。文本类是否真的能显示（大小、编码）由预览组件检查。
 */
export function previewKind(file: FileInfo, name: string): PreviewKind {
  const extension = extensionOf(name)
  if (file.image) return 'image'
  if (file.mime_type === 'application/pdf') return 'pdf'
  if (file.mime_type.startsWith('audio/')) return 'audio'
  if (file.mime_type.startsWith('video/')) return 'video'
  if (extension === 'csv') return 'csv'
  if (extension === 'md' || extension === 'markdown') return 'markdown'
  if (file.mime_type.startsWith('text/') || TEXT_EXTENSIONS.has(extension)) return 'text'
  return 'none'
}
