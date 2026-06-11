/**
 * Queries qualité données — compteurs d'inconnus + listes détaillées.
 * Pas de polling continu : staleTime 60 s + refetch au focus + invalidation
 * après chaque action de résolution.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  AdminDataQualityCounts,
  AdminDataQualityIssues,
  DataQualityIssueKind,
} from '@/lib/api/types'

export function useDataQualityCounts() {
  return useQuery({
    queryKey: queryKeys.adminDataQuality,
    queryFn: () => api.get<AdminDataQualityCounts>('/admin/monitoring/data-quality'),
    staleTime: 60_000,
    retry: false,
  })
}

export function useDataQualityIssues(kind: DataQualityIssueKind, limit = 50) {
  return useQuery({
    queryKey: queryKeys.adminDataQualityIssues(kind),
    queryFn: () =>
      api.get<AdminDataQualityIssues>(
        `/admin/monitoring/data-quality/issues?kind=${kind}&limit=${limit}`,
      ),
    staleTime: 60_000,
    retry: false,
  })
}
