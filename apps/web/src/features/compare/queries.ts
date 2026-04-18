/**
 * queries.ts — hooks TanStack Query pour Compare joueur vs joueur.
 * Sprint 54-C.
 */
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
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
