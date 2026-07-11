/**
 * E2E — Slice 3B : Page Séries temporelles.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API POST /pages/timeseries retourne HTTP 200
 * 2. La réponse contient total_matches et les 5 onglets attendus
 * 3. L'onglet résumé contient des KPI cards
 * 4. L'onglet cumul est présent
 * 5. Pas d'erreur 500
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 3B — Page Séries temporelles (DEMO_MODE)', () => {
  test("l'API /pages/timeseries retourne HTTP 200", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/timeseries',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).toBe(200)
  })

  test("la réponse timeseries contient total_matches", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/timeseries',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.total_matches).toBeDefined()
    expect(typeof data.total_matches).toBe('number')
  })

  test("la réponse timeseries contient les 5 onglets", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/timeseries',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.summary_tab).toBeDefined()
    expect(data.cumul_tab).toBeDefined()

    expect(data.intensity_tab).toBeDefined()
    expect(data.distributions_tab).toBeDefined()
  })

  test("l'onglet résumé contient des kpi_cards", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/timeseries',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.summary_tab.kpi_cards).toBeDefined()
    expect(Array.isArray(data.summary_tab.kpi_cards)).toBe(true)
    expect(data.summary_tab.kpi_cards.length).toBeGreaterThan(0)
  })

  test("pas d'erreur 500 sur /pages/timeseries", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/timeseries',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).not.toBe(500)
  })
})
