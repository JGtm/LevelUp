/**
 * E2E — Slice 7 : Synthèse.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /synthesis se charge sans erreur
 * 2. L'API pages/synthesis retourne HTTP 200
 * 3. La page se charge sans crash
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 7 — Synthèse (DEMO_MODE)', () => {
  test("la page Synthèse se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/synthesis')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API pages/synthesis retourne HTTP 200", async ({ page }) => {
    await skipIfNoDemoData()
    const synthPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/synthesis') && resp.status() === 200,
    )

    await page.goto('/players/demo-player/synthesis')
    const resp = await synthPromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })

  test("la page Synthèse affiche du contenu", async ({ page }) => {
    await page.goto('/players/demo-player/synthesis')
    await page.waitForLoadState('networkidle')

    // Vérifier que la page a contenu minimal
    const body = page.locator('body')
    await expect(body).not.toBeEmpty()
  })

  test("la page Synthèse ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/synthesis')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Internal Server Error')
    await expect(page.locator('body')).not.toContainText('Erreur critique')
  })
})
