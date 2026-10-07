import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'
import {serviceWorker} from './pwa/serviceWorkerPlugin.ts'

export default defineConfig({
  plugins: [react(), serviceWorker()],
  server: {
    host: '0.0.0.0',
    port: 3001,
    strictPort: true,
    // 开发时接口和同步连接转发到本地后端，页面与接口同源。
    // 不改写 Host：后端只接受与页面同源的 WebSocket 连接，改写后 Origin 与 Host 不一致会被拒绝。
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:3000',
        ws: true,
      },
    },
  },
  build: {
    outDir: 'build',
    assetsDir: 'static',
    sourcemap: false,
    manifest: true,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/setupTests.ts',
    css: true,
  },
})
