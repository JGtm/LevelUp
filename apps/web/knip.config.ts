import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  entry: [
    'src/main.tsx',
    'vite.config.ts',
    'playwright.config.ts',
    'eslint.config.js',
  ],
  project: ['src/**/*.{ts,tsx}'],
  ignore: [
    'src/lib/api/generated.ts',   // généré par openapi-typescript
    'src/routeTree.gen.ts',        // généré par TanStack Router
  ],
  ignoreDependencies: [
    'jsdom',     // utilisé par vitest env config, pas importé dans le code source
    'globals',   // eslint config uniquement
  ],
}

export default config
