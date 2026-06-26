import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { RelationsPageResponse, SeasonPassPageResponse } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export function useSeasonPassPage(playerSlug: string) {
  return useQuery<SeasonPassPageResponse>({
    queryKey: queryKeys.seasonPass(playerSlug),
    queryFn: () =>
      api.get<SeasonPassPageResponse>(`/players/${playerSlug}/pages/palmares/season-pass`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// useRelationsPage : hub Communauté > Relations. Consomme l'endpoint backend
// réel /pages/palmares/relations (forme {overview, relations[]}). Plus aucun
// mapper local — la donnée riche (win rates, frags échangés, badges) est servie
// par le service Go (internal/service/relations_service.go).
export function useRelationsPage(playerSlug: string) {
  return useQuery<RelationsPageResponse>({
    queryKey: queryKeys.palmaresRelations(playerSlug),
    queryFn: () =>
      api.get<RelationsPageResponse>(`/players/${playerSlug}/pages/palmares/relations`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
