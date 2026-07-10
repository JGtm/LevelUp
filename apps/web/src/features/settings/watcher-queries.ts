/**
 * Queries TanStack Query — Watcher présence Xbox RTA.
 *
 * 4 hooks :
 *   useWatcherStatus        — GET /watcher/status (polling 10s)
 *   useStartWatcherAuth     — mutation POST /watcher/auth/start
 *   useWatcherAuthPoll      — GET /watcher/auth/{id} (polling 3s quand pending)
 *   useUpdateWatcherSubscriptions — mutation PATCH /watcher/subscriptions
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type {
  WatcherStatusResponse,
  WatcherAuthAttempt,
  WatcherAuthStatus,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

// ---------------------------------------------------------------------------
// GET /api/v1/watcher/status
// ---------------------------------------------------------------------------

export function useWatcherStatus() {
  return useQuery({
    queryKey: queryKeys.watcher.status,
    queryFn: () => api.get<WatcherStatusResponse>('/watcher/status'),
    refetchInterval: 10_000,
    staleTime: 5_000,
    // Ne pas retry en boucle si 403 (non admin)
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message.includes('403')) return false
      return failureCount < 2
    },
  })
}

// ---------------------------------------------------------------------------
// POST /api/v1/watcher/auth/start
// ---------------------------------------------------------------------------

export function useStartWatcherAuth() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<WatcherAuthAttempt>('/watcher/auth/start', {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.watcher.status })
    },
  })
}

// ---------------------------------------------------------------------------
// GET /api/v1/watcher/auth/{attempt_id}  (poll)
// ---------------------------------------------------------------------------

export function useWatcherAuthPoll(attemptId: string | null) {
  return useQuery({
    queryKey: queryKeys.watcher.authPoll(attemptId ?? ''),
    queryFn: () =>
      api.get<WatcherAuthStatus>(`/watcher/auth/${attemptId}`),
    enabled: !!attemptId,
    refetchInterval: (query) => {
      const data = query.state.data
      if (!data) return 3_000
      if (data.status === 'pending') return 3_000
      return false
    },
    staleTime: 0,
  })
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/watcher/subscriptions
// ---------------------------------------------------------------------------

export function useUpdateWatcherSubscriptions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (subscribedPlayers: string[]) =>
      api.patch<{ subscribed_players: string[] }>('/watcher/subscriptions', {
        subscribed_players: subscribedPlayers,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.watcher.status })
    },
  })
}
