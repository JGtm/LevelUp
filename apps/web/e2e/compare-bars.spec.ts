/**
 * E2E — Compare : vérification visuelle des barres composites.
 *
 * Teste que les barres linear-gradient sont bien rendues avec le bon style.
 * Lance avec : npx playwright test e2e/compare-bars.spec.ts
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const PLAYER_SLUG = 'demo-player'
const TARGET = 'TestPlayer'
const API = '/api/v1'

// ─── Fixtures alignées sur src/test/handlers.ts ───────────────────────────────

const PLAYER = {
  player_slug: PLAYER_SLUG,
  gamertag: 'DemoPlayer',
  xuid: '0000000000000001',
  waypoint_player: 'DemoPlayer',
  is_demo: true,
}

const BOOTSTRAP = {
  setup_required: false,
  auth_state: 'ready',
  setup_state: 'ready',
  current_player: PLAYER,
  available_players: [PLAYER],
  current_title_slug: 'halo_infinite',
  available_titles: [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', capabilities: ['matchmaking'], is_default: true },
  ],
  locale: 'fr',
  hints_visible_default: true,
  feature_flags: { v7_enabled: true, media_enabled: true, demo_mode: true, discord_configured: false, tailscale_enabled: false },
  capabilities: {
    can_read_local_data: true, can_run_sync: false, can_use_live_halo: false,
    can_manage_settings: true, can_reset_media_index: false, can_view_media: false,
    can_self_provision: false, can_start_initial_sync: false, can_manage_instance: false,
  },
  settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
  auth_mode: 'none',
  registration_mode: 'closed',
  is_admin: false,
  first_launch: false,
}

const PLAYER_A = {
  xuid: '0000000000000001', gamertag: 'DemoPlayer', title_slug: 'halo_infinite', is_local: true,
  matches: 769, win_rate: 0.484, kda: 0.93, kdr: 0.83, kills_per_game: 9.54, deaths_per_game: 11.51,
  assists_per_game: 3.58, accuracy: 0.432, damage_per_game: 2780.38, damage_taken_per_game: 3302.61,
  perfect_kills_per_game: 0.15, max_killing_spree: 16, avg_life_secs: 31, headshot_kills_per_game: 3.21,
  perf_ath: 98, lusr_ath: 1606, career_rank: 179, csr_current: 1400, csr_best: 1500,
  favorite_weapon: { weapon_id: 1, label_fr: 'Sidekick', label_en: 'Sidekick', kills: 2231 },
}

const COMPARE_RESPONSE = {
  player_a: PLAYER_A,
  player_b: { ...PLAYER_A, xuid: '0000000000000002', gamertag: TARGET, kills_per_game: 16.81, kda: 1.82, matches: 21, favorite_weapon: null },
  metrics: [
    { metric: 'win_rate',       label_fr: 'Taux de victoire', value_a: 0.484, value_b: 0.476, delta: -0.008, winner: 'a' },
    { metric: 'kda',            label_fr: 'KDA',              value_a: 0.93,  value_b: 1.82,  delta: 0.89,  winner: 'b' },
    { metric: 'kdr',            label_fr: 'K/D',              value_a: 0.83,  value_b: 1.66,  delta: 0.83,  winner: 'b' },
    { metric: 'kills_per_game', label_fr: 'Frags/partie',     value_a: 9.54,  value_b: 16.81, delta: 7.27,  winner: 'b' },
    { metric: 'deaths_per_game',label_fr: 'Morts/partie',     value_a: 11.51, value_b: 10.14, delta: -1.37, winner: 'b' },
    { metric: 'damage_per_game',label_fr: 'Dégâts/partie',    value_a: 2780,  value_b: 4806,  delta: 2026,  winner: 'b' },
    { metric: 'matches',        label_fr: 'Parties jouées',   value_a: 769,   value_b: 21,    delta: -748,  winner: 'a' },
  ],
  title_slug: 'halo_infinite',
  privacy_warning: null,
  player_b_partial: false,
}

function ok(json: unknown) {
  return { status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(json) }
}

test.describe('Compare — barres composites', () => {
  test.beforeEach(async ({ page }) => {
    // Catch-all first (lowest priority), specific routes last (highest priority)
    await page.route(`**${API}/**`, (r) => r.fulfill(ok([])))

    await page.route(`**${API}/bootstrap`, (r) => r.fulfill(ok(BOOTSTRAP)))
    await page.route(`**${API}/health`, (r) => r.fulfill(ok({ status: 'ok' })))
    await page.route(`**${API}/players`, (r) => r.fulfill(ok({ items: [PLAYER], default_player_slug: PLAYER_SLUG })))
    await page.route(`**${API}/titles/*/field-mappings**`, (r) => r.fulfill(ok({ fields: {} })))
    await page.route(`**${API}/settings`, (r) => r.fulfill(ok({ lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true, theme: 'dark', color_palette: 'default', ui_density: 'comfortable', ally_team_color: null, enemy_team_color: null, hint_compare_auto_open: true })))
    await page.route(`**${API}/directory/gamertags/search*`, (r) => r.fulfill(ok({ query: '', items: [] })))
    await page.route(`**/notifications/unread-count`, (r) => r.fulfill(ok({ count: 0, by_category: {} })))
    await page.route(`**/battlepass`, (r) => r.fulfill(ok({ available: false })))
    await page.route(`**/challenges`, (r) => r.fulfill(ok({ available: false, items: [] })))
    await page.route(`**/pages/compare`, (r) => r.fulfill(ok(COMPARE_RESPONSE)))
  })

  test('les barres composites ont un style linear-gradient', async ({ page }) => {
    await page.goto(playerPath(PLAYER_SLUG, `compare?target=${TARGET}`))
    await page.waitForLoadState('networkidle')

    await expect(page.locator('text=DemoPlayer').first()).toBeVisible({ timeout: 10_000 })

    const barDivs = page.locator('[data-testid="compare-bar-track"]')
    expect(await barDivs.count()).toBeGreaterThan(0)

    const firstBar = barDivs.first()
    const styleAttr = await firstBar.evaluate((el) => el.getAttribute('style') ?? '')
    expect(styleAttr).toContain('linear-gradient')
    await expect(firstBar).toBeVisible()
  })
})
