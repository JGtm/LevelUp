/**
 * Mapping centralisé : catégorie de notification → route + search params.
 *
 * Utilisé par :
 * - le clic sur un item du dropdown (NotificationItem.tsx)
 * - le clic sur l'action "Voir" d'un toast (toastBridge.tsx)
 * - le clic sur la page dédiée (NotificationsPage.tsx)
 *
 * Le backend peut surcharger via target_route/target_search ; ce mapping ne
 * sert que de fallback quand ces champs sont vides.
 */
import { playerRelativePath } from '@/lib/title-routing/playerScopedPath'
import type { Notification } from './types'

export interface NotifTarget {
  /**
   * Cible passée à `navigate`/`<Link>`. Deux formes coexistent :
   *  - TEMPLATE de route title-scoped `FileRouteTypes['to']`
   *    (`/{-$lang}/t/$titleSlug/players/$playerSlug/…`, interpolé au runtime via
   *    `params`) pour les fallbacks par catégorie et les pages agnostiques
   *    (`/changelog`, `/settings`) — forme canonique post-migration (lot 2-C) ;
   *  - une string verbatim quand elle vient du `target_route` backend (runtime).
   */
  to: string
  /** Params de route (titleSlug/playerSlug[/matchId]) pour la forme template. */
  params?: Record<string, string>
  search?: Record<string, unknown>
}

// Routes fantômes émises par d'anciennes versions du backend, persistées dans
// la table notifications. On les ignore pour retomber sur le fallback par
// catégorie plutôt que d'envoyer l'utilisateur sur une 404. À nettoyer côté DB
// quand pratique (UPDATE notifications SET target_route = NULL WHERE …).
//
// Ce Set ne tient QUE les fantômes EXACTS (route agnostique fixe). Les fantômes
// PARAMÉTRÉS par slug ne peuvent pas y figurer (exact-match) → isFantomTargetRoute.
const FANTOM_TARGET_ROUTES = new Set<string>([
  '/admin/data-health', // émis par data_health_check.go avant 2026-05-20
])

// isFantomTargetRoute : true si `route` est une cible fantôme à ignorer au profit du
// fallback par catégorie — fantômes EXACTS (Set) ET fantômes PARAMÉTRÉS par slug.
// Aujourd'hui, fantôme paramétré = le stock legacy sync_error `/players/{slug}/sync`
// (route front inexistante ; l'émission n'en produit plus depuis le lot A — cf.
// sync_handler.go — mais les notifications déjà persistées subsistent, avec un slug
// variable jamais capté par un Set exact-match). playerRelativePath tolère l'ancien ET
// le nouveau format → le stock est neutralisé quel que soit son format persisté.
function isFantomTargetRoute(route: string): boolean {
  if (FANTOM_TARGET_ROUTES.has(route)) return true
  return playerRelativePath(route) === '/sync'
}

/**
 * Fallback par catégorie vers une page JOUEUR title-scoped : forme typée `to` (template
 * `FileRouteTypes['to']`) + `params` (lot 2-C — plus aucun littéral `/players/` d'URL
 * ancien format). Le segment `lang` est hérité de l'URL courante (jamais dans `params`,
 * prouvé par `langSegmentInheritance.test.ts`). `suffix` est le chemin sous la racine
 * joueur (`/media`, `/ascension/objectifs`, …).
 */
function playerTarget(
  titleSlug: string,
  playerSlug: string,
  suffix: string,
  search?: Record<string, unknown>,
): NotifTarget {
  return {
    to: `/{-$lang}/t/$titleSlug/players/$playerSlug${suffix}`,
    params: { titleSlug, playerSlug },
    search,
  }
}

