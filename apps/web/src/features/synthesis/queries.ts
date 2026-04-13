/**
 * Queries TanStack Query — Synthèse (Slice 7).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { SynthesisQueryRequest, SynthesisPageResponse } from '@/lib/api/types'

export function useSynthesisPage(
  playerSlug: string,
  period: string,
  request: SynthesisQueryRequest,
) {
  return useQuery({
    queryKey: queryKeys.synthesis(playerSlug, period),
    queryFn: () =>
      api.post<SynthesisPageResponse>(
        `/players/${playerSlug}/pages/synthesis`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
