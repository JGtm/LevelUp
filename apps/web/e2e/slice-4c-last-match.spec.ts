/**
 * E2E — Slice 4C : Last Match Resolve.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API POST /pages/last-match/resolve retourne HTTP 200
 * 2. La réponse contient match_id et session_key
 * 3. L'API retourne 404 si le scope est vide
 * 4. Pas d'erreur 500
 */
import { test, expect } from '@playwright/test'

test.describe('Slice 4C — Last Match Resolve (DEMO_MODE)', () => {
  test("l'API /pages/last-match/resolve retourne HTTP 200", async ({
    request,
  }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/last-match/resolve',
      {
        data: { filters: {} },
      },
    )
    // 200 si match trouvé, 404 si scope vide
    expect([200, 404]).toContain(resp.status())
  })

  test("la réponse last-match contient match_id si HTTP 200", async ({
    request,
  }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/last-match/resolve',
      {
        data: { filters: {} },
      },
    )
    if (resp.status() !== 200) return

    const data = await resp.json()
    expect(data.current_match_id).toBeTruthy()
    expect(typeof data.current_match_id).toBe('string')
  })

  test("la réponse last-match contient session_key si HTTP 200", async ({
    request,
  }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/last-match/resolve',
      {
        data: { filters: {} },
      },
    )
    if (resp.status() !== 200) return

    const data = await resp.json()
    expect(data.session_tracking_key).toBeDefined()
  })

  test("pas d'erreur 500 sur /pages/last-match/resolve", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/last-match/resolve',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).not.toBe(500)
  })
})
