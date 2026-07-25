/**
 * TanStack Query hooks pour les notifications.
 *
 * Stratégie polling :
 * - unread-count : 30s (payload léger, badge cloche)
 * - liste dropdown : enabled à l'ouverture, refetch 60s
 * - liste page dédiée : 60s + bouton refresh manuel
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  Notification,
  NotificationListFilter,
  NotificationListResult,
  NotificationPreference,
  UnreadCount,
} from './types'

export type NotificationsListResponse = NotificationListResult

/** Liste paginée (le composant fournit son scope via filter). */
export function useNotificationsList(
  playerSlug: string,
  filter: NotificationListFilter,
  options?: { enabled?: boolean; refetchInterval?: number },
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const qs = buildQueryString(filter)
  return useQuery<NotificationsListResponse>({
    queryKey: queryKeys.notifications(playerSlug, titleSlug, filter),
    queryFn: () =>
      api.get<NotificationsListResponse>(
        `/players/${playerSlug}/notifications${qs}`,
      ),
    enabled: !!playerSlug && (options?.enabled ?? true),
    refetchInterval: options?.refetchInterval,
    refetchOnWindowFocus: true,
    staleTime: 30_000,
  })
}

/** Compteur léger pour le badge cloche. Polling 30s. */
export function useUnreadCount(playerSlug: string, enabled: boolean = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<UnreadCount>({
    queryKey: queryKeys.notificationsUnreadCount(playerSlug, titleSlug),
    queryFn: () =>
      api.get<UnreadCount>(`/players/${playerSlug}/notifications/unread-count`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
    staleTime: 15_000,
  })
}

/** Préférences par catégorie. Cache plus long, change rarement. */
export function useNotificationPreferences(playerSlug: string, enabled: boolean = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<{ items: NotificationPreference[] }>({
    queryKey: queryKeys.notificationsPreferences(playerSlug, titleSlug),
    queryFn: () =>
      api.get<{ items: NotificationPreference[] }>(
        `/players/${playerSlug}/notifications/preferences`,
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60_000,
  })
}

function buildQueryString(f: NotificationListFilter): string {
  const params = new URLSearchParams()
  if (f.unread_only) params.set('unread_only', 'true')
  if (f.category) params.set('category', f.category)
  if (f.limit && f.limit > 0) params.set('limit', String(f.limit))
  if (f.before_id && f.before_id > 0) params.set('before_id', String(f.before_id))
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

/** Helper exposé pour le toastBridge — extrait l'ID max d'une liste. */
export function maxIdOf(items: Notification[] | undefined): number | null {
  if (!items || items.length === 0) return null
  return items.reduce((m, n) => (n.id > m ? n.id : m), 0) || null
}
