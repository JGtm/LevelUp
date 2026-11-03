/**
 * Mutation TanStack Query — rapport de diagnostic de l'application de bureau (lot L22) :
 * POST /desktop/diagnostic crée l'archive des journaux et de l'état (sans secret) et rend son chemin.
 *
 * Type écrit à la main : `generated.ts` ne porte pas encore le schéma DiagnosticReport ; à remplacer par
 * ce ré-export du contrat dès la régénération des types.
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
