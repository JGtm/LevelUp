/**
 * TanStack Query — « Synchroniser maintenant » : POST /sync/now lance le cycle, GET /sync/cycle dit
 * s'il tourne ; GET /players/{slug}/sync/history donne l'état de l'historique du joueur par titre suivi.
 *
 * L'état du cycle se relit toutes les 3 s tant qu'il tourne, jamais au repos. L'état de l'historique se
 * relit toutes les 5 s pendant un cycle, toutes les 30 s pendant un rattrapage, jamais au repos, et une
 * fois à la fin de chaque cycle. La clé de l'historique porte la langue : le serveur résout les libellés.
 */
import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { components } from '@/lib/api/generated'
import type { Locale } from '@/lib/i18n/locale'
import { historyRefetchMs, syncCycleRefetchMs } from './syncNowLogic'

export type SyncCycleState = components['schemas']['SyncCycleState']
export type SyncNowResponse = components['schemas']['SyncNowResponse']
export type HistorySyncStatus = components['schemas']['HistorySyncStatus']
export type TitleHistoryStatus = components['schemas']['TitleHistoryStatus']
export type HistoryPass = components['schemas']['HistoryPass']

export function useSyncCycle(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.syncCycle,
    queryFn: () => api.get<SyncCycleState>('/sync/cycle'),
    staleTime: 0,
    refetchInterval: (query) => syncCycleRefetchMs(query.state.data),
    enabled,
  })
}

/** Lance un cycle tout de suite ; l'état relu juste après dit qu'il tourne (ou qu'un autre tournait déjà). */
export function useSyncNow() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<SyncNowResponse>('/sync/now', {}),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.syncCycle }),
  })
}

/**
 * L'état de l'historique du joueur. `cycleRunning` règle la cadence de relecture ; le passage d'un cycle
 * « en cours » à « terminé » relit l'état une dernière fois, pour que les matchs ajoutés apparaissent.
 */
export function useHistorySyncStatus(playerSlug: string, locale: Locale, enabled: boolean, cycleRunning: boolean) {
  const qc = useQueryClient()
  const wasRunning = useRef(false)
  useEffect(() => {
    if (wasRunning.current && !cycleRunning) {
      void qc.invalidateQueries({ queryKey: queryKeys.historySyncStatus(playerSlug, locale) })
    }
    wasRunning.current = cycleRunning
  }, [cycleRunning, playerSlug, locale, qc])
  return useQuery({
    queryKey: queryKeys.historySyncStatus(playerSlug, locale),
    queryFn: () => api.get<HistorySyncStatus>(`/players/${playerSlug}/sync/history`),
    staleTime: 0,
    refetchInterval: (query) => historyRefetchMs(query.state.data, cycleRunning),
    enabled: enabled && !!playerSlug,
  })
}
