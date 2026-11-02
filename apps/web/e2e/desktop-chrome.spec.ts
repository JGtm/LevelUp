/**
 * E2E — Interface de bureau (lot L18) : sous la capacité `can_use_compact_chrome` de /bootstrap,
 * l'application montre une barre latérale à la place de la barre haute, aucun pied de page, un
 * accueil sans image de couverture et une fenêtre « À propos » qui porte la version, le dossier
 * des données et la recherche de mise à jour.
 *
 * Le serveur est simulé (bootstrap du profil desktop) ; la coquille aussi : la page n'appelle que
 * `http://levelup-shell.localhost/check-update`, que le test intercepte. Le comportement réel de la
 * coquille (boîte native, retour sur la page) est vérifié sur l'application construite (DevTools
 * WebView2), pas ici.
 *
 * Prérequis : Vite (`npx vite` depuis apps/web, ou `make dev`) ; E2E_BASE_URL si ce n'est pas :5173.
 */
import { test, expect, type Page } from '@playwright/test'

const API = '/api/v1'
/** Le pied de page du site (balise footer étiquetée) : un `footer` dans `main` n'a pas le rôle contentinfo. */
const FOOTER = 'footer[aria-label="Pied de page"]'
const DATA_ROOT = 'C:\\Users\\Test\\AppData\\Local\\LevelUp'

const PLAYER = {
  player_slug: 'jgtm',
  gamertag: 'JGtm',
  xuid: 'xuid(1)',
  waypoint_player: 'JGtm',
  is_demo: false,
  sync_enabled: true,
}

function bootstrap(compact: boolean) {
  return {
    setup_required: false,
    auth_state: 'ready',
    setup_state: 'ready',
    current_player: PLAYER,
    available_players: [PLAYER],
    writable_player_slugs: ['jgtm'],
    current_title_slug: 'halo_infinite',
    available_titles: [
      { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', capabilities: ['matchmaking'], is_default: true },
    ],
    locale: 'fr',
    hints_visible_default: true,
    feature_flags: { v7_enabled: true, media_enabled: true, demo_mode: false, discord_configured: false, tailscale_enabled: false },
    capabilities: {
      can_read_local_data: true, can_run_sync: true, can_use_live_halo: true,
      can_manage_settings: true, can_reset_media_index: true, can_view_media: true,
      can_self_provision: true, can_start_initial_sync: true, can_manage_instance: true,
      can_use_accounts: false, can_sign_in_with_xbox: false, can_use_invitations: false,
      can_use_groups: !compact, can_pool_tokens: !compact, can_use_admin: !compact, can_use_restic: !compact,
      can_use_discord: !compact, can_use_outbound_network: true, can_update_app: compact,
      can_offer_windows_download: !compact, can_edit_settings: true, can_upload_media: true,
      can_use_compact_chrome: compact, deployment: compact ? 'desktop' : 'hosted',
    },
    ...(compact ? { app_info: { version: '8.1.0', data_root: DATA_ROOT } } : {}),
    settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
    auth_mode: 'none',
    registration_mode: 'invite',
    current_username: null,
    is_admin: false,
    first_launch: false,
    pool_sharing: 'unavailable',
  }
}

function json(body: unknown) {
  return { status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}

async function mockServer(page: Page, compact: boolean) {
  await page.route(`**${API}/**`, (route) => route.fulfill(json({})))
  await page.route(`**${API}/bootstrap`, (route) => route.fulfill(json(bootstrap(compact))))
  // /help : page statique, qui ne dépend d'aucune donnée de l'API simulée.
  await page.goto('/help')
  await page.waitForLoadState('networkidle')
}

test.describe('Interface de bureau', () => {
  test('capacité vraie : barre latérale, ni barre haute ni pied de page', async ({ page }) => {
    await mockServer(page, true)

    const sidebar = page.getByRole('complementary', { name: 'Navigation principale' })
    await expect(sidebar).toBeVisible()
    await expect(sidebar.getByRole('link', { name: 'Accueil', exact: true })).toBeVisible()
    await expect(sidebar.getByRole('button', { name: 'À propos' })).toBeVisible()
    await expect(sidebar.getByTestId('sidebar-version')).toHaveText('Version 8.1.0')
    // Aucun pied de page de site, aucune déconnexion, aucun lien « Application Windows ».
    await expect(page.locator(FOOTER)).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Application Windows' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: /Se déconnecter/ })).toHaveCount(0)
  })

  test('« À propos » : version, dossier des données copiable, Échap ferme', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])
    await mockServer(page, true)

    await page.getByRole('button', { name: 'À propos' }).click()
    const dialog = page.getByRole('dialog', { name: 'LevelUp' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByTestId('about-version')).toContainText('8.1.0')
    await expect(dialog.getByTestId('about-data-root')).toHaveText(DATA_ROOT)
    await expect(dialog.getByTestId('microsoft-notice')).toContainText('not endorsed by or affiliated with Microsoft')

    await dialog.getByRole('button', { name: 'Copier' }).click()
    await expect(dialog.getByRole('button', { name: 'Copié' })).toBeVisible()
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(DATA_ROOT)

    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
  })

  test('« Rechercher une mise à jour » appelle la coquille et la page reste en place', async ({ page }) => {
    let shellCalls = 0
    await page.route('http://levelup-shell.localhost/**', (route) => {
      shellCalls += 1
      return route.fulfill({ status: 204, headers: { 'Access-Control-Allow-Origin': '*' } })
    })
    await mockServer(page, true)
    const before = page.url()

    await page.getByRole('button', { name: 'À propos' }).click()
    await page.getByRole('button', { name: 'Rechercher une mise à jour' }).click()

    await expect(page.getByRole('status')).toContainText('Recherche lancée')
    expect(shellCalls).toBe(1)
    expect(page.url()).toBe(before)
    await expect(page.getByRole('dialog', { name: 'LevelUp' })).toBeVisible()
  })

  test('capacité fausse (le site) : barre haute et pied de page, aucune barre latérale', async ({ page }) => {
    await mockServer(page, false)

    await expect(page.getByRole('complementary', { name: 'Navigation principale' })).toHaveCount(0)
    await expect(page.getByRole('navigation', { name: 'Navigation principale' })).toBeVisible()
    await expect(page.locator(FOOTER)).toBeVisible()
    await expect(page.getByRole('button', { name: 'À propos' })).toHaveCount(0)
  })
})
