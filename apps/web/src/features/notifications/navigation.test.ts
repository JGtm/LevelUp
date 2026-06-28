/**
 * Tests régression B1 (revue 2026-04-29) : routes notifications.
 *
 * Avant le fix B1, navigation.ts résolvait 3 catégories vers des routes
 * inexistantes (/defis, /help/changelog, /players/$slug/sync). Ces tests
 * vérifient que :
 *   1. Toutes les catégories de notification résolvent vers une route
 *      valide (préfixe whitelist).
 *   2. Aucune des 3 routes fantômes B1 n'est plus retournée.
 *
 * Tests symétriques côté Go : apps/go-api/internal/api/post_sync_deltas_test.go
 *   - TestEmitPostSyncDeltas_AllTargetRoutesValid
 *   - TestEmitPostSyncDeltas_NoFantomRoutes
 */
import { describe, it, expect } from 'vitest'

import { resolveTarget } from './navigation'
import type { Notification, NotificationCategory } from './types'
import { ALL_CATEGORIES } from './types'

const PLAYER_SLUG = 'test-player'

// Whitelist des préfixes de routes acceptés.
// À synchroniser avec routeTree.gen.ts si la nav front évolue.
const VALID_TOP_ROUTES = ['/changelog', '/settings'] as const

const VALID_PLAYER_SUBPATHS = [
  '/home',
  '/synthesis',
  '/objectifs',
  '/ascension',
  '/ascension/coaching',
  '/palmares',
  '/palmares/season-pass',
  '/palmares/prestige',
  '/palmares/relations',
  '/palmares/compare',
  '/community',
  '/community/relations',
  '/community/prestige',
  '/community/compare',
  '/career',
  '/career/season-pass',
  '/match',
  '/matches',
  '/media',
  '/stats',
  '/stats/query',
  '/explorer',
  '/sessions',
  '/squad',
  '/timeseries',
  '/teammates',
  '/compare',
  '/notifications',
] as const

const FANTOM_ROUTES_B1 = [
  `/players/${PLAYER_SLUG}/defis`,
  '/help/changelog',
  `/players/${PLAYER_SLUG}/sync`,
] as const

function targetRouteIsValid(to: string): boolean {
  if ((VALID_TOP_ROUTES as readonly string[]).includes(to)) return true
  const prefix = '/players/'
  if (!to.startsWith(prefix)) return false
  const rest = to.slice(prefix.length)
  const slash = rest.indexOf('/')
  if (slash < 0) return true // /players/{slug} sans sous-chemin
  const subPath = rest.slice(slash)
  return (VALID_PLAYER_SUBPATHS as readonly string[]).includes(subPath)
}

function makeNotif(category: NotificationCategory): Notification {
  return {
    id: 1,
    category,
    severity: 'info',
    title_key: `notif.${category}.title`,
    source: 'test',
    created_at: '2026-04-29T00:00:00Z',
    read_at: null,
    params: { id: 42, job_id: 'job-1', metric: 'kd_ratio', match_id: 'm-1' },
  }
}

describe('navigation.ts - regression B1 routes notifications', () => {
  it('toutes les categories resolvent vers une route valide (whitelist)', () => {
    for (const category of ALL_CATEGORIES) {
      const target = resolveTarget(makeNotif(category), PLAYER_SLUG)
      // null = pas de route ciblee : OK pour les categories sans drill-down
      if (target === null) continue
      expect(
        targetRouteIsValid(target.to),
        `category=${category} renvoie route invalide "${target.to}"`,
      ).toBe(true)
    }
  })

  it('aucune des 3 routes fantomes B1 n\'est plus retournee', () => {
    for (const category of ALL_CATEGORIES) {
      const target = resolveTarget(makeNotif(category), PLAYER_SLUG)
      if (target === null) continue
      for (const fantom of FANTOM_ROUTES_B1) {
        expect(
          target.to,
          `category=${category} renvoie route fantome "${fantom}" (regression B1)`,
        ).not.toBe(fantom)
      }
    }
  })

  it('challenge_added cible /ascension (tab profil & objectifs)', () => {
    const target = resolveTarget(makeNotif('challenge_added'), PLAYER_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`/players/${PLAYER_SLUG}/ascension`)
  })

  it('app_release cible /changelog (pas /help/changelog)', () => {
    const target = resolveTarget(makeNotif('app_release'), PLAYER_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe('/changelog')
  })

  it('sync_error cible /settings (pas /players/$slug/sync)', () => {
    const target = resolveTarget(makeNotif('sync_error'), PLAYER_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe('/settings')
    expect(target!.search).toMatchObject({ jobId: 'job-1' })
  })

  it('target_route backend prend le pas sur le fallback', () => {
    const notif = makeNotif('app_release')
    notif.target_route = '/some/custom/route'
    const target = resolveTarget(notif, PLAYER_SLUG)
    expect(target!.to).toBe('/some/custom/route')
  })

  it('target_route /admin/data-health (route fantome legacy) est ignore et fallback', () => {
    // Avant 2026-05-20, data_health_check.go emettait des notifs avec
    // target_route = '/admin/data-health' (route jamais creee). Les notifs
    // persistees doivent retomber sur le fallback /players/{slug}/notifications.
    const notif = makeNotif('data_health_warning')
    notif.target_route = '/admin/data-health'
    const target = resolveTarget(notif, PLAYER_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`/players/${PLAYER_SLUG}/notifications`)
  })
})
