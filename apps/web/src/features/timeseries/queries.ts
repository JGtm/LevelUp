/**
 * Queries TanStack Query — Timeseries (Slice 3B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  TimeseriesPageResponse,
  TimeseriesQueryRequest,
} from '@/lib/api/types'

export function useTimeseriesPage(
  playerSlug: string,
  request: TimeseriesQueryRequest,
  filterHash: string,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Les noms de cartes et d armes sont servis dans la langue de la requête : la locale scope la clé.
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.timeseries(playerSlug, titleSlug, filterHash, locale),
    queryFn: ({ signal }) =>
      api.post<TimeseriesPageResponse>(
        `/players/${playerSlug}/pages/timeseries`,
        request,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
