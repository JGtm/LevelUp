/**
 * Mutation du diagnostic apparence Spartan ID (volet 2, Lot G).
 *
 * Le diagnostic est un GET porté par une MUTATION (déclenché impérativement au
 * clic « Diagnostiquer ») : aucune query automatique, aucun refetch au focus —
 * le calcul serveur (fetch live Halo, budget 8 s) ne doit partir que sur action
 * explicite de l'opérateur.
 */
import { useMutation } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AppearanceDiagnosisResponse } from '@/lib/api/types'

/** Diagnostique l'apparence d'un joueur suivi : `mutate(playerSlug)`. */
export function useAppearanceDiag() {
  return useMutation({
    mutationKey: queryKeys.adminAppearanceDiagMutation,
    mutationFn: (playerSlug: string) =>
      api.get<AppearanceDiagnosisResponse>(
        `/admin/diag/appearance/${encodeURIComponent(playerSlug)}`,
      ),
  })
}
