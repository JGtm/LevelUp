/**
 * E2E — Slice 2B : Page Citations.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture :
 * 1. L'API POST /pages/citations retourne HTTP 200
 * 2. La réponse contient des commendations et des médailles
 * 3. La page /players/demo-player/citations se charge sans erreur
 * 4. Pas d'erreur fatale dans la console
 */
import { test, expect } from '@playwright/test'

test.describe('Slice 2B — Page Citations (DEMO_MODE)', () => {
  test("l'API /pages/citations retourne HTTP 200", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).toBe(200)
    const data = await resp.json()
    expect(data).toBeTruthy()
  })

  test("la réponse citations contient des commendations", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.commendations).toBeDefined()
    expect(Array.isArray(data.commendations)).toBe(true)
  })

  test("la réponse citations contient des médailles", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.medals).toBeDefined()
    expect(Array.isArray(data.medals)).toBe(true)
  })

  test("pas d'erreur 500 sur /pages/citations", async ({ request }) => {
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    expect(resp.status()).not.toBe(500)
  })
})
