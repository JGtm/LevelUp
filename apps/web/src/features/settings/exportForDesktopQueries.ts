/**
 * Mutation TanStack Query — jeton d'export vers l'application de bureau (lot L22, site hébergé) :
 * POST /export/desktop/token. Le jeton est servi une fois, à usage unique, valable 15 minutes ; il
 * reste dans l'état de la mutation, le temps de l'affichage, et n'entre dans aucune clé ni stockage.
 *
 * Type écrit à la main : `openapi-gen` décrit le routeur du profil démo, qui ne monte aucune route sous capacité ;
 * ce type n'entrera dans `generated.ts` qu'une fois le routeur de contrat séparé de celui de démonstration (liste
 * MANUAL_CAPABILITY_TYPES de lib/api/response-types.guard.test.ts, qui le rappelle dès qu'il y entre).
 */
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'

export interface ExportToken {
  token: string
  /** Instant d'échéance du jeton (15 minutes après sa création). */
  expires_at: string
}

export function useGenerateExportToken() {
  return useMutation({
    mutationFn: () => api.post<ExportToken>('/export/desktop/token', {}),
  })
}
