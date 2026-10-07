import {RouterProvider} from 'react-router-dom'
import {router} from './router/router'
import {SessionProvider} from './session/SessionProvider'

/** 应用根组件：登录状态在最外层，页面由路由决定。 */
export default function App() {
  return (
    <SessionProvider>
      <RouterProvider router={router} />
    </SessionProvider>
  )
}
