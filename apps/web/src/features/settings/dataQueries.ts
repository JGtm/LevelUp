/**
 * TanStack Query — état des données de l'application de bureau (lot L22) : sauvegarde périodique et
 * mémoire (GET /settings/data), lancement d'une sauvegarde (POST /settings/data/backup/run).
 *
 * L'état se relit toutes les 5 s tant qu'une sauvegarde tourne (`running`), toutes les 30 s sinon,
 * tant que la carte est affichée. Une section absente de la réponse = capacité absente du serveur.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { components } from '@/lib/api/generated'
import { DATA_STATUS_FAST_MS, dataStatusRefetchMs } from './dataLogic'

export type DataStatus = components['schemas']['DataStatus']
export type DataBackupStatus = components['schemas']['DataBackupStatus']
export type DataBackupSummary = components['schemas']['DataBackupSummary']
export type MemoryStatus = components['schemas']['MemoryStatus']
export type DataBackupRunResult = components['schemas']['DataBackupRunResponse']

export function useDataStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.dataStatus,
    queryFn: () => api.get<DataStatus>('/settings/data'),
    staleTime: DATA_STATUS_FAST_MS,
    refetchInterval: (query) => dataStatusRefetchMs(query.state.data),
    enabled,
  })
}

/** Lance une sauvegarde tout de suite ; l'état relu juste après dit qu'elle tourne. */
export function useRunDataBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<DataBackupRunResult>('/settings/data/backup/run', {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.dataStatus }),
  })
}
