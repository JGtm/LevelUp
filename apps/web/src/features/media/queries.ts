/**
 * Queries TanStack Query — Médias (Slice 8).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MediaQueryRequest, MediaPageResponse } from '@/lib/api/types'

export function useMediaPage(
  playerSlug: string,
  request: MediaQueryRequest,
  page: number,
) {
  return useQuery({
    queryKey: queryKeys.media(playerSlug, page),
    queryFn: () =>
      api.post<MediaPageResponse>(
        `/players/${playerSlug}/pages/media`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}
