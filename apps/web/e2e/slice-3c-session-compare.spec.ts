/**
 * E2E — Slice 3C : Comparaison de sessions.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API POST /pages/session-compare retourne HTTP 200
 * 2. La réponse contient available_sessions et metrics
 * 3. La réponse est correcte même sans session_a/session_b fournis
 * 4. Pas d'erreur 500
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData, skipObsoleteSpec } from './_helpers/demoData'

test.describe('Slice 3C — Comparaison de sessions (DEMO_MODE)', () => {
  // L'endpoint POST /pages/session-compare a été supprimé côté Go (couvert par
  // /pages/timeseries — cf. openapi.yaml) → 404. Spec à réécrire sur /pages/timeseries.
  test.beforeEach(() => {
    skipObsoleteSpec(
      'endpoint POST /pages/session-compare supprimé (couvert par /pages/timeseries — cf. openapi.yaml)',
    )
  })
  test("l'API /pages/session-compare retourne HTTP 200", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/session-compare',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).toBe(200)
  })

  test("la réponse session-compare contient available_sessions", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/session-compare',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.available_sessions).toBeDefined()
    expect(Array.isArray(data.available_sessions)).toBe(true)
  })

  test("la réponse session-compare contient metrics", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/session-compare',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.metrics).toBeDefined()
    expect(Array.isArray(data.metrics)).toBe(true)
  })

  test("session_a et session_b sont retournés si données disponibles", async ({
    request,
  }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/session-compare',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    // Si des sessions sont disponibles, session_a et session_b doivent être présents
    if (data.available_sessions.length >= 2) {
      expect(data.session_a).toBeTruthy()
      expect(data.session_b).toBeTruthy()
    }
  })

  test("pas d'erreur 500 sur /pages/session-compare", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/session-compare',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).not.toBe(500)
  })
})
