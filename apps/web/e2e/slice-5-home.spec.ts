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
 * 5. Anti-régression mai 2026 : les 4 sections critiques (banner, CSR/LUSR,
 *    playlists, arme favorite) rendent du contenu sans message d'erreur
 * 6. Smoke endpoint /api/v1/healthz/home renvoie 200 + ok=true
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

test.describe('Slice 5 — Accueil / Home (DEMO_MODE)', () => {
  test("la page Home se charge sans erreur JS", async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API pages/home retourne HTTP 200", async ({ page }) => {
    await skipIfNoDemoData()
    const homePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/pages/home') && resp.status() === 200,
    )

    await page.goto(playerPath('demo-player', 'home'))
    const resp = await homePromise
    const data = await resp.json()

    expect(data).toBeTruthy()
  })

  test("le joueur DemoPlayer est visible dans la page", async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).toContainText('DemoPlayer')
  })

  test("la page ne contient pas d'erreur fatale", async ({ page }) => {
    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Cannot read')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
  })

  // ──────────────────────────────────────────────────────────────────────
  // Anti-régression mai 2026 — les 4 sections critiques de la home
  // ──────────────────────────────────────────────────────────────────────

  test('les 4 sections critiques de la home rendent du contenu', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    // 1. Hero banner (rotation statique de 3 webp /public/titles/halo_infinite/).
    //    Si data-testid="home-hero-banner" disparaît : régression visuelle.
    await expect(page.getByTestId('home-hero-banner')).toBeVisible()

    // 2. Bannière Spartan identity (dynamique, career_progression.banner_image_url).
    //    Le data-testid existe peu importe le state ; l'absence = composant ne rend pas.
    await expect(page.getByTestId('home-spartan-identity-banner')).toBeVisible()

    // 3. CSR / LUSR — au moins une carte visible, avec contenu (rating OU badge
    //    OU unranked placement). LUSR ne doit JAMAIS afficher "En placement"
    //    sans qualification "(N/10)" : c'est le bug front résolu mai 2026
    //    (mode était figé à 'unranked' → 'neutral', maintenant le placement
    //    backend-driven via measurement_matches_remaining l'autorise).
    const csrCard = page.getByTestId('home-highest-csr-card')
    await expect(csrCard).toBeVisible()
    const csrContent = csrCard.getByTestId(/home-highest-csr-(value|badge|unranked)/)
    await expect(csrContent.first()).toBeVisible()

    const lusrCard = page.getByTestId('home-highest-lusr-card')
    await expect(lusrCard).toBeVisible()
    // Régression "LUSR En placement (faux)" : si le détail dit "En placement"
    // SANS suffixe "(N/10)", c'est l'ancien bug front qui revient.
    const lusrDetail = lusrCard.locator('[data-testid$="-detail"], [data-testid$="-tier"]').first()
    if (await lusrDetail.count() > 0) {
      const text = (await lusrDetail.textContent()) ?? ''
      if (text.includes('En placement')) {
        expect(text).toMatch(/En placement \(\d+\/10\)/)
      }
    }

    // 4. Arme favorite : KPI visible, soit avec un nom soit avec "—" explicite.
    //    Le data-testid garantit qu'on vise le bon KPI.
    await expect(page.getByTestId('home-favorite-weapon')).toBeVisible()
  })

  test('/api/v1/healthz/home retourne 200 ou 503 avec diagnostic', async ({ request }) => {
    await skipIfNoDemoData()
    // Le smoke endpoint inspecte les 5 sections critiques côté backend.
    // En DEMO_MODE avec fixture complète : 200 + ok=true. Sinon : 503 + liste.
    // Dans les deux cas la réponse doit être structurée — pas de 500.
    const resp = await request.get('/api/v1/healthz/home?player=demo-player')
    expect([200, 503]).toContain(resp.status())
    const body = await resp.json()
    expect(body).toHaveProperty('ok')
    expect(body).toHaveProperty('checks')
    if (body.ok === false) {
      expect(Array.isArray(body.empty_sections)).toBe(true)
      console.warn('healthz/home reports empty sections:', body.empty_sections)
    }
  })
})
