/**
 * queries.ts — TanStack Query hook pour le endpoint Squad V2 (S11 backend).
 *
 * GET /api/v1/players/{slug}/pages/squad/v2?teammates=gt1,gt2&period=1y
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

import type { SquadPageV2Response, SquadPeriod } from './types'

export interface UseSquadV2Params {
  playerSlug: string
  teammates?: string[]
  period?: SquadPeriod
  experienceTypes?: string[]
  playlists?: string[]
}

export function useSquadV2({ playerSlug, teammates, period, experienceTypes, playlists }: UseSquadV2Params) {
  const teammatesQuery = teammates && teammates.length > 0 ? teammates.join(',') : ''
  const periodQuery = period ?? 'all'
  const expQuery = experienceTypes && experienceTypes.length > 0 ? experienceTypes.join(',') : ''
  const playlistsQuery = playlists && playlists.length > 0 ? playlists.join(',') : ''

  const params = new URLSearchParams({ teammates: teammatesQuery, period: periodQuery })
  if (expQuery) params.set('experience_types', expQuery)
  if (playlistsQuery) params.set('playlists', playlistsQuery)
  const path = `/players/${playerSlug}/pages/squad/v2?${params.toString()}`

  return useQuery<SquadPageV2Response>({
    queryKey: queryKeys.squadV2(
      playerSlug,
      teammates ?? [],
      periodQuery,
      experienceTypes ?? [],
      playlists ?? [],
    ),
    queryFn: () => api.get<SquadPageV2Response>(path),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
