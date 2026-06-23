/**
 * Queries TanStack Query — Accueil Home (Slice 5).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { HomePageResponse, SeasonPassPageResponse } from '@/lib/api/types'

export function useHomePage(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.home(playerSlug),
    queryFn: () => api.get<HomePageResponse>(`/players/${playerSlug}/pages/home`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    // AXE E first-sync : tant que le joueur n'a AUCUN match synchronisé (première
    // synchro d'un titre fraîchement activé en cours), on poll toutes les 30 s pour
    // basculer automatiquement vers l'accueil dès que les données arrivent. S'arrête
    // (false) dès qu'au moins un match est présent → aucun poll pour un joueur établi.
    refetchInterval: (query) =>
      (query.state.data?.hero.kpis.total_matches ?? 0) <= 0 ? 30_000 : false,
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
