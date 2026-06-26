import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { FilterContextInput, RelationsPageResponse, SeasonPassPageResponse } from '@/lib/api/types'
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
// réel POST /pages/palmares/relations (forme {overview, relations[]}). Phase 2 :
// le FilterContextInput committed (expérience/classé, saison/période,
// playlist/mode, vue solo/escouade) est envoyé en body ; le service Go restreint
// l'agrégation au sous-ensemble de matchs. `hash` (hash stable du contexte
// committed) participe à la queryKey → refetch au clic « Analyser ».
export function useRelationsPage(playerSlug: string, filterContext: FilterContextInput, hash: string) {
  return useQuery<RelationsPageResponse>({
    queryKey: [...queryKeys.palmaresRelations(playerSlug), hash],
    queryFn: () =>
      api.post<RelationsPageResponse>(`/players/${playerSlug}/pages/palmares/relations`, filterContext),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
