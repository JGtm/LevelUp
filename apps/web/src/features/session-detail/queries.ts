import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { SessionPageRequest, SessionPageResponse } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

export function useSessionDetailPage(
  playerSlug: string,
  request: SessionPageRequest,
  filterHash: string,
  sessionLabel: string,
  compareSessionLabel: string,
  enableCompare: boolean,
  locale: string,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.sessionDetail(
      playerSlug,
      titleSlug,
      filterHash,
      sessionLabel,
      compareSessionLabel,
      enableCompare,
      locale,
    ),
    queryFn: () =>
      api.post<SessionPageResponse>(
        `/players/${playerSlug}/pages/sessions/detail`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    // Garde les donnees de la requete precedente pendant le fetch de la nouvelle
    // cle (toggle/changement compare) : sans ca, data passe a undefined et le
    // garde `if (isLoading)` de la page remplace tout par un spinner plein ecran,
    // ce qui demonte/remonte le layout et empeche l'animation du drawer compare.
    placeholderData: keepPreviousData,
  })
}
