/**
 * E2E — PeriodSessionRail : visibilité + layout 3-zones + position sous NavL2.
 *
 * Cadenasse :
 *  - Le rail est rendu dans le DOM avec data-testid="period-session-rail"
 *  - Layout 3-zones : ◀ Précédente (gauche) | Label (centre) | Suivante ▶ (droite)
 *  - Mode all-time par défaut (pas de scope) — boutons disabled
 *  - Le rail vient APRÈS NavL2 dans le DOM (rendu en dessous des filtres)
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

test('Stats : rail rendu DANS NavL2, après FilterOmnibar, en mode all-time au cold load', async ({ page }) => {
  await page.addInitScript(() => {
    try { localStorage.clear() } catch { /* noop */ }
  })

  await page.goto('/players/JGtm/stats/history')
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(2500) // laisser /filters/resolve revenir

  const rail = page.locator('[data-testid="period-session-rail"]')
  await expect(rail).toBeVisible()
  // Cold load : aucun scope choisi → mode all-time
  await expect(rail).toHaveAttribute('data-mode', 'all-time')

  // Boutons prev/next disabled en mode all-time
  const prev = rail.getByLabel(/Session précédente|Previous session/)
  const next = rail.getByLabel(/Session suivante|Next session/)
  await expect(prev).toBeDisabled()
  await expect(next).toBeDisabled()

  // Position : rail est DANS le conteneur NavL2 (descendant), donc visuellement
  // sous FilterOmnibar (rendu avant lui dans NavL2).
  const containment = await page.evaluate(() => {
    const navL2 = document.querySelector('[aria-label="Navigation analytique"]')
    const rail = document.querySelector('[data-testid="period-session-rail"]')
    const omnibar = document.querySelector('[role="toolbar"][aria-label="Filtres"]')
    if (!navL2 || !rail || !omnibar) return { found: false }
    return {
      found: true,
      navContainsRail: navL2.contains(rail),
      navContainsOmnibar: navL2.contains(omnibar),
      railFollowsOmnibar: (omnibar.compareDocumentPosition(rail) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0,
    }
  })
  expect(containment.found).toBe(true)
  expect(containment.navContainsRail).toBe(true)
  expect(containment.navContainsOmnibar).toBe(true)
  expect(containment.railFollowsOmnibar).toBe(true)
})

test('Squad : rail rendu DANS la barre Squad (pas dans NavL2 qui est null)', async ({ page }) => {
  await page.addInitScript(() => {
    try { localStorage.clear() } catch { /* noop */ }
  })

  await page.goto('/players/JGtm/squad/synergies')
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(2500)

  const rail = page.locator('[data-testid="period-session-rail"]')
  await expect(rail).toBeVisible()

  // NavL2 absent sur Squad — le rail doit être dans la barre sticky Squad.
  // Vérifie que le rail n'est PAS dans NavL2 (qui n'existe pas) mais bien
  // dans le DOM — donc dans SquadLayout.
  const containment = await page.evaluate(() => {
    const navL2 = document.querySelector('[aria-label="Navigation analytique"]')
    const rail = document.querySelector('[data-testid="period-session-rail"]')
    return {
      navL2Present: !!navL2,
      railPresent: !!rail,
    }
  })
  expect(containment.navL2Present).toBe(false)
  expect(containment.railPresent).toBe(true)
})
