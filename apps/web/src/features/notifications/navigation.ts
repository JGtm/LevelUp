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

export function resolveTarget(notif: Notification, playerSlug: string): NotifTarget | null {
  // Priorité au target_route renvoyé par le backend (mapping spécifique au runtime).
  if (notif.target_route) {
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
        to: `/players/${playerSlug}/objectifs`,
        search: notif.params?.id ? { selectedObjectiveId: String(notif.params.id) } : undefined,
      }
    case 'challenge_added':
    case 'challenge_completed':
      // Les défis sont un onglet de la page Objectifs (pas de route /defis dédiée).
      return {
        to: `/players/${playerSlug}/objectifs`,
        search: {
          tab: 'challenges',
          ...(notif.params?.id ? { selectedChallengeId: String(notif.params.id) } : {}),
        },
      }
    case 'season_pass_level':
      return { to: `/players/${playerSlug}/palmares/season-pass` }
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
    default:
      return null
  }
}
