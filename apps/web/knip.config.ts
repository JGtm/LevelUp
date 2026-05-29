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
    'src/lib/api/types.ts',        // types OpenAPI partiellement consommés — bruit acceptable
    'src/routeTree.gen.ts',        // généré par TanStack Router
  ],
  ignoreDependencies: [
    'jsdom',                       // utilisé par vitest env config, pas importé dans le code source
    'globals',                     // eslint config uniquement
    'tailwindcss',                 // via le plugin Vite @tailwindcss/vite, pas d'import JS
    '@tailwindcss/typography',     // via @plugin dans src/styles/globals.css, pas d'import JS
    '@iarna/toml',                 // utilisé dans tools/ (racine repo), pas dans apps/web/src
  ],
}

export default config
