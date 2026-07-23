/**
 * E2E — Slice 6 : Escouade (Teammates).
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /squad se charge sans erreur
 * 2. L'API pages/teammates retourne HTTP 200
 * 3. Le titre "Escouade" est visible dans la page
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

test.describe('Slice 6 — Escouade / Teammates (DEMO_MODE)', () => {
  test("la page Escouade se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(playerPath('demo-player', 'squad'))
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API pages/teammates retourne HTTP 200", async ({ page }) => {
    await skipIfNoDemoData()
    const squadPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/teammates') && resp.status() === 200,
    )

    await page.goto(playerPath('demo-player', 'squad'))
    const resp = await squadPromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })

  test("le titre Escouade est affiché", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(playerPath('demo-player', 'squad'))
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('Escouade')
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto(playerPath('demo-player', 'squad'))
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })
})
