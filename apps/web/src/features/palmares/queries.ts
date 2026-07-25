import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type {
  FilterContextInput,
  RelationsMomentsResponse,
  RelationsPageResponse,
  SeasonPassPageResponse,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

export function useSeasonPassPage(playerSlug: string) {
  // Locale dans la clé : libellés du pass bakés serveur selon X-LevelUp-Locale
  // au fetch → refetch à la bascule de langue (cf. queryKeys.seasonPass).
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<SeasonPassPageResponse>({
    queryKey: queryKeys.seasonPass(playerSlug, titleSlug, locale),
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<RelationsPageResponse>({
    queryKey: [...queryKeys.palmaresRelations(playerSlug, titleSlug), hash],
    queryFn: () =>
      api.post<RelationsPageResponse>(`/players/${playerSlug}/pages/palmares/relations`, filterContext),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// useRelationsMoments : section « Moments & Rivalités » (Phase 3a). Consomme le
// sous-endpoint POST /pages/palmares/relations/moments (forme {heatmap,
// rivalries, top_relations}). Hérite de la même segmentation serveur que la
// page (FilterContextInput committed + hash dans la queryKey). Activé seulement
// quand `enabled` (la section est dépliée) pour ne pas charger inutilement.
export function useRelationsMoments(
  playerSlug: string,
  filterContext: FilterContextInput,
  hash: string,
  enabled: boolean,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery<RelationsMomentsResponse>({
    queryKey: [...queryKeys.palmaresRelations(playerSlug, titleSlug), 'moments', hash],
    queryFn: () =>
      api.post<RelationsMomentsResponse>(
        `/players/${playerSlug}/pages/palmares/relations/moments`,
        filterContext,
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60 * 1000,
  })
}
