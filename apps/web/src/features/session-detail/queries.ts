import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { SessionPageRequest, SessionPageResponse } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export function useSessionDetailPage(
  playerSlug: string,
  request: SessionPageRequest,
  filterHash: string,
  sessionLabel: string,
  compareSessionLabel: string,
  enableCompare: boolean,
) {
  return useQuery({
    queryKey: queryKeys.sessionDetail(
      playerSlug,
      filterHash,
      sessionLabel,
      compareSessionLabel,
      enableCompare,
    ),
    queryFn: () =>
      api.post<SessionPageResponse>(
        `/players/${playerSlug}/pages/sessions/detail`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
