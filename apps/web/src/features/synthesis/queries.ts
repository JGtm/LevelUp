/**
 * Queries TanStack Query — Synthèse (Slice 7).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SynthesisQueryRequest, SynthesisPageResponse, PeriodInput } from '@/lib/api/types'

export function useSynthesisPage(
  playerSlug: string,
  period: PeriodInput | undefined,
  request: SynthesisQueryRequest,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Le scopeHash doit refléter TOUS les filtres locaux (période ET cascade) pour
  // que React Query refetch après un clic sur "Analyser" qui modifie la cascade.
  const c = request.filters?.cascade
  const cascadeHash = [
    (c?.experience_types ?? []).join(','),
    (c?.playlists ?? []).join(','),
    (c?.modes ?? []).join(','),
    (c?.maps ?? []).join(','),
  ].join('|')
  const scopeHash = `${period?.start_date ?? ''}_${period?.end_date ?? ''}_${cascadeHash}`
  return useQuery({
    queryKey: queryKeys.synthesis(playerSlug, titleSlug, scopeHash),
    queryFn: () =>
      api.post<SynthesisPageResponse>(
        `/players/${playerSlug}/pages/synthesis`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
