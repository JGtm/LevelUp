/**
 * E2E — Slice 0b : Contrat de filtres.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API filters/resolve répond HTTP 200
 * 2. La réponse contient des options de filtres non vides (counts, available_options)
 * 3. La page home charge sans erreur
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

const API_BASE = 'http://localhost:8000/api/v1'

test.describe('Slice 0b — Contrat de filtres (DEMO_MODE)', () => {
  test('filters/resolve retourne HTTP 200 avec des options', async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      `${API_BASE}/players/demo-player/filters/resolve`,
      { data: { filter_mode: 'period' } },
    )

    expect(resp.status()).toBe(200)
    const data = await resp.json()

    expect(data.counts).toBeTruthy()
    expect(data.available_options).toBeTruthy()
    expect(typeof data.counts.total_matches_before_filters).toBe('number')
  })

  test('la réponse filters/resolve contient des session_options', async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      `${API_BASE}/players/demo-player/filters/resolve`,
      { data: { filter_mode: 'period' } },
    )

    expect(resp.status()).toBe(200)
    const data = await resp.json()

    expect(data.session_options).toBeTruthy()
    expect(Array.isArray(data.session_options.all_sessions)).toBe(true)
  })

  test('la page home se charge sans erreur de filtres', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
    await expect(page.locator('body')).not.toContainText('Erreur critique')
  })
})
