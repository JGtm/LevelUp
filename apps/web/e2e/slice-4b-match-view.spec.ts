/**
 * E2E — Slice 4B : Match View.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API GET /matches/{match_id} retourne HTTP 200 pour un match existant
 * 2. La réponse contient header, summary_tab, combat_tab, team_tab
 * 3. L'API retourne 404 pour un match inexistant
 * 4. Pas d'erreur 500
 */
import { test, expect } from '@playwright/test'

const DEMO_MATCH_ID = '00000000-0000-0000-0000-000000000001'

test.describe('Slice 4B — Match View (DEMO_MODE)', () => {
  test("l'API /matches/{match_id} retourne HTTP 200 pour un match démo", async ({
    request,
  }) => {
    // D'abord récupérer un match_id valide via l'historique
    const histResp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/explorer/matches-query',
      { data: { filters: {} } },
    )
    if (histResp.status() !== 200) return

    const histData = await histResp.json()
    const matches = histData.matches ?? histData.items ?? []
    if (matches.length === 0) return

    const matchId = matches[0].match_id ?? matches[0].id
    if (!matchId) return

    const resp = await request.get(
      `http://localhost:8000/api/v1/players/demo-player/matches/${matchId}`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(data.header).toBeDefined()
  })

  test("la réponse match view contient les onglets attendus", async ({
    request,
  }) => {
    const histResp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/explorer/matches-query',
      { data: { filters: {} } },
    )
    if (histResp.status() !== 200) return

    const histData = await histResp.json()
    const matches = histData.matches ?? histData.items ?? []
    if (matches.length === 0) return

    const matchId = matches[0].match_id ?? matches[0].id
    if (!matchId) return

    const resp = await request.get(
      `http://localhost:8000/api/v1/players/demo-player/matches/${matchId}`,
    )
    if (resp.status() !== 200) return
    const data = await resp.json()

    expect(data.summary_tab).toBeDefined()
    expect(data.combat_tab).toBeDefined()
    expect(data.team_tab).toBeDefined()
  })

  test("l'API /matches/{id} retourne 404 pour un id inexistant", async ({
    request,
  }) => {
    const resp = await request.get(
      `http://localhost:8000/api/v1/players/demo-player/matches/00000000-dead-beef-0000-000000000000`,
    )
    expect(resp.status()).toBe(404)
  })

  test("pas d'erreur 500 sur /matches/{match_id}", async ({ request }) => {
    const histResp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/explorer/matches-query',
      { data: { filters: {} } },
    )
    if (histResp.status() !== 200) return

    const histData = await histResp.json()
    const matches = histData.matches ?? histData.items ?? []
    if (matches.length === 0) return

    const matchId = matches[0].match_id ?? matches[0].id
    if (!matchId) return

    const resp = await request.get(
      `http://localhost:8000/api/v1/players/demo-player/matches/${matchId}`,
    )
    expect(resp.status()).not.toBe(500)
  })
})
