/**
 * Queries TanStack Query — Accueil Home (Slice 5).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { HomePageResponse, BattlePassResponse, ChallengesResponse } from '@/lib/api/types'

export function useHomePage(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.home(playerSlug),
    queryFn: () => api.get<HomePageResponse>(`/players/${playerSlug}/pages/home`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

export function useBattlePass(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.battlepass(playerSlug),
    queryFn: () => api.get<BattlePassResponse>(`/players/${playerSlug}/battlepass`),
    enabled: !!playerSlug,
    staleTime: 4 * 60 * 60 * 1000, // 4h — données live avec cache process
    retry: false,
  })
}

export function useChallenges(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.challenges(playerSlug),
    queryFn: () => api.get<ChallengesResponse>(`/players/${playerSlug}/challenges`),
    enabled: !!playerSlug,
    staleTime: 60 * 60 * 1000, // 1h
    retry: false,
  })
}
