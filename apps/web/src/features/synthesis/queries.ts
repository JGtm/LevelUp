/**
 * Queries TanStack Query — Synthèse (Slice 7).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { SynthesisQueryRequest, SynthesisPageResponse, PeriodInput } from '@/lib/api/types'

export function useSynthesisPage(
  playerSlug: string,
  period: PeriodInput | undefined,
  request: SynthesisQueryRequest,
) {
  const scopeHash = `${period?.start_date ?? ''}_${period?.end_date ?? ''}`
  return useQuery({
    queryKey: queryKeys.synthesis(playerSlug, scopeHash),
    queryFn: () =>
      api.post<SynthesisPageResponse>(
        `/players/${playerSlug}/pages/synthesis`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
