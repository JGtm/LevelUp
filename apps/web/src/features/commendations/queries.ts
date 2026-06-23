/**
 * Queries TanStack Query — Totaux des commendations natives (Halo 5, AXE B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { NativeCommendationsTotalsResponse } from '@/lib/api/types'

export function useCommendationTotals(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.commendationTotals(playerSlug),
    queryFn: () =>
      api.get<NativeCommendationsTotalsResponse>(
        `/players/${playerSlug}/commendations/totals`,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
