/**
 * Queries TanStack Query — Citations (Slice 2B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CitationsPageResponse, CitationsQueryRequest } from '@/lib/api/types'

export function useCitationsPage(
  playerSlug: string,
  request: CitationsQueryRequest,
  filterHash: string,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.citations(playerSlug, titleSlug, filterHash, locale),
    queryFn: () =>
      api.post<CitationsPageResponse>(`/players/${playerSlug}/pages/citations`, request),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
