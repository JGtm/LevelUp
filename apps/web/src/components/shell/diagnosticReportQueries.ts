/**
 * Mutation TanStack Query — rapport de diagnostic de l'application de bureau (lot L22) :
 * POST /desktop/diagnostic crée l'archive des journaux et de l'état (sans secret) et rend son chemin.
 *
 * Type écrit à la main : `openapi-gen` décrit le routeur du profil démo, qui ne monte aucune route sous capacité ;
 * ce type n'entrera dans `generated.ts` qu'une fois le routeur de contrat séparé de celui de démonstration (liste
 * MANUAL_CAPABILITY_TYPES de lib/api/response-types.guard.test.ts, qui le rappelle dès qu'il y entre).
 */
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'

export interface DiagnosticReport {
  /** Chemin complet de l'archive, à joindre à un signalement. */
  path: string
  file_name: string
  /** Taille de l'archive, en octets. */
  bytes: number
}

export function useDiagnosticReport() {
  return useMutation({
    mutationFn: () => api.post<DiagnosticReport>('/desktop/diagnostic', {}),
  })
}
