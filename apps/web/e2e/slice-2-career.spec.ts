/**
 * E2E — Slice 2 : Page Carrière.
 *
 * Prérequis : `make dev` doit être lancé (API :8000 + Vite :5173).
 * Variables : LEVELUP_DEMO_MODE=true dans .env.local
 *
 * Couverture :
 * 1. La page /players/demo-player/career se charge sans erreur
 * 2. L'API retourne un rang valide (HTTP 200)
 * 3. Le rang du joueur est affiché dans la page (tier localisé FR/EN : « Or IV »/« Gold IV »)
 * 4. Pas d'erreur fatale React dans la console
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 2 — Page Carrière (DEMO_MODE)', () => {
  test("la page Carrière se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/players/demo-player/career')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API career retourne HTTP 200 avec les données du joueur", async ({
    page,
  }) => {
    await skipIfNoDemoData()
    const careerPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/players/demo-player/pages/career') &&
        !resp.url().includes('top-matches') &&
        !resp.url().includes('encounters') &&
        resp.status() === 200,
    )

    await page.goto('/players/demo-player/career')
    const careerResp = await careerPromise
    const data = await careerResp.json()

    expect(data.summary).toBeTruthy()
    expect(data.summary.rank_number).toBeGreaterThan(0)
    expect(data.summary.xp_total).toBeGreaterThan(0)
  })

  test("le rang du joueur est visible dans la page", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/players/demo-player/career')
    await page.waitForLoadState('networkidle')

    // Le tier Gold du demo-player doit apparaître quelque part dans la page,
    // LOCALISÉ à l'affichage (localizeTierName) : « Or IV » sous UI FR (défaut),
    // « Gold IV » sous UI EN. Pas d'ancre de tête (\b) : le textContent du body
    // concatène les éléments adjacents SANS espace (« Arène classéeOr I ») — entre
    // deux caractères de mot, pas de word boundary possible. Les faux positifs
    // restent écartés sans elle : regex sensible à la casse (« décor », « or »
    // minuscules non matchés) et espace exigé APRÈS « Or »/« Gold » suivi d'un
    // sous-palier romain (« Ordre I », « Score IV » non matchés).
    await expect(page.locator('body')).toContainText(/(Gold|Or)\s+(I|II|III|IV|V|VI)\b/)
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto('/players/demo-player/career')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Cannot read')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })

  test("le titre Carrière est affiché", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/players/demo-player/career')
    await page.waitForLoadState('networkidle')

    // Le heading ou titre de la page doit contenir "Carrière"
    await expect(page.locator('body')).toContainText('Carrière')
  })
})
