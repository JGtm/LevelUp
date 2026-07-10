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
  AdminDetectionsResponse,
  AdminFreshnessResponse,
  AdminResourcesResponse,
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

/**
 * Détections persistées avec cycle de vie (open/acked/muted/resolved) — vue
 * detections_latest de la base monitoring, survit au restart. Remplace l'ancien
 * panneau « erreurs récurrentes » mémoire (perdu au reboot). On récupère toutes
 * les détections (plafond serveur) et le filtrage statut/niveau/module se fait
 * côté table (TanStack) — pas de refetch par filtre.
 */
export function useMonitoringDetections() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringDetections,
    queryFn: () => api.get<AdminDetectionsResponse>('/admin/monitoring/detections'),
    // Le GET flush d'abord le collecteur mémoire côté serveur : cadence tranquille.
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: false,
  })
}

/** Un weapon_id non résolu + son volume de kills (id en string : > 2^53). */
export interface AdminWeaponCoverageItem {
  weapon_id: string
  kills: number
}

/** Couverture de la résolution d'arme d'un titre (diagnostic registre/labels). */
export interface AdminWeaponCoverage {
  title_slug: string
  generated_at: string
  distinct_weapons: number
  resolved_registry: number
  resolved_label: number
  unresolved: number
  coverage_percent: number
  registry_percent: number
  top_unresolved: AdminWeaponCoverageItem[]
}

/**
 * Couverture de résolution d'arme pour un titre donné. Le slug est passé en
 * query (?title=) — autoritaire, indépendant du titre courant : un seul panneau
 * affiche tous les titres côte à côte.
 */
export function useWeaponCoverage(titleSlug: string) {
  return useQuery({
    queryKey: queryKeys.adminWeaponCoverage(titleSlug),
    queryFn: () =>
      api.get<AdminWeaponCoverage>(
        `/admin/monitoring/weapon-coverage?title=${encodeURIComponent(titleSlug)}`,
      ),
    enabled: !!titleSlug,
    // Référentiel + agrégat de kills : pas de polling, cache tranquille.
    staleTime: 60_000,
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

/**
 * Fraîcheur des données par joueur suivi et par titre actif (A4, DC-3).
 * Lectures shared par joueur côté Go : pas de polling agressif — staleTime
 * 60 s + refetch au focus (pattern convergence). Le calcul pose aussi la gauge
 * freshness_critical lue par l'overview (badge État).
 */
export function useMonitoringFreshness() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringFreshness,
    queryFn: () => api.get<AdminFreshnessResponse>('/admin/monitoring/freshness'),
    staleTime: 60_000,
    retry: false,
  })
}

/**
 * Ressources machine & process (A5) : runtime Go, tailles DB + WAL, disque
 * libre, budgets/pool DuckDB, restarts. os.Stat + expvar côté Go (pas de
 * lecture DuckDB) — polling tranquille aligné sur l'overview.
 */
export function useMonitoringResources() {
  return useQuery({
    queryKey: queryKeys.adminMonitoringResources,
    queryFn: () => api.get<AdminResourcesResponse>('/admin/monitoring/resources'),
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: false,
  })
}
