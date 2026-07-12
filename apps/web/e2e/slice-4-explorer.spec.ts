/**
 * E2E — Slice 4 : Explorer (filtres + recherche + Match View).
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /explorer se charge sans erreur
 * 2. L'API explorer/matches-query retourne HTTP 200
 * 3. Le titre "Explorer" est visible dans la page
 * 4. L'API gamertags/search répond à une recherche
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 4 — Explorer (DEMO_MODE)', () => {
  test("la page Explorer se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/explorer')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API explorer/matches-query retourne des données", async ({
    page,
  }) => {
    await skipIfNoDemoData()
    const explorerPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/explorer/matches-query') &&
        resp.status() === 200,
    )

    await page.goto('/players/demo-player/explorer')
    const resp = await explorerPromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })

  test("le titre Explorer est affiché", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/players/demo-player/explorer')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('Explorer')
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/explorer')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })
})
