/**
 * queries.ts — lecture de l'artefact de rejeu 2D d'un match.
 * GET /players/{slug}/matches/{matchId}/replay → ReplayDocument (404 si pas d'artefact).
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { ReplayDocument } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export function useMatchReplay(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchReplay(playerSlug, matchId),
    queryFn: () => api.get<ReplayDocument>(`/players/${playerSlug}/matches/${matchId}/replay`),
    staleTime: 5 * 60_000,
    enabled: !!playerSlug && !!matchId,
    // 404 = pas d'artefact de rejeu pour ce match → état vide, ne pas réessayer.
    retry: false,
  })
}
