import {useState} from 'react'
import {Link, NavLink, Outlet, useMatch} from 'react-router-dom'
import {Menu, MenuItem} from '../components/Menu'
import {useDirectoryActions} from '../directory/actions'
import {DirectoryActionsProvider} from '../directory/DirectoryActionsProvider'
import {DirectoryTree} from '../directory/DirectoryTree'
import {useMe} from '../session/context'
import {displayName} from '../session/account'
import {AccountsDialog} from '../session/accounts/AccountsDialog'
import {SidebarContext, useSidebarState} from './sidebar'
import './AppLayout.css'
import {useT} from '../i18n'

/**
 * 登录后的页面框架：左侧是目录和功能入口，右侧是当前页面。
 * 电脑宽度下侧栏可以收起；平板和手机宽度下侧栏默认收起，滑出时覆盖在页面上，点遮罩收回。
 */
export function AppLayout() {
  const {collapsed, open, control} = useSidebarState()
  const className = ['app', collapsed && 'sidebar-collapsed', open && 'sidebar-open'].filter(Boolean).join(' ')
  return (
    <DirectoryActionsProvider>
      <SidebarContext.Provider value={control}>
        <div className={className}>
          <Sidebar />
          <div className="sidebar-scrim" aria-hidden="true" onClick={control.close} />
          <main className="main">
            <Outlet />
          </main>
        </div>
      </SidebarContext.Provider>
    </DirectoryActionsProvider>
  )
}

function Sidebar() {
  const t = useT()
  const me = useMe()
  const [accountsOpen, setAccountsOpen] = useState(false)
  const actions = useDirectoryActions()
  const boardMatch = useMatch('/board/:boardId/*')
  const activeBoardId = boardMatch ? Number(boardMatch.params.boardId) : null
  const navClass = ({isActive}: {isActive: boolean}) => (isActive ? 'nav-item active' : 'nav-item')

  return (
    <aside className="sidebar">
      <Link to="/" state={{stay: true}} className="brand">
        <img className="logo" src="/icon-192.png" alt="" />
        {t('common.board')}
      </Link>
      <div className="side-title">
        <span>{t('directory.title')}</span>
        <Menu
          trigger={
            <button type="button" className="side-add" aria-label={t('nav.create')}>
              +
            </button>
          }
        >
          <MenuItem onSelect={() => actions.createBoard(null)}>{t('directory.createBoard')}</MenuItem>
          <MenuItem onSelect={() => actions.createFolder(null)}>{t('directory.createFolder')}</MenuItem>
        </Menu>
      </div>
      <div className="side-tree">
        <DirectoryTree activeBoardId={activeBoardId} />
      </div>
      <div className="side-bottom">
        <NavLink to="/archive/boards" className={navClass}>
          {t('nav.boardArchive')}
        </NavLink>
        <NavLink to="/archive/panels" className={navClass}>
          {t('nav.panelArchive')}
        </NavLink>
        <NavLink to="/files" className={navClass}>
          {t('nav.files')}
        </NavLink>
        <NavLink to="/channels" className={navClass}>
          {t('notify.channels')}
        </NavLink>
        {me.account.role === 'admin' && (
          <NavLink to="/admin/users" className={navClass}>
            {t('nav.users')}
          </NavLink>
        )}
        <NavLink to="/settings" className={navClass}>
          {t('nav.settings')}
        </NavLink>
        <button type="button" className="nav-item" onClick={() => setAccountsOpen(true)}>
          {t('account.button')}
        </button>
        <div className="side-user">{displayName(me.account)}</div>
      </div>
      <AccountsDialog open={accountsOpen} onOpenChange={setAccountsOpen} />
    </aside>
  )
}
