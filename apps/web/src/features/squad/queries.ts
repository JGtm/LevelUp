/**
 * Queries TanStack Query — Escouade / Coéquipiers (Slice 6).
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammatesQueryRequest, TeammatesPageResponse } from '@/lib/api/types'

/**
 * Page Escouade (POST `/pages/teammates`).
 *
 * La clé ne porte PAS les sessions pickées (lot perf L4a, D4.1, 2026-09-23) : elles
 * vivent dans le store escouade et `filterContextHash` les couvre déjà. Le corps,
 * lui, envoie toujours `picked_squad_session_labels` (contrat serveur inchangé),
 * construit par l'appelant depuis ce même store.
 *
 * `enabled` : faux tant que la composition initiale n'est pas connue (D4.2) —
 * sinon une première requête partait sans coéquipier (6 s pour rien, mesuré).
 */
export function useTeammates(
  playerSlug: string,
  request: TeammatesQueryRequest,
  filterContextHash: string,
  confirmedGts: string[],
  enabled: boolean,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.teammates(
      playerSlug,
      titleSlug,
      filterContextHash,
      confirmedGts,
      request.locale ?? '',
      request.filter_exact_composition ?? true,
    ),
    queryFn: () =>
      api.post<TeammatesPageResponse>(
        `/players/${playerSlug}/pages/teammates`,
        request,
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60 * 1000,
    // Sans cela, chaque toggle de filtre (cascade, période, multi-select sessions,
    // coéquipier) renvoie `data` à `undefined` pendant le refetch — la barre
    // sticky perd son SessionMultiSelect, ses options de cascade et son count
    // pendant ~200-800 ms. keepPreviousData garde la sélection visible jusqu'à
    // ce que la nouvelle donnée arrive : pas de flicker, pas de filtre fantôme.
    placeholderData: keepPreviousData,
  })
}
