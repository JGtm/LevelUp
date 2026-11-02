/**
 * E2E — Barre latérale repliable de l'application de bureau (L21.0), capacités de la communauté
 * (L21.0b) et message sans encodeur H.264 (L21.0d) : sous la capacité `can_use_compact_chrome` de /bootstrap, la barre se replie en icônes (bouton
 * du pied, Ctrl+B, fenêtre étroite), mémorise le choix manuel, ouvre un menu volant des onglets ; sans
 * `can_compare_accounts`, « Classements » disparaît de la Communauté et son adresse renvoie à « Relations ».
 *
 * Le serveur est simulé (bootstrap du profil desktop), comme dans desktop-chrome.spec.ts. La variable
 * L21_WITNESS_DIR, quand elle est posée, reçoit les captures témoins des deux états de la barre.
 *
 * Prérequis : Vite (`npx vite`) ou `vite preview` sur le build ; E2E_BASE_URL si ce n'est pas :5173.
 */
import { test, expect, type Page } from '@playwright/test'

const API = '/api/v1'
const WITNESS_DIR = process.env.L21_WITNESS_DIR

const PLAYER = {
  player_slug: 'jgtm',
  gamertag: 'JGtm',
  xuid: 'xuid(1)',
  waypoint_player: 'JGtm',
  is_demo: false,
  sync_enabled: true,
}

function bootstrap(desktop: boolean, extra: Record<string, unknown> = {}) {
  return {
    setup_required: false,
    auth_state: 'ready',
    setup_state: 'ready',
    current_player: PLAYER,
    available_players: [PLAYER],
    writable_player_slugs: ['jgtm'],
    current_title_slug: 'halo_infinite',
    available_titles: [
      {
        slug: 'halo_infinite',
        name: 'Halo Infinite',
        status: 'active',
        capabilities: ['matchmaking', 'world.leaderboard', 'career', 'media', 'lusr', 'replay'],
        is_default: true,
      },
      { slug: 'halo_5', name: 'Halo 5', status: 'active', capabilities: ['matchmaking'], is_default: false },
    ],
    locale: 'fr',
    hints_visible_default: true,
    feature_flags: { v7_enabled: true, media_enabled: true, demo_mode: false, discord_configured: false, tailscale_enabled: false },
    capabilities: {
      can_read_local_data: true, can_run_sync: true, can_use_live_halo: true,
      can_manage_settings: true, can_reset_media_index: true, can_view_media: true,
      can_self_provision: true, can_start_initial_sync: true, can_manage_instance: true,
      can_use_accounts: false, can_sign_in_with_xbox: false, can_use_invitations: false,
      can_use_groups: !desktop, can_pool_tokens: !desktop, can_use_admin: !desktop, can_use_restic: !desktop,
      can_use_discord: !desktop, can_use_outbound_network: true, can_update_app: desktop,
      can_offer_windows_download: !desktop, can_edit_settings: true, can_upload_media: true,
      can_use_compact_chrome: desktop, can_login_via_browser: desktop, can_compare_accounts: !desktop,
      deployment: desktop ? 'desktop' : 'hosted',
    },
    ...(desktop ? { app_info: { version: '8.1.0', data_root: 'C:\\Users\\Test\\AppData\\Local\\LevelUp' } } : {}),
    settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
    auth_mode: 'none',
    registration_mode: 'invite',
    current_username: null,
    is_admin: false,
    first_launch: false,
    pool_sharing: 'unavailable',
    ...extra,
  }
}

