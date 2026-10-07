import {StrictMode} from 'react'
import {createRoot} from 'react-dom/client'
import App from './App'
import {registerServiceWorker} from './pwa/registerServiceWorker'
import './theme/palettes.css'
import './styles/global.css'
import './styles/controls.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

registerServiceWorker()
