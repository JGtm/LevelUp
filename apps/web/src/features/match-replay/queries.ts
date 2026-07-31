/**
 * queries.ts — lecture de l'artefact de rejeu 2D d'un match.
 * GET /players/{slug}/matches/{matchId}/replay → ReplayDocument (404 si pas d'artefact).
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { ReplayDocument } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

export function useMatchReplay(playerSlug: string, matchId: string) {
  // Le titre courant entre dans la clé comme pour la vue match : deux titres ne
  // partagent jamais un artefact, et changer de titre ne doit pas servir le cache
  // de l'autre.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchReplay(playerSlug, titleSlug, matchId),
    queryFn: () => api.get<ReplayDocument>(`/players/${playerSlug}/matches/${matchId}/replay`),
    staleTime: 5 * 60_000,
    enabled: !!playerSlug && !!matchId,
    // 404 = pas d'artefact de rejeu pour ce match → état vide, ne pas réessayer.
    retry: false,
  })
}