function json(body: unknown) {
  return { status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}

async function open(page: Page, desktop: boolean, path = '/help', extra: Record<string, unknown> = {}) {
  await page.route(`**${API}/**`, (route) => route.fulfill(json({})))
  await page.route(`**${API}/bootstrap`, (route) => route.fulfill(json(bootstrap(desktop, extra))))
  await page.goto(path)
  await page.waitForLoadState('networkidle')
}

async function witness(page: Page, name: string) {
  if (WITNESS_DIR) await page.screenshot({ path: `${WITNESS_DIR}/${name}.png` })
}

const sidebar = (page: Page) => page.locator('aside[data-sidebar]')

test.describe('Barre latérale repliable', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('bouton, mémorisation, menu volant et Ctrl+B', async ({ page }) => {
    await open(page, true)
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'expanded')
    expect((await sidebar(page).boundingBox())?.width).toBeCloseTo(240, 0)
    await witness(page, 'sidebar_1_deployee')

    await page.getByRole('button', { name: 'Replier la barre latérale' }).click()
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'collapsed')
    expect((await sidebar(page).boundingBox())?.width).toBeCloseTo(56, 0)
    await witness(page, 'sidebar_2_repliee')

    // Menu volant : le survol d'une section à onglets les liste.
    await sidebar(page).getByRole('button', { name: 'Solo' }).hover()
    const menu = page.getByRole('menu', { name: 'Solo' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: 'Synthèse' })).toBeVisible()
    await witness(page, 'sidebar_3_menu_volant')

    // Infobulle d'une section sans onglets.
    await sidebar(page).getByRole('link', { name: 'Explorer' }).hover()
    await expect(page.getByRole('tooltip')).toHaveText('Explorer')

    // Le choix manuel survit au rechargement.
    await page.reload()
    await page.waitForLoadState('networkidle')
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'collapsed')

    await page.keyboard.press('Control+b')
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'expanded')
    await page.keyboard.press('Control+b')
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'collapsed')
  })

  test('fenêtre étroite : repli automatique, redéploiement quand elle s’élargit', async ({ page }) => {
    await open(page, true)
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'expanded')
    await page.setViewportSize({ width: 1050, height: 800 })
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'collapsed')
    await page.setViewportSize({ width: 1300, height: 800 })
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'expanded')
  })

  test('un choix manuel n’est pas écrasé par la largeur de la fenêtre', async ({ page }) => {
    await open(page, true)
    await page.getByRole('button', { name: 'Replier la barre latérale' }).click()
    await page.setViewportSize({ width: 1300, height: 800 })
    await page.setViewportSize({ width: 1440, height: 900 })
    await expect(sidebar(page)).toHaveAttribute('data-sidebar', 'collapsed')
  })

  test('le site hébergé garde sa barre haute, sans barre latérale', async ({ page }) => {
    await open(page, false)
    await expect(sidebar(page)).toHaveCount(0)
    await expect(page.getByRole('navigation', { name: 'Navigation principale' })).toBeVisible()
  })
})

test.describe('Communauté sans comparaison de comptes', () => {
  test.use({ viewport: { width: 1440, height: 900 } })
  const RELATIONS = '/t/halo_infinite/players/jgtm/community/relations'

  test('bureau : pas d’onglet Classements, et /community renvoie à Relations', async ({ page }) => {
    await open(page, true, RELATIONS)
    await expect(page.getByRole('link', { name: 'Face-à-face' }).first()).toBeVisible()
    await expect(page.getByRole('link', { name: 'Classements' })).toHaveCount(0)

    await page.goto('/t/halo_infinite/players/jgtm/community')
    await page.waitForURL('**/community/relations')
  })

  test('site : Classements reste offert', async ({ page }) => {
    await open(page, false, RELATIONS)
    await expect(page.getByRole('link', { name: 'Classements' }).first()).toBeVisible()
  })
})

test.describe('Médias sans encodeur H.264', () => {
  test.use({ viewport: { width: 1440, height: 900 } })
  const MEDIA = '/t/halo_infinite/players/jgtm/media'

  test('le serveur signale l’absence d’encodeur : la page explique et pointe le Media Feature Pack', async ({ page }) => {
    await open(page, true, MEDIA, { media_h264_missing: true })
    const notice = page.getByTestId('media-no-h264')
    await expect(notice).toBeVisible()
    await expect(notice).toContainText('aucun encodeur vidéo H.264')
    const link = notice.getByRole('link', { name: 'Page du Media Feature Pack' })
    await expect(link).toHaveAttribute('href', /media-feature-pack-list-for-windows-n-editions/)
    await expect(link).toHaveAttribute('target', '_blank')
    await witness(page, 'medias_sans_encodeur')
  })

  test('un encodeur existe : aucun message', async ({ page }) => {
    await open(page, true, MEDIA)
    await expect(page.getByTestId('media-no-h264')).toHaveCount(0)
  })
})
