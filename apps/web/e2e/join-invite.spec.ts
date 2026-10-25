/**
 * E2E — Lien d'invitation (/join?invite=CODE) ouvert par un visiteur SANS compte.
 *
 * Régression : la racine renvoyait tout visiteur anonyme vers /login, y compris depuis
 * /join ; le code d'invitation était perdu, l'invité se connectait sans lui et l'instance
 * fermée le refusait (instance_locked). La page doit rester affichée et le bouton Xbox
 * doit porter le code jusqu'au serveur (/auth/xbox/login?invite=CODE).
 *
 * Le bootstrap est simulé (mode xbox, personne de connecté) et la redirection vers
 * Microsoft est interceptée : aucun compte ni code réel n'est consommé.
 */
import { test, expect } from '@playwright/test'

const API = '/api/v1'
const CODE = 'CODE1234'

const ANON_BOOTSTRAP = {
  setup_required: false,
  auth_state: 'missing',
  setup_state: 'ready',
  current_player: null,
  available_players: [],
  current_title_slug: 'halo_infinite',
  available_titles: [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', capabilities: ['matchmaking'], is_default: true },
  ],
  locale: 'fr',
  hints_visible_default: true,
  feature_flags: { v7_enabled: true, media_enabled: true, demo_mode: false, discord_configured: false, tailscale_enabled: false },
  capabilities: {
    can_read_local_data: false, can_run_sync: false, can_use_live_halo: false,
    can_manage_settings: false, can_reset_media_index: false, can_view_media: false,
    can_self_provision: false, can_start_initial_sync: false, can_manage_instance: false,
  },
  settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
  auth_mode: 'xbox',
  registration_mode: 'closed',
  current_username: null,
  is_admin: false,
  first_launch: false,
}

function ok(json: unknown) {
  return { status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(json) }
}

test.describe("Invitation — visiteur sans compte", () => {
  test.beforeEach(async ({ page }) => {
    // Catch-all d'abord (priorité la plus basse), routes précises ensuite.
    await page.route(`**${API}/**`, (r) => r.fulfill(ok({})))
    await page.route(`**${API}/bootstrap`, (r) => r.fulfill(ok(ANON_BOOTSTRAP)))
  })

  test("la page d'invitation reste affichée et transmet le code au login Xbox", async ({ page }) => {
    let loginUrl: string | null = null
    await page.route('**/auth/xbox/login**', (r) => {
      loginUrl = r.request().url()
      return r.fulfill({ status: 200, headers: { 'Content-Type': 'text/html' }, body: '<html></html>' })
    })

    await page.goto(`/join?invite=${CODE}`)
    await page.waitForLoadState('networkidle')

    expect(new URL(page.url()).pathname).toBe('/join')
    const button = page.getByRole('button', { name: 'Continuer avec Xbox' })
    await expect(button).toBeVisible()

    await button.click()
    await expect.poll(() => loginUrl).not.toBeNull()
    expect(new URL(loginUrl ?? '').searchParams.get('invite')).toBe(CODE)
  })

  test('une page applicative renvoie toujours le visiteur anonyme vers /login', async ({ page }) => {
    await page.goto('/settings')
    await page.waitForURL('**/login')
    expect(new URL(page.url()).pathname).toBe('/login')
  })
})
