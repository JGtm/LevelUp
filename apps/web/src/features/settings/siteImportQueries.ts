/**
 * TanStack Query — import des données depuis l'instance hébergée (lot L22) : démarrage
 * (POST /import/site) et progression (GET /import/site).
 *
 * Le jeton d'export ne sert qu'au corps de la requête : il n'entre dans aucune clé de requête, aucune
 * URL, aucun stockage du navigateur. La progression se relit toutes les 2 s tant que l'import tourne.
 *
 * Types écrits à la main : `generated.ts` ne porte pas encore les schémas SiteImportStatus et
 * SiteImportRequest ; à remplacer par ces ré-exports du contrat dès la régénération des types.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

export type SiteImportPhase = 'idle' | 'downloading' | 'importing' | 'done' | 'failed'

export interface SiteImportStatus {
  phase: SiteImportPhase
  /** Code fixe de l'échec, quand la phase est `failed`. */
  error_code?: string
  /** Lignes insérées par table, quand l'import est terminé. */
  inserted?: Record<string, number>
  /** Matchs ajoutés. */
  matches: number
}

export interface SiteImportRequest {
  url: string
  token: string
  /** Joueur suivi qui reçoit l'import ; nécessaire seulement quand plusieurs joueurs sont suivis. */
  gamertag?: string
}

/** Période de relecture de la progression tant que l'import tourne. */
export const SITE_IMPORT_POLL_MS = 2_000

/** Vrai tant que l'import travaille (téléchargement ou écriture). */
export function isImportRunning(phase: SiteImportPhase | undefined): boolean {
  return phase === 'downloading' || phase === 'importing'
}

export function useSiteImportStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.siteImport,
    queryFn: () => api.get<SiteImportStatus>('/import/site'),
    refetchInterval: (query) => (isImportRunning(query.state.data?.phase) ? SITE_IMPORT_POLL_MS : false),
    enabled,
  })
}

/** Démarre l'import ; la réponse (202) porte déjà l'état initial, posé dans le cache de progression. */
export function useStartSiteImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: SiteImportRequest) => api.post<SiteImportStatus>('/import/site', req),
    onSuccess: (status) => qc.setQueryData(queryKeys.siteImport, status),
  })
}
