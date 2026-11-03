/**
 * Mutation TanStack Query — jeton d'export vers l'application de bureau (lot L22, site hébergé) :
 * POST /export/desktop/token. Le jeton est servi une fois, à usage unique, valable 15 minutes ; il
 * reste dans l'état de la mutation, le temps de l'affichage, et n'entre dans aucune clé ni stockage.
 *
 * Type écrit à la main : `generated.ts` ne porte pas encore le schéma ExportToken ; à remplacer par
 * ce ré-export du contrat dès la régénération des types.
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
