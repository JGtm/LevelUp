/**
 * E2E — Slice 1 : Setup / Settings.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /setup se charge sans erreur
 * 2. L'API setup/status retourne HTTP 200
 * 3. La page /settings se charge avec le titre attendu
 * 4. L'API settings retourne HTTP 200 avec une config valide
 */
import { test, expect } from '@playwright/test'

const API_BASE = 'http://localhost:8000/api/v1'

test.describe('Slice 1 — Setup / Settings (DEMO_MODE)', () => {
  test('la page /setup se charge sans erreur JS', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/setup')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API setup/status retourne HTTP 200", async ({ request }) => {
    const resp = await request.get(`${API_BASE}/setup/status`)

    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(data).toBeTruthy()
  })

  test('la page /settings affiche le titre Paramètres', async ({ page }) => {
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('Paramètres')
  })

  test("l'API settings retourne HTTP 200 avec une config", async ({ page }) => {
    const settingsPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/settings') && resp.status() === 200,
    )

    await page.goto('/settings')
    const resp = await settingsPromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })
})
