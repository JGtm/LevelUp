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

test('PeriodSessionRail — rail visible en mode all-time + sous NavL2 + 3-zones', async ({ page }) => {
  // Reset localStorage AVANT le chargement pour repartir d'un filterContext vierge.
  await page.addInitScript(() => {
    try { localStorage.clear() } catch { /* noop */ }
  })

  await page.goto('/players/JGtm/stats/history')
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(2000) // laisser /filters/resolve revenir

  const rail = page.locator('[data-testid="period-session-rail"]')
  await expect(rail).toBeVisible()
  await expect(rail).toHaveAttribute('data-mode', 'all-time')

  // Boutons prev/next présents et disabled (mode all-time)
  const prev = rail.getByLabel(/Session précédente|Previous session/)
  const next = rail.getByLabel(/Session suivante|Next session/)
  await expect(prev).toBeDisabled()
  await expect(next).toBeDisabled()

  // Label central avec compteur de sessions
  await expect(rail).toContainText(/Toutes les sessions \(\d+\)|All sessions \(\d+\)/)

  // Position : rail rendu APRÈS NavL2 dans le DOM (donc visuellement en dessous).
  const order = await page.evaluate(() => {
    const navL2 = document.querySelector('[aria-label="Navigation analytique"]')
    const rail = document.querySelector('[data-testid="period-session-rail"]')
    if (!navL2 || !rail) return { found: false }
    const rel = navL2.compareDocumentPosition(rail)
    return {
      found: true,
      railFollowsNavL2: (rel & Node.DOCUMENT_POSITION_FOLLOWING) !== 0,
    }
  })
  expect(order.found).toBe(true)
  expect(order.railFollowsNavL2).toBe(true)
})
