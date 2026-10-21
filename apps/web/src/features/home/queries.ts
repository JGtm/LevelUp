/**
 * Queries TanStack Query — Accueil Home (Slice 5).
 */
import { useCallback, useMemo, useState } from 'react'
import { useQueries, useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeMatchesPage, HomePageResponse, SeasonPassPageResponse } from '@/lib/api/types'
import {
  homeRefetchInterval,
  homeStaleTime,
  tilesPageRefetchInterval,
  tilesPageStaleTime,
} from '@/lib/analysis-pending/analysisPending.logic'
import { homeMatchesUrl, mergeHomeTiles, type HomeTiles, type HomeTilesHead } from '@/lib/home-tiles/homeTiles.logic'

const HOME_STALE_MS = 5 * 60 * 1000

export function useHomePage(playerSlug: string) {
  // Le titre courant scope la clé : au switch de titre, la clé change → refetch
  // des données du bon titre (plus de Spartan ID / playlists périmées).
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // La locale scope aussi la clé : les libellés (titres de défis, map/mode) sont
  // bakés serveur selon X-LevelUp-Locale au fetch → refetch à la bascule de langue.
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.home(playerSlug, titleSlug, locale),
    queryFn: ({ signal }) =>
      api.get<HomePageResponse>(`/players/${playerSlug}/pages/home`, undefined, { signal }),
    enabled: !!playerSlug,
    staleTime: (query) => homeStaleTime(query.state.data, HOME_STALE_MS),
    // AXE E first-sync : tant que le joueur n'a AUCUN match synchronisé (première
    // synchro d'un titre fraîchement activé en cours), on poll toutes les 30 s pour
    // basculer automatiquement vers l'accueil dès que les données arrivent. Ensuite, 60 s
    // tant qu'une tuile affichée a son analyse de film en cours (`analysis_pending`),
    // puis plus aucun poll. Règle dans lib/analysis-pending/analysisPending.logic.ts.
    refetchInterval: (query) => homeRefetchInterval(query.state.data),
  })
}

/**
 * @param enabled Gating multi-titre : passer `false` pour un titre sans la
 * capability `season_pass` (ex. Halo 5) afin que la requête ne parte pas.
 * Quand désactivée, `data` reste `undefined` et `isLoading`/`error` restent
 * neutres — les consommateurs doivent gérer l'absence de `seasonPass`.
 */
export function useSeasonPassPreview(playerSlug: string, enabled = true) {
  // Locale dans la clé : libellés du pass bakés serveur selon X-LevelUp-Locale.
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.seasonPass(playerSlug, titleSlug, locale),
    queryFn: ({ signal }) =>
      api.get<SeasonPassPageResponse>(
        `/players/${playerSlug}/pages/palmares/season-pass`,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

/**
 * Favoris de l'Accueil : leur propre lecture (`kind=favorites`), servis en entier. Rechargés seuls
 * après un favori basculé (`useSetMatchFavorite`) et, tant qu'une de leurs tuiles a son analyse en
 * cours, toutes les 60 s.
 */
export function useHomeFavorites(playerSlug: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.homeFavorites(playerSlug, titleSlug, locale),
    queryFn: ({ signal }) =>
      api.get<HomeMatchesPage>(homeMatchesUrl(playerSlug, { favorites: true }), undefined, { signal }),
    enabled: !!playerSlug,
    staleTime: (query) => tilesPageStaleTime(query.state.data, HOME_STALE_MS),
    refetchInterval: (query) => tilesPageRefetchInterval(query.state.data),
  })
}

export interface HomeTilesState extends HomeTiles {
  /** La page demandée n'est pas encore arrivée. */
  loadingMore: boolean
  /** La page demandée a échoué : `retry` la redemande. */
  loadError: boolean
  /** Demande la page suivante si elle existe. Idempotent (appelable à chaque rendu). */
  fetchMore: () => void
  retry: () => void
}

/**
 * Tuiles de l'historique : la tête (déjà chargée avec la page) puis les pages demandées au fil du
 * défilement. Une requête PAR PAGE, clé = curseur : chaque page se recharge seule quand une de ses
 * tuiles a son analyse en cours. Une tête qui change de curseur (synchro) repart de zéro : des
 * pages chargées sur l'ancienne tête ne se recollent jamais à la nouvelle.
 */
export function useHomeTiles(playerSlug: string, head: HomeTilesHead): HomeTilesState {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  const headCursor = head.recent_matches_next_before
  const [requested, setRequested] = useState<{ head: string | undefined; cursors: string[] }>({
    head: undefined,
    cursors: [],
  })
  const cursors = useMemo(
    () => (requested.head === headCursor ? requested.cursors : []),
    [requested, headCursor],
  )

  const results = useQueries({
    queries: cursors.map((before) => ({
      queryKey: queryKeys.homeMatchesPage(playerSlug, titleSlug, locale, before),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        api.get<HomeMatchesPage>(homeMatchesUrl(playerSlug, { before }), undefined, { signal }),
      staleTime: (query: { state: { data: HomeMatchesPage | undefined } }) =>
        tilesPageStaleTime(query.state.data, HOME_STALE_MS),
      refetchInterval: (query: { state: { data: HomeMatchesPage | undefined } }) =>
        tilesPageRefetchInterval(query.state.data),
    })),
  })
  const merged = mergeHomeTiles(head, results.map((r) => r.data))
  const last = results[results.length - 1]
  const loadingMore = !!last && last.isFetching && !last.data
  const loadError = !!last && last.isError && !last.data

  const { hasMore, nextBefore } = merged
  const fetchMore = useCallback(() => {
    if (!hasMore || !nextBefore || loadingMore || loadError || cursors.includes(nextBefore)) return
    setRequested({ head: headCursor, cursors: [...cursors, nextBefore] })
  }, [hasMore, nextBefore, loadingMore, loadError, cursors, headCursor])
  const retry = useCallback(() => {
    void last?.refetch()
  }, [last])

  return { ...merged, loadingMore, loadError, fetchMore, retry }
}
