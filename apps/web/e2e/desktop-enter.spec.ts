/**
 * E2E — Entrée de la coquille desktop (ADR 0045 D-3 et D-8, lot L13.4).
 *
 * Le serveur desktop répond 401 `desktop_unauthorized` à toute requête sans le cookie posé par
 * `GET /desktop/enter?secret=` (cookie `levelup_session_<port>`). L'interface doit alors :
 *   - sans cookie : afficher l'écran « accès expiré » et ne PAS rejouer le bootstrap (un rejeu ne
 *     change rien) ni afficher « API injoignable » ;
 *   - avec le cookie : charger normalement, et ne PAS proposer la bannière de consentement à la
 *     mise en commun des jetons (accordée d'office à l'utilisateur local, capacité
 *     `can_pool_tokens` fausse).
 *
 * Le serveur est simulé (aucune API réelle) : le contrôle de cookie du serveur Go est prouvé par
 * ses propres tests (internal/api/desktop_entry_cgo_test.go, cmd/server TestE2ELanceurDesktop) ;
 * ici le mock en applique la règle (cookie présent => 200, absent => 401) pour exercer l'interface.
 *
 * Prérequis : Vite sur :5173 (`make dev`, ou `npx vite` depuis apps/web) ; l'API n'est pas utilisée.
 */
import { test, expect, type BrowserContext, type Page } from '@playwright/test'

const API = '/api/v1'
const PORT = 49555
const COOKIE_NAME = `levelup_session_${PORT}`

const UNAUTHORIZED = {
  code: 'desktop_unauthorized',
  message: 'entrée de la coquille requise',
  retryable: false,
}

/** Bootstrap du profil desktop : aucun compte, mise en commun non proposée, mise à jour offerte. */
const DESKTOP_BOOTSTRAP = {
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
    // Profil desktop (ADR 0044) : ni comptes, ni mise en commun proposée, mise à jour offerte.
    can_use_accounts: false, can_sign_in_with_xbox: false, can_use_invitations: false, can_use_groups: false,
    can_pool_tokens: false, can_use_admin: false, can_use_restic: false, can_use_discord: false,
    can_use_outbound_network: true, can_update_app: true, can_offer_windows_download: false,
    can_edit_settings: true, can_upload_media: true, deployment: 'desktop',
  },
  settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
  auth_mode: 'none',
  registration_mode: 'invite',
  current_username: null,
  is_admin: false,
  first_launch: false,
  // Décision enregistrée d'office par le serveur : « undecided » ferait apparaître la bannière si
  // la capacité ne la masquait pas.
  pool_sharing: 'undecided',
}

function json(status: number, body: unknown) {
  return { status, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}

/** Le serveur simulé : 401 sans le cookie de la coquille, 200 avec. Rend le compteur de bootstraps. */
async function mockDesktopServer(page: Page, context: BrowserContext): Promise<{ bootstrapCalls: () => number }> {
  let bootstrapCalls = 0
  const hasCookie = async () => (await context.cookies()).some((c) => c.name === COOKIE_NAME)
  // Catch-all d'abord (priorité la plus basse), routes précises ensuite.
  await page.route(`**${API}/**`, async (route) => {
    await route.fulfill((await hasCookie()) ? json(200, {}) : json(401, UNAUTHORIZED))
  })
  await page.route(`**${API}/bootstrap`, async (route) => {
    bootstrapCalls += 1
    await route.fulfill((await hasCookie()) ? json(200, DESKTOP_BOOTSTRAP) : json(401, UNAUTHORIZED))
  })
  return { bootstrapCalls: () => bootstrapCalls }
}

test.describe('Entrée de la coquille desktop', () => {
  test("sans cookie : l'écran « accès expiré », sans rejeu ni « API injoignable »", async ({ page, context }) => {
    const server = await mockDesktopServer(page, context)

    await page.goto('/')

    await expect(page.getByText("L'accès à LevelUp a expiré.")).toBeVisible()
    await expect(page.getByText('Fermez cette fenêtre, puis rouvrez LevelUp.')).toBeVisible()
    await expect(page.getByText("Impossible de contacter l'API.")).toHaveCount(0)
    // Un rejeu (≈ 0,5 s puis 1 s) aurait déjà eu lieu : la requête n'est faite qu'une fois.
    await page.waitForTimeout(3000)
    expect(server.bootstrapCalls()).toBe(1)
  })

  test("avec le cookie : l'interface charge et ne propose pas la mise en commun des jetons", async ({ page, context, baseURL }) => {
    const server = await mockDesktopServer(page, context)
    await context.addCookies([{ name: COOKIE_NAME, value: 'x.y', url: baseURL ?? 'http://localhost:5173' }])

    await page.goto('/')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText("L'accès à LevelUp a expiré.")).toHaveCount(0)
    expect(server.bootstrapCalls()).toBeGreaterThanOrEqual(1)
    // La bannière de consentement (ADR 0038) est masquée par la capacité ; accordée d'office (ADR 0045 D-8).
    await expect(page.getByRole('region', { name: 'Mise en commun des jetons' })).toHaveCount(0)
  })
})
