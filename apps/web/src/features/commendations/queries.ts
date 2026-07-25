/**
 * Queries TanStack Query — Totaux des commendations natives (Halo 5, AXE B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { NativeCommendationsTotalsResponse } from '@/lib/api/types'

export function useCommendationTotals(playerSlug: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.commendationTotals(playerSlug, titleSlug),
    queryFn: () =>
      api.get<NativeCommendationsTotalsResponse>(
        `/players/${playerSlug}/commendations/totals`,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
