import '@gravity-ui/uikit/styles/styles.css'
import {
  configure,
  MarkdownEditorView,
  useMarkdownEditor,
  type MarkdownEditorInstance,
  type ReactRenderStorage,
  type ToolbarsPreset,
} from '@gravity-ui/markdown-editor'
import {ActionName} from '@gravity-ui/markdown-editor/_/bundle/config/action-names.js'
import {i18n} from '@gravity-ui/markdown-editor/_/i18n/i18n.js'
import {full} from '@gravity-ui/markdown-editor/_/modules/toolbars/presets.js'
import {ThemeProvider, Toaster, ToasterComponent, ToasterProvider} from '@gravity-ui/uikit'
import {useEffect, useLayoutEffect, useRef} from 'react'
import type {EditorMode} from '../api/types'
import {useLocale} from '../i18n'
import {ZH_KEYSETS} from './editorLocale'
import './MarkdownEditor.css'

// 编辑器界面语言跟随应用语言：英文用编辑器自带词条，简体中文用 ZH_KEYSETS，缺少的词条回退到英文。
const LANG = 'zh'
i18n.setFallbackLang('en')
for (const [keyset, data] of Object.entries(ZH_KEYSETS)) {
  i18n.registerKeyset(LANG, keyset, data)
}
configure({lang: LANG as never})

/**
 * 工具栏和「/」命令菜单中去掉的操作：文件附件、选项卡、可折叠标题。
 * 描述的渲染（render.ts）不支持这几种语法，插入后只能显示为原文。
 */
const REMOVED_ACTIONS = new Set<string>([ActionName.file, ActionName.filePopup, ActionName.tabs, ActionName.foldingHeading])

/** 在完整工具栏的基础上去掉不支持的操作，生成新的配置，不修改编辑器自带的预设。 */
function buildToolbars(preset: ToolbarsPreset): ToolbarsPreset {
  const items = Object.fromEntries(Object.entries(preset.items).filter(([id]) => !REMOVED_ACTIONS.has(id)))
  const orders = Object.fromEntries(
    Object.entries(preset.orders).map(([name, groups]) => [
      name,
      groups.map((group) =>
        group
          .filter((entry) => typeof entry !== 'string' || !REMOVED_ACTIONS.has(entry))
          .map((entry) => (typeof entry === 'string' ? entry : {...entry, items: entry.items.filter((id) => !REMOVED_ACTIONS.has(id))})),
      ),
    ]),
  )
  return {items, orders}
}

const TOOLBARS = buildToolbars(full)

/** 编辑器内部的提示（例如图片插入失败）使用 Gravity UI 的 Toaster，只在编辑器中提供。 */
const toaster = new Toaster()

/** 已经处理过 renderStorage 订阅的编辑器实例。StrictMode 下 effect 会重复执行，同一实例只处理一次。 */
const resyncedEditors = new WeakSet<MarkdownEditorInstance>()

/**
 * 让渲染 React 节点（图片等）的组件在订阅 renderStorage 后，按当前已登记的节点渲染一次。
 *
 * 编辑器库（@gravity-ui/markdown-editor 15.48）在外层组件的 useLayoutEffect 中创建所见即所得编辑区，
 * 图片等节点这时登记到 renderStorage 并发出 update；负责渲染它们的组件要到之后的 useEffect 才订阅 update，
 * 错过了这次通知，已有的图片显示为空白，直到再登记新的节点（例如粘贴图片）。打开编辑器和每次切换到所见即所得模式都会这样。
 * 这里在有组件订阅 update 时补发一次 update；放进微任务，等这一轮 effect 全部执行完再发。
 * 必须在这些组件订阅之前调用（编辑器组件的 useLayoutEffect 中）。
 * renderStorage 是编辑器实例的属性，不在库的公开类型中，升级库时需要复核。
 */
function resyncReactNodesOnSubscribe(editor: MarkdownEditorInstance) {
  if (resyncedEditors.has(editor)) return
  resyncedEditors.add(editor)
  const {renderStorage} = editor as MarkdownEditorInstance & {renderStorage: ReactRenderStorage}
  const subscribe = renderStorage.on.bind(renderStorage)
  renderStorage.on = ((type, listener) => {
    subscribe(type, listener)
    if (type === 'update') queueMicrotask(() => renderStorage.emit('update', null))
  }) as typeof renderStorage.on
}

/** 粘贴或插入的图片转换为 data URL，直接嵌入描述。 */
function readAsDataUrl(file: File): Promise<{url: string}> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve({url: String(reader.result)})
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

export interface MarkdownEditorProps {
  initialValue: string
  mode: EditorMode
  onChange: (value: string) => void
  /** Ctrl/Cmd + Enter。 */
  onSubmit: () => void
  /** Esc。 */
  onCancel: () => void
  /** 用户切换了所见即所得和源码模式。 */
  onModeChange: (mode: EditorMode) => void
}

/**
 * 卡片描述的 Markdown 编辑器，支持所见即所得和源码两种模式，带工具栏和「/」命令菜单。
 * 编辑器使用 Gravity UI 组件，主题限定在编辑器容器内（scoped），跟随当前配色的亮暗。
 */
export default function MarkdownEditor({initialValue, mode, onChange, onSubmit, onCancel, onModeChange}: MarkdownEditorProps) {
  const callbacks = useRef({onChange, onSubmit, onCancel, onModeChange})
  const locale = useLocale()
  useEffect(() => {
    i18n.setLang(locale === 'en' ? 'en' : LANG)
  }, [locale])
  useEffect(() => {
    callbacks.current = {onChange, onSubmit, onCancel, onModeChange}
  })

  const editor = useMarkdownEditor({
    md: {breaks: true, linkify: true},
    handlers: {uploadFile: readAsDataUrl},
    initial: {markup: initialValue, mode},
  })

  // layout effect 先于所有子组件的 useEffect 执行，渲染 React 节点的组件此后才订阅。
  useLayoutEffect(() => resyncReactNodesOnSubscribe(editor), [editor])

  useEffect(() => {
    const change = () => callbacks.current.onChange(editor.getValue())
    const submit = () => callbacks.current.onSubmit()
    const cancel = () => callbacks.current.onCancel()
    const modeChange = ({mode: next}: {mode: EditorMode}) => callbacks.current.onModeChange(next)
    editor.on('change', change)
    editor.on('submit', submit)
    editor.on('cancel', cancel)
    editor.on('change-editor-mode', modeChange)
    return () => {
      editor.off('change', change)
      editor.off('submit', submit)
      editor.off('cancel', cancel)
      editor.off('change-editor-mode', modeChange)
    }
  }, [editor])

  const theme = document.documentElement.dataset.palette === 'dark' ? 'dark' : 'light'
  return (
    <ThemeProvider theme={theme} scoped rootClassName="markdown-editor">
      <ToasterProvider toaster={toaster}>
        <MarkdownEditorView autofocus stickyToolbar editor={editor} toolbarsPreset={TOOLBARS} className="markdown-editor-view" />
        <ToasterComponent />
      </ToasterProvider>
    </ThemeProvider>
  )
}
