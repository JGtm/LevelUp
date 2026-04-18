/**
 * queries.ts — hooks TanStack Query pour le Classement CSR.
 * Sprint 54-E.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { LeaderboardResponse } from '@/lib/api/types'

/**
 * useLeaderboard — récupère le classement CSR d'un joueur.
 * GET /players/{slug}/pages/leaderboard?season=...&playlist=...
 */
export function useLeaderboard(
  playerSlug: string,
  seasonId?: string,
  playlistId?: string,
) {
  const params = new URLSearchParams()
  if (seasonId) params.set('season', seasonId)
  if (playlistId) params.set('playlist', playlistId)
  const qs = params.toString() ? `?${params.toString()}` : ''

  return useQuery<LeaderboardResponse>({
    queryKey: queryKeys.leaderboard(playerSlug, seasonId, playlistId),
    queryFn: () =>
      api.get<LeaderboardResponse>(`/players/${playerSlug}/pages/leaderboard${qs}`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