export function resolveTarget(
  notif: Notification,
  playerSlug: string,
  titleSlug: string,
): NotifTarget | null {
  // Priorité au target_route renvoyé par le backend (mapping spécifique au runtime),
  // sauf s'il est fantôme (isFantomTargetRoute). Depuis le lot A (2026-07-23) le backend
  // émet le format title-préfixé sous /t/{titleSlug}/players/…, navigué VERBATIM (aucun
  // re-préfixage) donc zéro hop. Le stock de notifications PERSISTÉES avant ce lot porte
  // l'ancien format sous /players/… : les cibles vers une route RÉELLE sont prises en
  // charge par le splat de redirection legacy (Phase 3, 1 hop) ; les fantômes paramétrés
  // (sync_error /players/{slug}/sync, route morte) sont neutralisés ici → fallback catégorie.
  if (notif.target_route && !isFantomTargetRoute(notif.target_route)) {
    return {
      to: notif.target_route,
      search: notif.target_search,
    }
  }

  // Fallback déterministe par catégorie.
  switch (notif.category) {
    case 'media_added':
      return playerTarget(
        titleSlug,
        playerSlug,
        '/media',
        notif.params?.match_id ? { matchId: String(notif.params.match_id) } : undefined,
      )
    case 'match_synced':
      return playerTarget(titleSlug, playerSlug, '/explorer')
    case 'objective_assigned':
      // Objectif attribué → onglet "Objectifs" (couche Prestige, l'actif).
      return playerTarget(
        titleSlug,
        playerSlug,
        '/ascension/objectifs',
        notif.params?.id ? { selectedObjectiveId: String(notif.params.id) } : undefined,
      )
    case 'objective_completed':
      // Objectif complété → onglet "Réalisations" où l'item apparaît en carte
      // moment (AM-5). selectedObjectiveId ancre/surligne la carte concernée.
      return playerTarget(
        titleSlug,
        playerSlug,
        '/ascension/realisations',
        notif.params?.id ? { selectedObjectiveId: String(notif.params.id) } : undefined,
      )
    case 'challenge_added':
      return playerTarget(
        titleSlug,
        playerSlug,
        '/ascension/objectifs',
        notif.params?.id ? { selectedChallengeId: String(notif.params.id) } : undefined,
      )
    case 'challenge_completed':
      // Défi complété → "Réalisations" avec ancrage de la carte moment (AM-5).
      return playerTarget(
        titleSlug,
        playerSlug,
        '/ascension/realisations',
        notif.params?.id ? { selectedChallengeId: String(notif.params.id) } : undefined,
      )
    case 'season_pass_level':
      return playerTarget(titleSlug, playerSlug, '/career/season-pass')
    case 'app_release':
      return { to: '/changelog' }
    case 'sync_error':
      // Pas de route /sync dédiée — la gestion sync se fait dans Settings.
      return {
        to: '/settings',
        search: notif.params?.job_id ? { jobId: String(notif.params.job_id) } : undefined,
      }
    case 'personal_record':
    case 'threshold_crossed':
      return playerTarget(
        titleSlug,
        playerSlug,
        '/synthesis',
        notif.params?.metric ? { metric: String(notif.params.metric) } : undefined,
      )
    case 'data_health_warning':
      // Pas de page admin dédiée. On renvoie sur la page notifications du
      // joueur qui affiche le body complet (compteurs + hint repair).
      return playerTarget(titleSlug, playerSlug, '/notifications')
    case 'trend_consolidate':
      // Onglet Entraînement (CoachFocusCard « Cap du moment »).
      return playerTarget(titleSlug, playerSlug, '/ascension/coaching')
    case 'title_ready':
      // MT-19 / axe E : accueil du titre — l'écran « première synchro » y bascule
      // sur le dashboard désormais peuplé. (Le backend renvoie déjà ce target_route.)
      return playerTarget(titleSlug, playerSlug, '/home')
    case 'rival_encounter':
      // Relations-E : la match view du duel. Le backend renvoie déjà ce
      // target_route (priorité ci-dessus) ; ce fallback couvre le cas où il
      // serait absent, à partir du match_id des params. matchId est un param de
      // ROUTE (segment `$matchId`), pas un search param.
      return notif.params?.match_id
        ? {
            to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
            params: { titleSlug, playerSlug, matchId: String(notif.params.match_id) },
          }
        : null
    default:
      return null
  }
}
