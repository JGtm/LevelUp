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
 * Lot 2-C (2026-07-23) : les fallbacks par catégorie émettent désormais la forme
 * TYPÉE title-scoped (`to` template `/{-$lang}/t/$titleSlug/players/$playerSlug/…`
 * + `params`), plus l'ancien littéral `/players/{slug}/…`. Les assertions ci-dessous
 * valident donc le couple (to template, params) — le `suffix` relatif au joueur (via
 * routeTemplateSuffix) reste le point de contrôle de la whitelist.
 *
 * Tests symétriques côté Go : apps/go-api/internal/api/post_sync_deltas_test.go
 *   - TestEmitPostSyncDeltas_AllTargetRoutesValid
 *   - TestEmitPostSyncDeltas_NoFantomRoutes
 */
import { describe, it, expect } from 'vitest'

import { routeTemplateSuffix } from '@/lib/title-routing'
import { resolveTarget } from './navigation'
import type { Notification, NotificationCategory } from './types'
import { ALL_CATEGORIES } from './types'

const PLAYER_SLUG = 'test-player'
const TITLE_SLUG = 'halo_infinite'

// Template title-scoped canonique (préfixe commun des fallbacks joueur).
const PLAYER_TPL = '/{-$lang}/t/$titleSlug/players/$playerSlug'
// Params attendus des fallbacks joueur sans segment dynamique supplémentaire.
const PLAYER_PARAMS = { titleSlug: TITLE_SLUG, playerSlug: PLAYER_SLUG }

// Whitelist des préfixes de routes acceptés.
// À synchroniser avec routeTree.gen.ts si la nav front évolue.
const VALID_TOP_ROUTES = ['/changelog', '/settings'] as const

