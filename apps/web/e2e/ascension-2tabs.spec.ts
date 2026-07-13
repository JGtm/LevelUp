/**
 * E2E — Ascension refonte 2-onglets (Phases 5-8 + Option D).
 *
 * Prérequis : `make dev` lancé (API :8000 + Vite :5173) + LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. /ascension affiche les deux LayerSection (Prestige + Ascension)
 * 2. Click sur le tab "Réalisations" change l'URL + le contenu
 * 3. Click sur le tab "Profil & objectifs" ramène à l'index
 * 4. /objectifs (route legacy) redirige vers /ascension
 * 5. TipsTicker présent avec liens vers /help#glossary-entry-*
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData, skipObsoleteSpec } from './_helpers/demoData'

let PLAYER = 'demo-player'

test.beforeAll(async ({ request }) => {
  // Résout le slug du joueur courant depuis le bootstrap. Permet aux tests
  // de tourner sur instance demo (`demo-player`) ou sur instance réelle
  // (slug = current_player.player_slug retourné par /api/v1/bootstrap).
  const resp = await request.get('http://localhost:8000/api/v1/bootstrap')
  if (resp.ok()) {
    const data = await resp.json()
    if (data?.current_player?.player_slug) {
      PLAYER = data.current_player.player_slug
    }
  }
})

test.describe('Ascension — 2 onglets (Profil & objectifs + Réalisations)', () => {
  // Page Ascension redessinée : 3 onglets (Profil & objectifs / Entraînement /
  // Réalisations) ; le header "Ascension — Coaching d'amélioration" est désormais
  // sous l'onglet Entraînement, plus sur le landing. Spec à réécrire pour 3 onglets.
  test.beforeEach(() => {
    skipObsoleteSpec(
      "page Ascension redessinée (3 onglets ; header Coaching déplacé sous l'onglet Entraînement)",
    )
  })
  test('landing on /ascension renders H1, layer headers and tabs', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/ascension`)
    await page.waitForLoadState('networkidle')

    // H1 + sous-titre du layout
    await expect(page.locator('main h1', { hasText: 'Ascension' }).first()).toBeVisible()

    // Les deux headers de couche (LayerSection) — séparation visuelle Option D
    await expect(page.getByText('Prestige — Objectifs et arcs')).toBeVisible()
    await expect(page.getByText("Ascension — Coaching d'amélioration")).toBeVisible()

    // Les deux tabs sont présents
    await expect(page.getByRole('tab', { name: /Profil & objectifs/i })).toBeVisible()
    await expect(page.getByRole('tab', { name: /Réalisations/i })).toBeVisible()
  })

  test('tab switch /ascension → /ascension/realisations updates URL and content', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/ascension`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('tab', { name: /Réalisations/i }).click()
    await page.waitForURL(/\/ascension\/realisations$/, { timeout: 5_000 })
    expect(page.url()).toMatch(new RegExp(`/players/${PLAYER}/ascension/realisations$`))

    // Le contenu Réalisations contient toujours la section "Moments marquants"
    // (MomentsSection ne fait pas d'early return même si la liste est vide).
    await expect(page.getByText(/Moments marquants/i).first()).toBeVisible()
  })

  test('tab switch back /ascension/realisations → /ascension', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/ascension/realisations`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('tab', { name: /Profil & objectifs/i }).click()
    await page.waitForURL(/\/ascension$/, { timeout: 5_000 })
    expect(page.url()).toMatch(new RegExp(`/players/${PLAYER}/ascension$`))
  })

  test('legacy /objectifs redirects to /ascension', async ({ page }) => {
    await page.goto(`/players/${PLAYER}/objectifs`)
    // beforeLoad redirect — l'URL finale doit être /ascension sans /objectifs
    await page.waitForURL(/\/ascension(?!\/realisations)/, { timeout: 5_000 })
    expect(page.url()).toContain('/ascension')
    expect(page.url()).not.toContain('/objectifs')
  })

  test('TipsTicker is visible with stable links to /help glossary', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/ascension`)
    await page.waitForLoadState('networkidle')

    // Région labellisée "Astuces …"
    const ticker = page.getByRole('region', { name: /Astuces/i })
    await expect(ticker).toBeVisible()

    // Au moins un lien visible (les copies dupliquées sont aria-hidden) pointe
    // vers /help?tab=glossary#glossary-entry-<slug>
    const visibleLink = ticker.locator('a:not([aria-hidden="true"])').first()
    const href = await visibleLink.getAttribute('href')
    expect(href).toMatch(/^\/help\?tab=glossary#glossary-entry-/)
  })

  test('Prestige layer renders before Ascension layer (Option D ordering)', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/ascension`)
    await page.waitForLoadState('networkidle')

    const prestigeBox = await page.getByText('Prestige — Objectifs et arcs').boundingBox()
    const ascensionBox = await page.getByText("Ascension — Coaching d'amélioration").boundingBox()
    expect(prestigeBox).not.toBeNull()
    expect(ascensionBox).not.toBeNull()
    // Prestige est au-dessus d'Ascension verticalement (Option D : autonomie d'abord)
    expect(prestigeBox!.y).toBeLessThan(ascensionBox!.y)
  })
})
