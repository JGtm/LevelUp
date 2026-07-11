/**
 * E2E — Slice 0a : Shell, bootstrap et navigation de base.
 *
 * Prérequis : `make dev` doit être lancé (API :8000 + Vite :5173).
 * Variables : LEVELUP_DEMO_MODE=true dans .env.local
 *
 * Couverture :
 * 1. Montage du shell React sans erreur JS
 * 2. Bootstrap DEMO_MODE : joueur "DemoPlayer" visible
 * 3. Redirection index → /players/demo-player/home
 * 4. Navigation vers /setup quand pas de joueur (non applicable en DEMO_MODE)
 * 5. Header / NavBar présente
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 0a — Shell & Bootstrap (DEMO_MODE)', () => {
  test('le shell se monte sans erreur JS console', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/')
    // Attendre que le contenu React soit rendu (pas juste le HTML statique)
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')), // faux positif connu
    ).toHaveLength(0)
  })

  test('le bootstrap retourne le joueur démo', async ({ page }) => {
    await skipIfNoDemoData()
    // Intercepter et valider la réponse bootstrap
    const bootstrapPromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/bootstrap') && resp.status() === 200,
    )

    await page.goto('/')
    const bootstrapResp = await bootstrapPromise
    const data = await bootstrapResp.json()

    expect(data.setup_required).toBe(false)
    expect(data.current_player?.gamertag).toBe('DemoPlayer')
    expect(data.feature_flags?.demo_mode).toBe(true)
  })

  test('la racine "/" redirige vers /players/demo-player/home', async ({
    page,
  }) => {
    await skipIfNoDemoData()
    await page.goto('/')
    // Attendre la redirection TanStack Router
    await page.waitForURL(/\/players\/demo-player\/home/, { timeout: 8_000 })
    expect(page.url()).toContain('/players/demo-player/home')
  })

  test('la NavBar est visible après le bootstrap', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto('/')
    await page.waitForURL(/\/players\/demo-player\/home/, { timeout: 8_000 })

    // La NavBar doit être rendue (au moins un élément de navigation)
    const nav = page.locator('nav, [role="navigation"]').first()
    await expect(nav).toBeVisible()
  })

  test('changer de joueur reste dans le même shell', async ({ page }) => {
    await page.goto('/players/demo-player/home')
    await page.waitForLoadState('networkidle')

    // En DEMO_MODE il y a un seul joueur — la page ne doit pas planter
    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Cannot read')
  })
})
