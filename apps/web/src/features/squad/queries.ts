/**
 * Queries TanStack Query — Escouade / Coéquipiers (Slice 6).
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  CompositionSessionsResponse,
  TeammatesQueryRequest,
  TeammatesPageResponse,
} from '@/lib/api/types'

/**
 * Page Escouade (POST `/pages/teammates`).
 *
 * La clé ne porte PAS les sessions pickées (lot perf L4a, D4.1, 2026-09-23) : elles
 * vivent dans le store escouade et `filterContextHash` les couvre déjà. Le corps,
 * lui, envoie toujours `picked_squad_session_labels` (contrat serveur inchangé),
 * construit par l'appelant depuis ce même store.
 *
 * `enabled` : faux tant que la composition initiale n'est pas connue (D4.2) —
 * sinon une première requête partait sans coéquipier (6 s pour rien, mesuré) — puis,
 * depuis le lot L4b, tant que l'ancrage de session n'est pas décidé
 * (`useSquadPageRequests`).
 *
 * `signal` (lot perf L4b, découverte des lots L3 et L4a) : une page devenue inutile
 * (clic du rail, composition changée, page quittée) est abandonnée côté client ; le
 * serveur, dont le contexte est alors annulé, s'arrête entre deux sections (499).
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
    queryFn: ({ signal }) =>
      api.post<TeammatesPageResponse>(
        `/players/${playerSlug}/pages/teammates`,
        request,
        undefined,
        { signal },
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

/** Chemin de la lecture légère : `?teammates=a,b&exact=true`, sans `teammates` si vide. */
function compositionSessionsPath(playerSlug: string, teammates: string[], exact: boolean): string {
  const params = new URLSearchParams()
  if (teammates.length > 0) params.set('teammates', teammates.join(','))
  params.set('exact', String(exact))
  return `/players/${playerSlug}/pages/teammates/sessions?${params.toString()}`
}

/**
 * Sessions de la composition SANS la page (GET `/pages/teammates/sessions`, lot perf
 * L4b, 2026-09-23) : les mêmes `composition_sessions` et `latest_composition_session`
 * que la réponse lourde, en quelques dizaines de millisecondes. La page Escouade s'y
 * ancre sur la bonne session AVANT d'envoyer sa requête lourde (`useSquadPageRequests`).
 *
 * `enabled` : le même verrou que la requête lourde (état de montage posé, composition
 * initiale connue — lot L4a). `retry: false` : un échec retombe AUSSITÔT sur la réponse
 * lourde, qui porte les mêmes champs (repli L4a) — rejouer retarderait la page du backoff
 * de l'application (1 s puis 2 s) pour une donnée disponible ailleurs.
 */
export function useCompositionSessions(
  playerSlug: string,
  teammates: string[],
  exact: boolean,
  enabled: boolean,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.compositionSessions(playerSlug, titleSlug, teammates, exact),
    queryFn: ({ signal }) =>
      api.get<CompositionSessionsResponse>(
        compositionSessionsPath(playerSlug, teammates, exact),
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
    retry: false,
  })
}
