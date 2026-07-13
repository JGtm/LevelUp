/**
 * E2E — Slice 3 : Historique des parties (Stats Phase A).
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. La page /stats/history se charge sans erreur
 * 2. L'API match-history/query retourne HTTP 200 avec des matchs
 * 3. Le titre "Historique" est visible dans la page
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData, skipObsoleteSpec } from './_helpers/demoData'

test.describe('Slice 3 — Historique des parties (DEMO_MODE)', () => {
  // La route /players/{slug}/stats/history n'existe plus (routes actuelles :
  // stats/{index,sessions,synthesis,timeseries}) → contenu principal "Not Found".
  test.beforeEach(() => {
    skipObsoleteSpec(
      'route /players/{slug}/stats/history supprimée (routes actuelles : stats/{index,sessions,synthesis,timeseries})',
    )
  })
  test("la page Historique se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/stats/history')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API match-history/query retourne des matchs", async ({ page }) => {
    await skipIfNoDemoData()
    const historyPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/match-history/query') &&
        resp.status() === 200,
    )

    await page.goto('/players/demo-player/stats/history')
    const resp = await historyPromise
    const data = await resp.json()

    expect(data.summary.total_matches_scoped).toBeGreaterThan(0)
  })

  test("le titre Historique est affiché", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/players/demo-player/stats/history')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('Historique')
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/stats/history')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Cannot read')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })
})
