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
import type { Notification } from './types'

export interface NotifTarget {
  to: string
  search?: Record<string, unknown>
}

// Routes fantômes émises par d'anciennes versions du backend, persistées dans
// la table notifications. On les ignore pour retomber sur le fallback par
// catégorie plutôt que d'envoyer l'utilisateur sur une 404. À nettoyer côté DB
// quand pratique (UPDATE notifications SET target_route = NULL WHERE …).
const FANTOM_TARGET_ROUTES = new Set<string>([
  '/admin/data-health', // émis par data_health_check.go avant 2026-05-20
])

export function resolveTarget(notif: Notification, playerSlug: string): NotifTarget | null {
  // Priorité au target_route renvoyé par le backend (mapping spécifique au runtime),
  // sauf s'il fait partie de la liste de routes fantômes connues.
  if (notif.target_route && !FANTOM_TARGET_ROUTES.has(notif.target_route)) {
    return {
      to: notif.target_route,
      search: notif.target_search,
    }
  }

  // Fallback déterministe par catégorie.
  switch (notif.category) {
    case 'media_added':
      return {
        to: `/players/${playerSlug}/media`,
        search: notif.params?.match_id ? { matchId: String(notif.params.match_id) } : undefined,
      }
    case 'match_synced':
      return { to: `/players/${playerSlug}/explorer` }
    case 'objective_assigned':
    case 'objective_completed':
      return {
        to: `/players/${playerSlug}/ascension`,
        search: notif.params?.id ? { selectedObjectiveId: String(notif.params.id) } : undefined,
      }
    case 'challenge_added':
    case 'challenge_completed':
      // Les défis et objectifs vivent dans le tab "Profil & objectifs" d'Ascension.
      return {
        to: `/players/${playerSlug}/ascension`,
        search: notif.params?.id ? { selectedChallengeId: String(notif.params.id) } : undefined,
      }
    case 'season_pass_level':
      return { to: `/players/${playerSlug}/career/season-pass` }
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
      return {
        to: `/players/${playerSlug}/synthesis`,
        search: notif.params?.metric ? { metric: String(notif.params.metric) } : undefined,
      }
    case 'data_health_warning':
      // Pas de page admin dédiée. On renvoie sur la page notifications du
      // joueur qui affiche le body complet (compteurs + hint repair).
      return { to: `/players/${playerSlug}/notifications` }
    case 'trend_consolidate':
      // Onglet Entraînement (CoachFocusCard « Cap du moment »).
      return { to: `/players/${playerSlug}/ascension/coaching` }
    case 'title_ready':
      // MT-19 / axe E : accueil du titre — l'écran « première synchro » y bascule
      // sur le dashboard désormais peuplé. (Le backend renvoie déjà ce target_route.)
      return { to: `/players/${playerSlug}/home` }
    case 'rival_encounter':
      // Relations-E : la match view du duel. Le backend renvoie déjà ce
      // target_route (priorité ci-dessus) ; ce fallback couvre le cas où il
      // serait absent, à partir du match_id des params.
      return notif.params?.match_id
        ? { to: `/players/${playerSlug}/matches/${String(notif.params.match_id)}` }
        : null
    default:
      return null
  }
}
