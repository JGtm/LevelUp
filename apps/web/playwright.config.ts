import { defineConfig, devices } from '@playwright/test'

/**
 * Configuration Playwright — tests E2E LevelUp.
 *
 * Cible : API FastAPI sur :8000 + Vite sur :5173.
 * Prérequis : `make dev` ou `make api` + `make web` en cours.
 *
 * Lance avec : `npx playwright test` ou `make test-e2e`
 */
export default defineConfig({
  testDir: './e2e',
  outputDir: '../../tests/e2e-results',
  fullyParallel: false, // API DuckDB mono-fichier → sériel pour éviter les locks
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'list',

  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'off',
    // Pas de JS disabled — l'app est 100% JS
    locale: 'fr-FR',
    // Origin header requis pour passer le middleware CSRF de l'API Go
    extraHTTPHeaders: {
      Origin: 'http://localhost:5173',
    },
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      // Les specs de régression VISUELLE vivent dans leur propre projet (`visual`) :
      // leurs baselines PNG sont liées à la plateforme de génération (rendu de police
      // et anti-aliasing diffèrent entre win32 et le runner ubuntu de la CI). Les
      // exclure ici évite de rendre la CI rouge sur des baselines non régénérées.
      testIgnore: /.*\.visual\.spec\.ts/,
    },
    {
      name: 'visual',
      testMatch: /.*\.visual\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        // Viewport HAUT et fixe : les pages scrollent dans `<main overflow-y-auto>`
        // (pas le body), donc `fullPage` ne capture rien de plus. Une hauteur fixe
        // rend la capture de contexte déterministe d'un run à l'autre.
        viewport: { width: 1440, height: 2000 },
        // 1x : neutralise le devicePixelRatio de la machine (ChartCard le passe à
        // ECharts via `opts.devicePixelRatio`) — sinon la taille des PNG varie.
        deviceScaleFactor: 1,
        colorScheme: 'dark',
      },
    },
  ],

  // Baselines de régression visuelle : versionnées à côté des specs. Le suffixe
  // `{platform}` laisse cohabiter des baselines générées sur un autre OS (une
  // future génération Linux pour la CI n'écrase pas celles de développement).
  snapshotPathTemplate: '{testDir}/visual/__screenshots__/{arg}-{platform}{ext}',

  // Asserts globaux : timeouts généreux pour les lazy loads
  expect: {
    timeout: 10_000,
    toHaveScreenshot: {
      // Les charts rendent en <canvas> : l'anti-aliasing n'est pas bit-à-bit
      // reproductible. `threshold` tolère l'écart colorimétrique par pixel,
      // `maxDiffPixelRatio` la part de pixels réellement différents. Volontairement
      // SERRÉ : ce harnais existe pour détecter un changement de moteur de rendu,
      // une tolérance large le rendrait aveugle. Toute diff doit être analysée,
      // jamais absorbée en élargissant ces seuils sans justification écrite.
      threshold: 0.15,
      maxDiffPixelRatio: 0.002,
      // Fige les animations CSS/Web Animations (ne couvre PAS le canvas ECharts —
      // celui-ci est traité par l'attente de stabilité, cf. _helpers/visual.ts).
      animations: 'disabled',
      caret: 'hide',
      scale: 'css',
    },
  },

  // Timeout de navigation
  timeout: 30_000,

  // webServer : ne JAMAIS démarrer les serveurs ici — ils sont gérés par `make dev`
  // Lancer `make dev` avant d'exécuter les tests E2E.
})
