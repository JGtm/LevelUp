/**
 * Queries du viewer de logs — modules disponibles + tail filtré.
 * keepPreviousData : pas de flash au changement de filtre ; auto-refresh
 * opt-in 5 s (off par défaut — un tail lit le disque côté serveur).
 */
import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AdminLogModules, AdminLogTail } from '@/lib/api/types'

export function useLogModules() {
  return useQuery({
    queryKey: queryKeys.adminLogModules,
    queryFn: () => api.get<AdminLogModules>('/admin/monitoring/logs/modules'),
    staleTime: 60_000,
    retry: false,
  })
}

export interface LogTailParams {
  module: string
  level: string // '' = tous
  contains: string
  limit: number
}

export function useLogTail(params: LogTailParams, autoRefresh: boolean) {
  const { module, level, contains, limit } = params
  const qs = new URLSearchParams({ module, n: String(limit) })
  if (level) qs.set('level', level)
  if (contains) qs.set('contains', contains)
  return useQuery({
    queryKey: queryKeys.adminLogTail(module, level, contains, limit),
    queryFn: () => api.get<AdminLogTail>(`/admin/monitoring/logs/tail?${qs.toString()}`),
    enabled: !!module,
    placeholderData: keepPreviousData,
    refetchInterval: autoRefresh ? 5_000 : false,
    staleTime: 4_000,
    retry: false,
  })
}
