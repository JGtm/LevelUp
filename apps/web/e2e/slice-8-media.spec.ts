/**
 * E2E — Slice 8 : Médias.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /media se charge sans erreur
 * 2. L'API POST pages/media retourne HTTP 200
 * 3. Le titre "Médias" est visible dans la page
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

const API_BASE = 'http://localhost:8000/api/v1'

test.describe('Slice 8 — Médias (DEMO_MODE)', () => {
  test("la page Médias se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/media')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API POST pages/media retourne HTTP 200", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      `${API_BASE}/players/demo-player/pages/media`,
      { data: {} },
    )

    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(data.items).toBeTruthy()
  })

  test("le titre Médias est affiché", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/players/demo-player/media')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('Médias')
  })

  test("la page Médias ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/media')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Internal Server Error')
    await expect(page.locator('body')).not.toContainText('Erreur critique')
  })
})
