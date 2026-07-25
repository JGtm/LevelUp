/**
 * queries.ts — hooks TanStack Query pour la page Classement.
 *
 * Catégorie "csr-world" (défaut) : classement CSR mondial (snapshots Halo
 * Waypoint). Catégories de stats (kills/kda/…) : agrégation des joueurs croisés.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { LeaderboardCatalog, LeaderboardResponse } from '@/lib/api/types'

export interface LeaderboardParams {
  category?: string
  season?: string
  playlist?: string
  limit?: number
}

/**
 * useLeaderboard — récupère un classement.
 * GET /players/{slug}/pages/leaderboard?category=...&season=...&playlist=...&limit=...
 */
export function useLeaderboard(playerSlug: string, params: LeaderboardParams = {}) {
  const { category, season, playlist, limit } = params
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const qs = new URLSearchParams()
  if (category) qs.set('category', category)
  if (season) qs.set('season', season)
  if (playlist) qs.set('playlist', playlist)
  if (limit) qs.set('limit', String(limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''

  return useQuery<LeaderboardResponse>({
    queryKey: queryKeys.leaderboard(playerSlug, titleSlug, category, season, playlist),
    queryFn: () =>
      api.get<LeaderboardResponse>(`/players/${playerSlug}/pages/leaderboard${suffix}`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * useLeaderboardCatalog — saisons + playlists pour lesquelles des snapshots CSR
 * mondiaux existent (sélecteurs dynamiques). Cache long : le catalogue ne bouge
 * qu'au rythme des saisons.
 * GET /players/{slug}/pages/leaderboard/catalog
 */
export function useLeaderboardCatalog(playerSlug: string) {
  // Locale dans la clé : à la bascule de langue, la clé change → TanStack refetch
  // le catalogue (display_name saisons/playlists relocalisés côté backend). Remplace
  // l'ex-invalidation ciblée du layout titre (clé = invalidation).
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<LeaderboardCatalog>({
    queryKey: queryKeys.leaderboardCatalog(playerSlug, titleSlug, locale),
    queryFn: () =>
      api.get<LeaderboardCatalog>(`/players/${playerSlug}/pages/leaderboard/catalog`),
    enabled: !!playerSlug,
    staleTime: 30 * 60 * 1000,
  })
}
