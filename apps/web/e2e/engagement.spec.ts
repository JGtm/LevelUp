/**
 * E2E - Engagement (Phase 4 plan engagement).
 *
 * Prerequis : `make dev` (API :8000 + Vite :5173) ; backfill engagement-scores
 * applique au prealable via `levelup backfill --all --engagement-scores`.
 *
 * Couverture :
 *   1. GET /matches/{match_id}/engagement (Mock 10 Match View team tab)
 *   2. GET /engagement/timeseries        (Mock 11 Timeseries intensity tab)
 *   3. GET /pages/squad/v2/engagement    (Mock 15 v2 Squad page)
 *   4. GET /engagement_profile           (Settings + Squad overlay)
 *
 * Configuration (revue 2026-04-29 B5) :
 *   - E2E_API_URL : URL de l'API (defaut http://localhost:8000/api/v1)
 *   - E2E_PLAYER_SLUG : joueur cible avec donnees engagement (defaut JGtm)
 *   - E2E_DEMO_MODE=1 : skip ces tests (mode demo n'a pas les fixtures)
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

const API = process.env.E2E_API_URL ?? 'http://localhost:8000/api/v1'
const PLAYER_SLUG = process.env.E2E_PLAYER_SLUG ?? 'JGtm'
const SKIP_DEMO = process.env.E2E_DEMO_MODE === '1'

test.describe('Engagement - Phase 4 endpoints', () => {
  test.skip(SKIP_DEMO, 'Mode demo : fixtures engagement absentes (revue B5)')

  test('GET /engagement/timeseries renvoie 200 avec items pace_*', async ({
    request,
  }) => {
    await skipIfNoDemoData()
    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/engagement/timeseries?limit=20`,
    )
    expect(resp.status()).toBe(200)
    const items = await resp.json()
    expect(Array.isArray(items)).toBe(true)
    expect(items.length).toBeGreaterThan(0)

    const first = items[0]
    expect(first.match_id).toBeDefined()
    expect(typeof first.pace_joueur).toBe('number')
    expect(typeof first.pace_team).toBe('number')
    expect(typeof first.pace_attendu).toBe('number')
    expect(typeof first.pace_lobby).toBe('number')
  })

  test('GET /matches/{id}/engagement renvoie 200 avec courbe non vide', async ({
    request,
  }) => {
    await skipIfNoDemoData()
    // 1. Pick un match_id depuis la timeseries
    const tsResp = await request.get(
      `${API}/players/${PLAYER_SLUG}/engagement/timeseries?limit=5`,
    )
    expect(tsResp.status()).toBe(200)
    const items = await tsResp.json()
    const matchId = items?.[0]?.match_id
    expect(matchId).toBeTruthy()

    // 2. Charger l'engagement live pour ce match
    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/matches/${matchId}/engagement`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data.engagement_curve)).toBe(true)
    expect(data.engagement_curve.length).toBeGreaterThan(0)
    expect(typeof data.confidence).toBe('string')

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
    await skipIfNoDemoData()
    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/pages/squad/v2/engagement`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data.labels)).toBe(true)
    expect(Array.isArray(data.lobby_per_player)).toBe(true)
    expect(Array.isArray(data.team_expected)).toBe(true)
    expect(Array.isArray(data.team_observed)).toBe(true)
    expect(Array.isArray(data.players)).toBe(true)
  })

  test('GET /engagement_profile renvoie 200 (array, possiblement vide)', async ({
    request,
  }) => {
    await skipIfNoDemoData()
    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/engagement_profile`,
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(Array.isArray(data)).toBe(true)
  })

  test("pas d'erreur 500 sur les endpoints engagement", async ({ request }) => {
    const endpoints = [
      `${API}/players/${PLAYER_SLUG}/engagement/timeseries?limit=10`,
      `${API}/players/${PLAYER_SLUG}/engagement_profile`,
      `${API}/players/${PLAYER_SLUG}/pages/squad/v2/engagement`,
    ]
    for (const url of endpoints) {
      const resp = await request.get(url)
      expect(resp.status(), `${url} should not 500`).not.toBe(500)
    }
  })

  test("la courbe d'engagement contient un signal non-trivial (pace_joueur ou pace_team > 0 quelque part)", async ({
    request,
  }) => {
    await skipIfNoDemoData()
    const tsResp = await request.get(
      `${API}/players/${PLAYER_SLUG}/engagement/timeseries?limit=5`,
    )
    const items = await tsResp.json()
    const matchId = items?.[0]?.match_id
    expect(matchId).toBeTruthy()

    const resp = await request.get(
      `${API}/players/${PLAYER_SLUG}/matches/${matchId}/engagement`,
    )
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
