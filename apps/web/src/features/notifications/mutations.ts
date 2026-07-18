/**
 * Mutations TanStack Query (mark read/unread/dismiss/markAllRead/updatePrefs).
 *
 * Toutes optimistes : mise à jour cache immédiate, rollback si erreur, invalidate
 * en finale pour resynchro.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MarkResult,
  Notification,
  NotificationCategory,
  NotificationPreference,
  UnreadCount,
} from './types'

interface MutationCtx {
  /** Slug joueur — utilisé pour scoper les query keys invalidées. */
  playerSlug: string
}

export function useMarkRead({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<MarkResult, Error, number[], CacheSnapshot>({
    mutationFn: async (ids: number[]) =>
      api.post<MarkResult>(
        `/players/${playerSlug}/notifications/mark-read`,
        { ids },
      ),
    onMutate: async (ids) => {
      const snapshot = patchListsForRead(qc, playerSlug, ids, true)
      patchUnreadCount(qc, playerSlug, -ids.length)
      return snapshot
    },
    onError: (_err, _ids, snapshot) => restore(qc, snapshot),
    onSettled: () => invalidateAll(qc, playerSlug),
  })
}

/**
 * Flush « keepalive » des ids lus, best-effort au unload (`visibilitychange`
 * hidden / `pagehide`) — filet du chemin nominal open→false qui, monté, ne se
 * déclenche pas sur F5 / fermeture d'onglet (W5). Réutilise le client keepalive
 * (`api.postKeepalive`), pas de patch de cache optimiste : la page part ou
 * reviendra via refetch. `void` + `.catch` : perte tolérée si le navigateur
 * coupe la requête au unload (aucun log utile — le contexte disparaît).
 */
export function markNotificationsReadKeepalive(playerSlug: string, ids: number[]): void {
  if (ids.length === 0) return
  void api
    .postKeepalive<MarkResult>(`/players/${playerSlug}/notifications/mark-read`, { ids })
    .catch(() => {
      /* best-effort : requête coupée au unload → sera re-marquée au prochain cycle */
    })
}

export function useMarkUnread({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<void, Error, number, CacheSnapshot>({
    mutationFn: async (id: number) =>
      api.patch<void>(`/players/${playerSlug}/notifications/${id}/unread`),
    onMutate: async (id) => {
      const snapshot = patchListsForRead(qc, playerSlug, [id], false)
      patchUnreadCount(qc, playerSlug, +1)
      return snapshot
    },
    onError: (_err, _id, snapshot) => restore(qc, snapshot),
    onSettled: () => invalidateAll(qc, playerSlug),
  })
}

export function useMarkAllRead({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<MarkResult, Error, NotificationCategory | undefined>({
    mutationFn: async (category) =>
      api.post<MarkResult>(
        `/players/${playerSlug}/notifications/mark-all-read`,
        category ? { category } : undefined,
      ),
    onSettled: () => invalidateAll(qc, playerSlug),
  })
}

export function useDismiss({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<void, Error, number>({
    mutationFn: async (id: number) =>
      api.delete<void>(`/players/${playerSlug}/notifications/${id}`),
    onSettled: () => invalidateAll(qc, playerSlug),
  })
}

export function useUpdatePreferences({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<{ items: NotificationPreference[] }, Error, NotificationPreference[]>({
    mutationFn: async (items) =>
      api.patch<{ items: NotificationPreference[] }>(
        `/players/${playerSlug}/notifications/preferences`,
        { items },
      ),
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.notificationsPreferences(playerSlug), data)
    },
  })
}

/**
 * Émet une notification de test côté serveur (POST /notifications/test).
 * Permet de valider visuellement le pipeline UI (toast + dropdown) depuis le
 * bouton "Envoyer une notification de test" du Settings tab.
 */
export function useSendTestNotification({ playerSlug }: MutationCtx) {
  const qc = useQueryClient()
  return useMutation<void, Error, void>({
    mutationFn: async () =>
      api.post<void>(`/players/${playerSlug}/notifications/test`),
    onSettled: () => {
      // Invalide la liste pour que la notif test apparaisse dans le dropdown.
      qc.invalidateQueries({ queryKey: queryKeys.notificationsAll(playerSlug) })
    },
  })
}

// ─── helpers ──────────────────────────────────────────────────────────────

interface CacheSnapshot {
  lists: Array<[unknown, unknown]>
  unread?: UnreadCount
}

function patchListsForRead(
  qc: ReturnType<typeof useQueryClient>,
  playerSlug: string,
  ids: number[],
  read: boolean,
): CacheSnapshot {
  const idSet = new Set(ids)
  const now = new Date().toISOString()
  // Match toutes les query keys ['notifications', playerSlug, ...]
  const matches = qc.getQueriesData<{ items: Notification[] }>({
    queryKey: queryKeys.notificationsAll(playerSlug),
  })
  const snapshot: CacheSnapshot = { lists: [] }
  for (const [key, data] of matches) {
    if (!data || !Array.isArray((data as { items?: unknown[] }).items)) continue
    snapshot.lists.push([key, data])
    const next = {
      ...(data as { items: Notification[] }),
      items: data.items.map((n) =>
        idSet.has(n.id) ? { ...n, read_at: read ? now : null } : n,
      ),
    }
    qc.setQueryData(key, next)
  }
  return snapshot
}

function patchUnreadCount(
  qc: ReturnType<typeof useQueryClient>,
  playerSlug: string,
  delta: number,
) {
  qc.setQueryData<UnreadCount>(queryKeys.notificationsUnreadCount(playerSlug), (cur) => {
    if (!cur) return cur
    return { ...cur, count: Math.max(0, cur.count + delta) }
  })
}

function restore(qc: ReturnType<typeof useQueryClient>, snap: CacheSnapshot | undefined) {
  if (!snap) return
  for (const [key, data] of snap.lists) {
    qc.setQueryData(key as readonly unknown[], data)
  }
}

function invalidateAll(qc: ReturnType<typeof useQueryClient>, playerSlug: string) {
  qc.invalidateQueries({ queryKey: queryKeys.notificationsAll(playerSlug) })
}
