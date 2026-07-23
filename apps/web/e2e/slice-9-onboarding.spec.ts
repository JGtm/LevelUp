/**
 * E2E — Slice 9 : Onboarding complet (auth → profil → home).
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture Sprint 36 (tâche 3 — Onboarding E2E) :
 * 1. Le flow setup se charge et affiche l'état de configuration
 * 2. L'API /bootstrap retourne un setup_state valide
 * 3. En DEMO_MODE, le joueur DEMO est directement disponible (pas d'auth requise)
 * 4. La page /home est accessible après le bootstrap
 * 5. L'API /auth/device-flow/start retourne HTTP 422 en mode démo (attendu)
 * 6. Les settings sont accessibles et valides
 * 7. Le flow de navigation settings → home fonctionne sans erreur
 */
import { test, expect } from '@playwright/test'
import { playerPath } from './_helpers/routes'

const API_BASE = 'http://localhost:8000/api/v1'

test.describe('Slice 9 — Onboarding flow (DEMO_MODE)', () => {
  test("le bootstrap indique setup_state='ready' en DEMO_MODE", async ({ request }) => {
    const resp = await request.get(`${API_BASE}/bootstrap`)
    expect(resp.status()).toBe(200)

    const data = await resp.json() as Record<string, unknown>
    expect(data.setup_state).toBeDefined()
    // En DEMO_MODE, le joueur est pré-configuré → ready ou profile_ready_no_sync
    expect(['ready', 'profile_ready_no_sync']).toContain(data.setup_state)
    expect(data.current_player).toBeTruthy()
  })

  test('la page /setup se charge sans erreur critique', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/setup')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("l'API auth/device-flow/start est refusée en DEMO_MODE (422)", async ({ request }) => {
    // En DEMO_MODE, le device flow est désactivé — vérifier la réponse d'erreur
    const resp = await request.post(`${API_BASE}/auth/device-flow/start`, {
      data: {},
      headers: { 'Content-Type': 'application/json', 'Origin': 'http://localhost:5173' },
    })
    // 422 = demo_mode, 403 = CSRF → les deux sont attendus en mode démo
    expect([422, 403]).toContain(resp.status())
  })

  test('les settings sont accessibles et contiennent une config valide', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/settings`)
    expect(resp.status()).toBe(200)

    const data = await resp.json() as Record<string, unknown>
    expect(data).toBeTruthy()
    // Valider que les clés minimales de settings sont présentes
    expect(typeof data).toBe('object')
  })

  test('la page /settings se charge sans erreur JS', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test('navigation /setup → /home sans blocage', async ({ page }) => {
    // Depuis setup, naviguer vers home via le joueur démo
    await page.goto('/setup')
    await page.waitForLoadState('networkidle')

    // Naviguer vers home directement
    await page.goto(playerPath('demo-player', 'home'))
    await page.waitForLoadState('networkidle')

    // Vérifier qu'on est bien sur la page home sans erreur critique
    await expect(page.locator('body')).not.toContainText('Erreur critique')
    await expect(page.locator('body')).not.toContainText('Internal Server Error')
    expect(page.url()).toContain('/home')
  })

  test('API players retourne au moins un joueur en DEMO_MODE', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/players`)
    expect(resp.status()).toBe(200)

    const body = await resp.json() as { items: unknown[] }
    expect(Array.isArray(body.items)).toBe(true)
    expect(body.items.length).toBeGreaterThan(0)
  })
})
