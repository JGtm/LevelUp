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
    },
  ],

  // Asserts globaux : timeouts généreux pour les lazy loads
  expect: {
    timeout: 10_000,
  },

  // Timeout de navigation
  timeout: 30_000,

  // webServer : ne JAMAIS démarrer les serveurs ici — ils sont gérés par `make dev`
  // Lancer `make dev` avant d'exécuter les tests E2E.
})
