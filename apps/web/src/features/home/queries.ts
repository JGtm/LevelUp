/**
 * Queries TanStack Query — Accueil Home (Slice 5).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { HomePageResponse, BattlePassResponse, SeasonPassPageResponse } from '@/lib/api/types'

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

export function useSeasonPassPreview(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.seasonPass(playerSlug),
    queryFn: () => api.get<SeasonPassPageResponse>(`/players/${playerSlug}/pages/palmares/season-pass`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}
