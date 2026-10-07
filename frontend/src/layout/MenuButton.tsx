import {useSidebar} from './sidebar'
import {useT} from '../i18n'

/** 收起或展开左侧目录的菜单按钮。看板页放在顶栏最左侧，其他页面放在标题左侧。 */
export function MenuButton() {
  const t = useT()
  const sidebar = useSidebar()
  return (
    <button type="button" className="icon-btn menu-btn" aria-label={t('nav.toggleSidebar')} title={t('nav.toggleSidebar')} onClick={sidebar.toggle}>
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
        <path d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>
  )
}
