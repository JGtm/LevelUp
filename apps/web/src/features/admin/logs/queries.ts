/**
 * Queries du viewer de logs — modules disponibles + tail filtré paginé.
 * keepPreviousData : pas de flash au changement de filtre ; auto-refresh
 * opt-in 5 s (off par défaut — un tail lit le disque côté serveur).
 *
 * Le tail est une requête INFINIE : la première page lit depuis la fin du
 * fichier, « Charger plus » (fetchNextPage) enchaîne les tranches plus anciennes
 * via le curseur arrière `before=next_offset` (aucun recouvrement, franchit le
 * budget 8 Mio page par page).
 */
import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AdminLogModules, AdminLogTail } from '@/lib/api/types'
import { nextLogCursor } from './logsCursor'

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
  return useInfiniteQuery({
    queryKey: queryKeys.adminLogTail(module, level, contains, limit),
    queryFn: ({ pageParam }) => {
      const qs = new URLSearchParams({ module, n: String(limit) })
      if (level) qs.set('level', level)
      if (contains) qs.set('contains', contains)
      if (pageParam) qs.set('before', String(pageParam)) // 0 = première page (depuis la fin)
      return api.get<AdminLogTail>(`/admin/monitoring/logs/tail?${qs.toString()}`)
    },
    initialPageParam: 0,
    getNextPageParam: nextLogCursor,
    enabled: !!module,
    placeholderData: keepPreviousData,
    refetchInterval: autoRefresh ? 5_000 : false,
    staleTime: 4_000,
    retry: false,
  })
}
