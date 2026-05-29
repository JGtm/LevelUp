import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  // entry : auto-détecté par les plugins knip (Vite → main.tsx/vite.config,
  // Playwright → playwright.config, ESLint → eslint.config). Pas de liste manuelle.
  project: ['src/**/*.{ts,tsx}'],
  ignore: [
    'src/lib/api/generated.ts',   // généré par openapi-typescript
    'src/lib/api/types.ts',        // types OpenAPI partiellement consommés — bruit acceptable
  ],
  ignoreDependencies: [
    'tailwindcss',                 // via le plugin Vite @tailwindcss/vite, pas d'import JS
    '@tailwindcss/typography',     // via @plugin dans src/styles/globals.css, pas d'import JS
    '@iarna/toml',                 // utilisé dans tools/ (racine repo), pas dans apps/web/src
  ],
}

export default config
