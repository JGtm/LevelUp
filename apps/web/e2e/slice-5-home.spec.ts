/**
 * E2E — Slice 5 : Accueil (Home Mission Control).
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /home se charge sans erreur
 * 2. L'API pages/home retourne HTTP 200
 * 3. Le joueur DemoPlayer est visible dans la page
 * 4. La page ne contient pas d'erreur fatale
 */
import { test, expect } from '@playwright/test'

test.describe('Slice 5 — Accueil / Home (DEMO_MODE)', () => {
  test("la page Home se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/home')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API pages/home retourne HTTP 200", async ({ page }) => {
    const homePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/home') && resp.status() === 200,
    )

    await page.goto('/players/demo-player/home')
    const resp = await homePromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })

  test("le joueur DemoPlayer est visible dans la page", async ({ page }) => {
    await page.goto('/players/demo-player/home')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('DemoPlayer')
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/home')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Cannot read')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })
})
