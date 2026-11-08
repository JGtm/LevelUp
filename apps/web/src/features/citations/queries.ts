/**
 * Queries TanStack Query — Citations (Slice 2B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CitationsPageResponse } from '@/lib/api/types'

/** Totaux de carrière : la requête ne porte aucun filtre (corps de requête absent). */
export function useCitationsPage(playerSlug: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.citations(playerSlug, titleSlug, locale),
    queryFn: () => api.post<CitationsPageResponse>(`/players/${playerSlug}/pages/citations`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
