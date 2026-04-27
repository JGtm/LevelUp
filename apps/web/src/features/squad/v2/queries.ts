/**
 * queries.ts — TanStack Query hook pour le endpoint Squad V2 (S11 backend).
 *
 * GET /api/v1/players/{slug}/pages/squad/v2?teammates=gt1,gt2&period=1y
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'

import type { SquadPageV2Response, SquadPeriod } from './types'

export interface UseSquadV2Params {
  playerSlug: string
  teammates?: string[]
  period?: SquadPeriod
}

export function useSquadV2({ playerSlug, teammates, period }: UseSquadV2Params) {
  const teammatesQuery = teammates && teammates.length > 0 ? teammates.join(',') : ''
  const periodQuery = period ?? 'all'
  const path = `/players/${playerSlug}/pages/squad/v2?teammates=${encodeURIComponent(
    teammatesQuery,
  )}&period=${periodQuery}`

  return useQuery<SquadPageV2Response>({
    queryKey: ['squad-v2', playerSlug, teammatesQuery, periodQuery],
    queryFn: () => api.get<SquadPageV2Response>(path),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
