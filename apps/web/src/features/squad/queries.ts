/**
 * Queries TanStack Query — Escouade / Coéquipiers (Slice 6).
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammatesQueryRequest, TeammatesPageResponse } from '@/lib/api/types'

export function useTeammates(
  playerSlug: string,
  request: TeammatesQueryRequest,
  filterContextHash: string,
  confirmedGts: string[],
  sessionLabels: string[] = [],
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.teammates(playerSlug, titleSlug, filterContextHash, confirmedGts, sessionLabels, request.locale ?? ''),
    queryFn: () =>
      api.post<TeammatesPageResponse>(
        `/players/${playerSlug}/pages/teammates`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    // Sans cela, chaque toggle de filtre (cascade, période, multi-select sessions,
    // coéquipier) renvoie `data` à `undefined` pendant le refetch — la barre
    // sticky perd son SessionMultiSelect, ses options de cascade et son count
    // pendant ~200-800 ms. keepPreviousData garde la sélection visible jusqu'à
    // ce que la nouvelle donnée arrive : pas de flicker, pas de filtre fantôme.
    placeholderData: keepPreviousData,
  })
}
