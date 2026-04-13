/**
 * Queries TanStack Query — page Carrière (Slice 2).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { CareerPageResponse, CareerTopMatchesResponse, CareerEncountersResponse } from '@/lib/api/types'

export function useCareerPage(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.career(playerSlug),
    queryFn: () => api.get<CareerPageResponse>(`/players/${playerSlug}/pages/career`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

export function useCareerTopMatches(playerSlug: string, enabled = false) {
  return useQuery({
    queryKey: queryKeys.careerTopMatches(playerSlug),
    queryFn: () => api.get<CareerTopMatchesResponse>(`/players/${playerSlug}/pages/career/top-matches`),
    enabled: enabled && !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}

export function useCareerEncounters(playerSlug: string, enabled = false) {
  return useQuery({
    queryKey: queryKeys.careerEncounters(playerSlug),
    queryFn: () => api.get<CareerEncountersResponse>(`/players/${playerSlug}/pages/career/encounters`),
    enabled: enabled && !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}
