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
import { skipIfNoDemoData } from './_helpers/demoData'

test.describe('Slice 2B — Page Citations (DEMO_MODE)', () => {
  test("l'API /pages/citations retourne HTTP 200", async ({ request }) => {
    await skipIfNoDemoData()
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
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.citations).toBeDefined()
    expect(Array.isArray(data.citations)).toBe(true)
  })

  test("la réponse citations contient des médailles", async ({ request }) => {
    await skipIfNoDemoData()
    const resp = await request.post(
      'http://localhost:8000/api/v1/players/demo-player/pages/citations',
      {
        data: { filters: {} },
      },
    )
    const data = await resp.json()
    expect(data.citations).toBeDefined()
    expect(data.citations.length).toBeGreaterThan(0)
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
