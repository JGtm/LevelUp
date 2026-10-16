/**
 * E2E - Engagement (Phase 4 plan engagement).
 *
 * Prerequis : API + Vite demarres (`make dev`) ; la fixture demo synthetique
 * (`levelup seed-demo --synthetic`) porte les colonnes engagement et les evenements
 * des matchs, donc ces tests s'executent en CI comme en local.
 *
 * Couverture :
 *   1. POST /engagement/timeseries       (Mock 11 Timeseries intensity tab)
 *   2. GET  /matches/{match_id}/engagement (Mock 10 Match View team tab)
 *   3. GET  /pages/squad/v2/engagement   (Mock 15 v2 Squad page)
 *   4. GET  /engagement_profile          (Settings + Squad overlay)
 *
 * Configuration :
 *   - E2E_API_URL : racine de l'API, SANS `/api/v1` (defaut http://localhost:8000)
 *   - E2E_PLAYER_SLUG : joueur cible (defaut demo-player, cf. _helpers/demoData.ts)
 */
import { test, expect, type APIRequestContext } from '@playwright/test'
import { skipIfNoDemoData, E2E_API_BASE, E2E_PLAYER_SLUG } from './_helpers/demoData'

const API = `${E2E_API_BASE}/api/v1`
const PLAYER_SLUG = E2E_PLAYER_SLUG
const JSON_HEADERS = { 'Content-Type': 'application/json', Origin: 'http://localhost:5173' }

// `limit` plafonne a 200 matchs calcules cote serveur (engagementWorkCap) : le demander
// force la granularite "match" tant que l'historique tient dans ce plafond. En dessous,
// un historique plus long que `limit` est agrege par session/semaine/mois (match_id vide).
const TIMESERIES_LIMIT = 200

interface EngagementPoint {
  match_id: string
  pace_joueur: number
  pace_team: number
  pace_attendu: number
  pace_lobby: number
}

interface EngagementTimeseries {
  granularity: string
  points: EngagementPoint[]
  total_matches: number
}

async function postTimeseries(request: APIRequestContext, limit = TIMESERIES_LIMIT) {
  return request.post(`${API}/players/${PLAYER_SLUG}/engagement/timeseries`, {
    headers: JSON_HEADERS,
    data: { limit },
  })
}

async function recentMatchIds(request: APIRequestContext): Promise<string[]> {
  const resp = await postTimeseries(request)
  expect(resp.status()).toBe(200)
  const body = (await resp.json()) as EngagementTimeseries
  expect(body.granularity).toBe('match')
  return body.points.map((p) => p.match_id).filter((id) => id !== '')
}

test.describe('Engagement - Phase 4 endpoints', () => {
  test.beforeEach(async () => {
    await skipIfNoDemoData()
  })

  test('POST /engagement/timeseries renvoie 200 avec des points pace_*', async ({
    request,
  }) => {
    const resp = await postTimeseries(request)
    expect(resp.status()).toBe(200)
    const body = (await resp.json()) as EngagementTimeseries
    expect(body.granularity).toBe('match')
    expect(Array.isArray(body.points)).toBe(true)
    expect(body.points.length).toBeGreaterThan(0)
    expect(body.total_matches).toBeGreaterThan(0)

    const first = body.points[0]
    expect(first.match_id).toBeTruthy()
    expect(typeof first.pace_joueur).toBe('number')
    expect(typeof first.pace_team).toBe('number')
    expect(typeof first.pace_attendu).toBe('number')
    expect(typeof first.pace_lobby).toBe('number')
  })

  test('GET /matches/{id}/engagement renvoie 200 avec courbe non vide', async ({
    request,
  }) => {
    const matchId = (await recentMatchIds(request))[0]
    expect(matchId).toBeTruthy()

    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/matches/${matchId}/engagement`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data.engagement_curve)).toBe(true)
    expect(data.engagement_curve.length).toBeGreaterThan(0)

    const point = data.engagement_curve[0]
    expect(typeof point.time_ms).toBe('number')
    expect(typeof point.pace_joueur).toBe('number')
    expect(typeof point.pace_team).toBe('number')
    expect(typeof point.pace_attendu).toBe('number')
    expect(typeof point.pace_lobby).toBe('number')
  })

  test('GET /pages/squad/v2/engagement renvoie 200 avec labels + means', async ({
    request,
  }) => {
    // Le front passe `match_ids` explicitement : sans eux, le serveur derive les matchs
    // d'une serie parfois agregee (match_id vide) et renvoie une session vide.
    const matchIds = (await recentMatchIds(request)).slice(-5)
    expect(matchIds.length).toBeGreaterThan(0)

    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/pages/squad/v2/engagement?match_ids=${matchIds.join(',')}`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data.labels)).toBe(true)
    expect(data.labels.length).toBe(matchIds.length)
    expect(Array.isArray(data.lobby_per_player)).toBe(true)
    expect(Array.isArray(data.team_expected)).toBe(true)
    expect(Array.isArray(data.team_observed)).toBe(true)
    expect(Array.isArray(data.players)).toBe(true)
  })

  test('GET /engagement_profile renvoie 200 (array, possiblement vide)', async ({
    request,
  }) => {
    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/engagement_profile`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data)).toBe(true)
  })

  test("pas d'erreur 500 sur les endpoints engagement", async ({ request }) => {
    const base = `${API}/players/${PLAYER_SLUG}`
    const responses = [
      await postTimeseries(request, 10),
      await request.get(`${base}/engagement_profile`),
      await request.get(`${base}/pages/squad/v2/engagement`),
    ]
    for (const resp of responses) {
      expect(resp.status(), `${resp.url()} should return 200`).toBe(200)
    }
  })

  test("la courbe d'engagement contient un signal non-trivial (pace_joueur ou pace_team > 0 quelque part)", async ({
    request,
  }) => {
    const matchId = (await recentMatchIds(request))[0]
    expect(matchId).toBeTruthy()

    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/matches/${matchId}/engagement`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    const curve = data.engagement_curve
    const hasSignal = curve.some(
      (p: { pace_joueur: number; pace_team: number }) =>
        p.pace_joueur > 0 || p.pace_team > 0,
    )
    expect(hasSignal, 'La courbe doit contenir au moins un point non nul').toBe(
      true,
    )
  })
})
