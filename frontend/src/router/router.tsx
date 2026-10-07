import {createBrowserRouter} from 'react-router-dom'
import {AppLayout} from '../layout/AppLayout'
import {UsersPage} from '../pages/admin/UsersPage'
import {BoardArchivePage} from '../pages/archive/BoardArchivePage'
import {PanelArchivePage} from '../pages/archive/PanelArchivePage'
import {BoardPage} from '../pages/board/BoardPage'
import {ChannelsPage} from '../pages/channels/ChannelsPage'
import {FilesPage} from '../pages/files/FilesPage'
import {HomePage} from '../pages/home/HomePage'
import {LoginPage} from '../pages/login/LoginPage'
import {SettingsPage} from '../pages/settings/SettingsPage'
import {RequireAdmin, RequireAuth} from './guards'

/** 页面路由。除登录页外都需要登录，用户管理只对管理员开放。 */
export const router = createBrowserRouter([
  {path: '/login', element: <LoginPage />},
  {
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      {index: true, element: <HomePage />},
      {path: 'board/:boardId', element: <BoardPage />},
      {path: 'board/:boardId/panel/:panelId', element: <BoardPage />},
      {path: 'board/:boardId/panel/:panelId/card/:cardId', element: <BoardPage />},
      {path: 'archive/boards', element: <BoardArchivePage />},
      {path: 'archive/panels', element: <PanelArchivePage />},
      {path: 'files', element: <FilesPage />},
      {path: 'channels', element: <ChannelsPage />},
      {path: 'settings', element: <SettingsPage />},
      {
        path: 'admin/users',
        element: (
          <RequireAdmin>
            <UsersPage />
          </RequireAdmin>
        ),
      },
      {path: '*', element: <HomePage />},
    ],
  },
])
