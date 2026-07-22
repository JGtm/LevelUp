import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import { visualizer } from 'rollup-plugin-visualizer'
import path from 'path'

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8000'

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [
    // Plugin TanStack Router — génère les types de routes depuis src/routes/
    // routeFileIgnorePattern : exclut les tests colocalisés (ex. __root.test.tsx)
    // du scan de routes — sinon le codegen avertit « does not export a Route ».
    TanStackRouterVite({
      target: 'react',
      autoCodeSplitting: true,
      routeFileIgnorePattern: '\\.test\\.tsx?$',
    }),
    react(),
    tailwindcss(),
    // Analyse du bundle : ANALYZE=true vite build → ouvre dist/stats.html
    ...(mode === 'analyze' ? [visualizer({ open: true, filename: 'dist/stats.html', gzipSize: true, brotliSize: true })] : []),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    // Proxy dev : redirige /api/** vers l'API Go configurée
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
      '/static': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    // Exclure les tests Playwright (e2e/) qui sont exécutés par leur propre runner
    exclude: ['**/node_modules/**', '**/e2e/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html'],
      reportsDirectory: './coverage',
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/test/**',
        'src/**/*.d.ts',
        'src/routes/**/*.lazy.tsx',
        'src/main.tsx',
      ],
    },
  },
}))
