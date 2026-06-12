/**
 * Queries du dashboard monitoring admin.
 *
 * Polling différencié :
 * - overview : 30 s (zéro I/O DuckDB côté Go — états mémoire uniquement)
 * - scheduler : 30 s, accéléré à 5 s quand un cycle forcé est en vol
 * - jobs : 5 s si au moins un job actif, sinon 30 s
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  AdminConvergenceReport,
  AdminErrorStats,
  AdminJobsResponse,
  AdminMonitoringOverview,
  AdminPerfStats,
  AdminSchedulerStatusResponse,
} from '@/lib/api/types'

export function useMonitoringOverview() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringOverview,
    queryFn: () => api.get<AdminMonitoringOverview>('/admin/monitoring/overview'),
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: false,
  })
}

export function useMonitoringScheduler(opts?: { fastPoll?: boolean }) {
  const fast = opts?.fastPoll ?? false
  return useQuery({
    queryKey: queryKeys.adminMonitoringScheduler,
    queryFn: () => api.get<AdminSchedulerStatusResponse>('/admin/monitoring/scheduler'),
    refetchInterval: fast ? 5_000 : 30_000,
    staleTime: 4_000,
    retry: false,
  })
}

export function useMonitoringConvergence() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringConvergence,
    queryFn: () => api.get<AdminConvergenceReport>('/admin/monitoring/convergence'),
    // Résout les DBs de tous les joueurs : pas de polling continu.
    staleTime: 60_000,
    retry: false,
  })
}

export function usePerfStats() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringPerf,
    queryFn: () => api.get<AdminPerfStats>('/admin/monitoring/perf'),
    // Agrégats expvar purs (zéro I/O serveur) — polling tranquille.
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: false,
  })
}

export function useMonitoringErrors() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringErrors,
    queryFn: () => api.get<AdminErrorStats>('/admin/monitoring/errors'),
    // Collecteur mémoire (zéro I/O) — polling tranquille aligné sur l'overview.
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: false,
  })
}

export function useAdminJobs(limit = 20) {
  return useQuery({
    queryKey: queryKeys.adminMonitoringJobs,
    queryFn: () => api.get<AdminJobsResponse>(`/admin/monitoring/jobs?limit=${limit}`),
    // Cadence pilotée par l'activité : un job actif → 5 s, sinon 30 s.
    refetchInterval: (query) =>
      (query.state.data?.jobs ?? []).some(
        (j) => j.status === 'running' || j.status === 'queued',
      )
        ? 5_000
        : 30_000,
    staleTime: 4_000,
    retry: false,
  })
}
