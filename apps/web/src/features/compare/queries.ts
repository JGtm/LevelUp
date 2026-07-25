/**
 * queries.ts — hooks TanStack Query pour Compare joueur vs joueur.
 * Sprint 54-C.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CompareRequest, CompareResponse } from '@/lib/api/types'

/**
 * useCompare — soumet une comparaison POST /players/{slug}/pages/compare.
 *
 * Utilise useMutation car l'endpoint est POST et la comparaison est
 * déclenchée à la demande (pas un fetch automatique).
 */
export function useCompare(playerSlug: string) {
  return useMutation<CompareResponse, Error, CompareRequest>({
    mutationFn: (req: CompareRequest) =>
      api.post<CompareResponse>(`/players/${playerSlug}/pages/compare`, req),
  })
}

/**
 * useComparePrefetch — retourne une fonction qui prefetch la comparaison
 * pour un gamertag cible. À déclencher sur onMouseEnter ou onSelect (C3.2).
 */
export function useComparePrefetch(playerSlug: string) {
  const queryClient = useQueryClient()

  return function prefetchCompare(targetGamertag: string) {
    void queryClient.prefetchQuery({
      queryKey: queryKeys.comparePlayer(
        playerSlug,
        useAppShellStore.getState().currentTitleSlug,
        targetGamertag,
      ),
      queryFn: () =>
        api.post<CompareResponse>(`/players/${playerSlug}/pages/compare`, {
          target_gamertag: targetGamertag,
        }),
      staleTime: 2 * 60 * 1000,
    })
  }
}
