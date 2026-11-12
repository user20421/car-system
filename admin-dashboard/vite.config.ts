import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  return {
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: env.VITE_PROXY_TARGET || 'https://localhost:8443',
          changeOrigin: true,
          secure: env.VITE_PROXY_SECURE !== 'false',
          ws: true,
        },
        '/health': {
          target: env.VITE_PROXY_TARGET || 'https://localhost:8443',
          changeOrigin: true,
          secure: env.VITE_PROXY_SECURE !== 'false',
        },
      },
    },
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: ['./src/tests/setup.ts'],
      css: true,
    },
  }
})
