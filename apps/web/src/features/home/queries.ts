/**
 * Queries TanStack Query — Accueil Home (Slice 5).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomePageResponse, SeasonPassPageResponse } from '@/lib/api/types'

export function useHomePage(playerSlug: string) {
  // Le titre courant scope la clé : au switch de titre, la clé change → refetch
  // des données du bon titre (plus de Spartan ID / playlists périmées).
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.home(playerSlug, titleSlug),
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

/**
 * @param enabled Gating multi-titre : passer `false` pour un titre sans la
 * capability `season_pass` (ex. Halo 5) afin que la requête ne parte pas.
 * Quand désactivée, `data` reste `undefined` et `isLoading`/`error` restent
 * neutres — les consommateurs doivent gérer l'absence de `seasonPass`.
 */
export function useSeasonPassPreview(playerSlug: string, enabled = true) {
  return useQuery({
    queryKey: queryKeys.seasonPass(playerSlug),
    queryFn: () => api.get<SeasonPassPageResponse>(`/players/${playerSlug}/pages/palmares/season-pass`),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}