const VALID_PLAYER_SUBPATHS = [
  '/home',
  '/synthesis',
  '/objectifs',
  '/ascension',
  '/ascension/objectifs',
  '/ascension/coaching',
  '/ascension/realisations',
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
  // Forme post-migration : template title-scoped
  // `/{-$lang}/t/$titleSlug/players/$playerSlug{suffix}`. Le suffixe relatif au joueur
  // (routeTemplateSuffix) porte l'identité de la page.
  if (!to.includes('/players/$playerSlug')) return false
  const subPath = routeTemplateSuffix(to)
  if (subPath === '') return true // /players/$playerSlug racine, sans sous-chemin
  if ((VALID_PLAYER_SUBPATHS as readonly string[]).includes(subPath)) return true
  // Routes dynamiques (ex: /matches/$matchId pour rival_encounter) : le sous-chemin
  // porte un id → valider sur le 1er segment. Les routes fantômes B1 (/defis, /sync)
  // n'ont pas leur 1er segment dans la whitelist, la protection tient.
  const firstSeg = '/' + subPath.split('/')[1]
  return (VALID_PLAYER_SUBPATHS as readonly string[]).includes(firstSeg)
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
      const target = resolveTarget(makeNotif(category), PLAYER_SLUG, TITLE_SLUG)
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
      const target = resolveTarget(makeNotif(category), PLAYER_SLUG, TITLE_SLUG)
      if (target === null) continue
      for (const fantom of FANTOM_ROUTES_B1) {
        expect(
          target.to,
          `category=${category} renvoie route fantome "${fantom}" (regression B1)`,
        ).not.toBe(fantom)
      }
    }
  })

  it('les fallbacks joueur portent titleSlug + playerSlug dans params', () => {
    // Défense lot 2-C : toute cible joueur (template) doit fournir les params de route
    // (sinon navigate n'interpole pas → URL morte). Les pages agnostiques et le
    // target_route backend (string verbatim) sont exemptés.
    for (const category of ALL_CATEGORIES) {
      const target = resolveTarget(makeNotif(category), PLAYER_SLUG, TITLE_SLUG)
      if (target === null) continue
      if (!target.to.includes('/players/$playerSlug')) continue
      expect(target.params?.titleSlug, `category=${category}`).toBe(TITLE_SLUG)
      expect(target.params?.playerSlug, `category=${category}`).toBe(PLAYER_SLUG)
    }
  })

  it('challenge_added cible /ascension/objectifs (onglet Objectifs)', () => {
    const target = resolveTarget(makeNotif('challenge_added'), PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`${PLAYER_TPL}/ascension/objectifs`)
    expect(target!.params).toEqual(PLAYER_PARAMS)
  })

  it('objective_assigned cible /ascension/objectifs (onglet Objectifs)', () => {
    const target = resolveTarget(makeNotif('objective_assigned'), PLAYER_SLUG, TITLE_SLUG)
    expect(target!.to).toBe(`${PLAYER_TPL}/ascension/objectifs`)
    expect(target!.params).toEqual(PLAYER_PARAMS)
  })

  it('objective_completed cible /ascension/realisations avec ancrage (AM-5)', () => {
    const target = resolveTarget(makeNotif('objective_completed'), PLAYER_SLUG, TITLE_SLUG)
    expect(target!.to).toBe(`${PLAYER_TPL}/ascension/realisations`)
    expect(target!.params).toEqual(PLAYER_PARAMS)
    expect(target!.search).toMatchObject({ selectedObjectiveId: '42' })
  })

  it('challenge_completed cible /ascension/realisations avec ancrage (AM-5)', () => {
    const target = resolveTarget(makeNotif('challenge_completed'), PLAYER_SLUG, TITLE_SLUG)
    expect(target!.to).toBe(`${PLAYER_TPL}/ascension/realisations`)
    expect(target!.params).toEqual(PLAYER_PARAMS)
    expect(target!.search).toMatchObject({ selectedChallengeId: '42' })
  })

  it('app_release cible /changelog (pas /help/changelog)', () => {
    const target = resolveTarget(makeNotif('app_release'), PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe('/changelog')
    expect(target!.params).toBeUndefined()
  })

  it('sync_error cible /settings (pas /players/$slug/sync)', () => {
    const target = resolveTarget(makeNotif('sync_error'), PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe('/settings')
    expect(target!.search).toMatchObject({ jobId: 'job-1' })
  })

  it('rival_encounter cible la match view via le target_route backend', () => {
    // Cas nominal : depuis le lot A (2026-07-23) le backend renvoie le format
    // title-préfixé /t/{titleSlug}/players/{slug}/matches/{match_id}, navigué VERBATIM
    // (zéro hop, aucun re-préfixage front).
    const notif = makeNotif('rival_encounter')
    notif.target_route = `/t/${TITLE_SLUG}/players/${PLAYER_SLUG}/matches/m-42`
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`/t/${TITLE_SLUG}/players/${PLAYER_SLUG}/matches/m-42`)
    expect(target!.params).toBeUndefined()
  })

  it('rival_encounter fallback sur match_id des params si target_route absent', () => {
    const notif = makeNotif('rival_encounter')
    notif.target_route = undefined
    notif.params = { match_id: 'm-99' }
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`${PLAYER_TPL}/matches/$matchId`)
    expect(target!.params).toEqual({ ...PLAYER_PARAMS, matchId: 'm-99' })
  })

  it('rival_encounter sans match_id ni target_route → aucune route', () => {
    const notif = makeNotif('rival_encounter')
    notif.target_route = undefined
    notif.params = {}
    expect(resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)).toBeNull()
  })

  it('target_route backend prend le pas sur le fallback', () => {
    const notif = makeNotif('app_release')
    notif.target_route = '/some/custom/route'
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target!.to).toBe('/some/custom/route')
  })

  it('target_route /admin/data-health (route fantome legacy) est ignore et fallback', () => {
    // Avant 2026-05-20, data_health_check.go emettait des notifs avec
    // target_route = '/admin/data-health' (route jamais creee). Les notifs
    // persistees doivent retomber sur le fallback /players/{slug}/notifications.
    const notif = makeNotif('data_health_warning')
    notif.target_route = '/admin/data-health'
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`${PLAYER_TPL}/notifications`)
    expect(target!.params).toEqual(PLAYER_PARAMS)
  })

  it('sync_error legacy (stock /players/{slug}/sync paramétré) est neutralisé → fallback /settings', () => {
    // Stock persisté antérieur au lot A : le backend émettait /players/{slug}/sync (route
    // MORTE, paramétrée par slug → invisible au Set exact-match). isFantomTargetRoute doit
    // l'ignorer pour retomber sur le fallback catégorie /settings (jobId préservé), et NON
    // naviguer le target_route verbatim (qui finirait en 404 via le splat legacy).
    const notif = makeNotif('sync_error')
    notif.target_route = '/players/some-player/sync'
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe('/settings')
    expect(target!.search).toMatchObject({ jobId: 'job-1' })
  })

  it('target_route nouveau format /t/{title}/players/{slug}/media reste prioritaire (non-fantôme)', () => {
    // Le prédicat ne doit PAS attraper une route joueur RÉELLE : un target_route
    // title-préfixé est navigué verbatim (zéro hop), prioritaire sur le fallback catégorie.
    const notif = makeNotif('media_added')
    notif.target_route = `/t/${TITLE_SLUG}/players/${PLAYER_SLUG}/media`
    const target = resolveTarget(notif, PLAYER_SLUG, TITLE_SLUG)
    expect(target).not.toBeNull()
    expect(target!.to).toBe(`/t/${TITLE_SLUG}/players/${PLAYER_SLUG}/media`)
    expect(target!.params).toBeUndefined()
  })
})
