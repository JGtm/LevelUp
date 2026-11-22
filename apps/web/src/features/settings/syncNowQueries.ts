/**
 * TanStack Query — « Synchroniser maintenant » : POST /sync/now lance le cycle, GET /sync/cycle dit
 * s'il tourne. L'état se relit toutes les 3 s tant qu'un cycle tourne, jamais au repos (une lecture
 * à l'ouverture de l'encadré, une autre après chaque lancement).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { components } from '@/lib/api/generated'
import { syncCycleRefetchMs } from './syncNowLogic'

export type SyncCycleState = components['schemas']['SyncCycleState']
export type SyncNowResponse = components['schemas']['SyncNowResponse']

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
