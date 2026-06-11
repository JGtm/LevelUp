/**
 * Mutations des actions correctives du dashboard monitoring.
 *
 * Sémantique HTTP :
 * - data-health/run : synchrone, 200 + compteurs.
 * - auto-sync/run : 202 + AsyncJobStatus à suivre via useJobStatus ; un 409
 *   renvoie le job DÉJÀ en vol — le client http lève une ApiError, le caller
 *   la traite comme « suivre l'existant » via runningJobFromConflict.
 */
import { useMutation } from '@tanstack/react-query'

import { api, apiErrorCode } from '@/lib/api/client'
import type { AsyncJobStatus, MonitoringDataHealth } from '@/lib/api/types'

export function useRunDataHealthCheck() {
  return useMutation({
    mutationFn: () => api.post<MonitoringDataHealth>('/admin/actions/data-health/run', {}),
  })
}

export function useRunSyncCycle() {
  return useMutation({
    mutationFn: () => api.post<AsyncJobStatus>('/admin/actions/auto-sync/run', {}),
  })
}

/**
 * Extrait details.job_id d'une ApiError 409 `already_running` — le backend y
 * renvoie le job déjà en vol pour que le front le suive au lieu d'en créer
 * un doublon.
 */
export function conflictJobId(err: unknown): string | null {
  if (apiErrorCode(err) !== 'already_running') return null
  const details = (err as { details?: unknown }).details
  if (details && typeof details === 'object' && 'job_id' in details) {
    const id = (details as { job_id?: unknown }).job_id
    if (typeof id === 'string' && id) return id
  }
  return null
}
