/**
 * Queries TanStack Query — page Achievements Xbox.
 *
 * Données stables (les définitions ne changent qu'à chaque DLC, la progression
 * ne change qu'aux unlocks utilisateur). staleTime généreux : 10 min.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AchievementsPageResponse } from '@/lib/api/types'

export function useAchievementsPage(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.achievements(playerSlug),
    queryFn: () =>
      api.get<AchievementsPageResponse>(`/players/${playerSlug}/pages/achievements`),
    enabled: !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}
