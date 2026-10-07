import {createContext, useCallback, useContext, useMemo, useState} from 'react'
import {useLocation} from 'react-router-dom'
import {t} from '../i18n'

/** 平板和手机宽度：侧栏从左侧滑出、覆盖在页面上。与 AppLayout.css 中的断点一致。 */
export const NARROW_QUERY = '(max-width: 1000px)'

/** 页面中控制侧栏的操作。 */
export interface SidebarControl {
  /** 电脑宽度下收起或展开侧栏；平板和手机宽度下滑出或收回侧栏。 */
  toggle: () => void
  /** 平板和手机宽度下收回侧栏。 */
  close: () => void
}

export const SidebarContext = createContext<SidebarControl | null>(null)

export function useSidebar(): SidebarControl {
  const control = useContext(SidebarContext)
  if (!control) {
    throw new Error(t('error.useSidebar'))
  }
  return control
}

/**
 * 侧栏状态。各种宽度下都默认收起，让看板（或主页）占满宽度，需要时点菜单按钮展开；
 * 展开状态不保存在任何设备上，重新打开页面后回到收起。
 * 电脑宽度下收起时侧栏滑出；平板和手机宽度下的展开只在当前页面有效，
 * 记为展开时的地址，地址变化（点了目录中的看板或页面入口）后自动收回。
 */
export function useSidebarState(): {collapsed: boolean; open: boolean; control: SidebarControl} {
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(true)
  const [openAt, setOpenAt] = useState<string | null>(null)
  const open = openAt === location.key

  const toggle = useCallback(() => {
    if (window.matchMedia(NARROW_QUERY).matches) {
      setOpenAt((current) => (current === location.key ? null : location.key))
      return
    }
    setCollapsed((current) => !current)
  }, [location.key])
  const close = useCallback(() => setOpenAt(null), [])

  const control = useMemo(() => ({toggle, close}), [toggle, close])
  return {collapsed, open, control}
}
