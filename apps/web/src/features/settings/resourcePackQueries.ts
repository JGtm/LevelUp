/**
 * TanStack Query — le paquet de ressources de l'application de bureau (capacité `can_update_app`) : état de la
 * préparation (GET /desktop/resources) et nouvel essai (POST /desktop/resources/retry).
 *
 * L'état se relit toutes les 2 s tant qu'une préparation tourne. Une réponse 503 (serveur de développement ou de
 * test, sans empreinte de paquet) n'est pas une erreur à montrer : la carte s'efface.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

/** Étapes d'une préparation (champ `phase`). */
export type ResourcePackPhase =
  | 'idle'
  | 'current'
  | 'waiting_game'
  | 'downloading'
  | 'verifying'
  | 'ready'
  | 'failed'

export interface ResourcePackStatus {
  phase: ResourcePackPhase
  /** Paquet visé par la préparation en cours ou la dernière. */
  fingerprint?: string
  /** Empreinte que le moteur attend. */
  expected?: string
  /** Empreinte du paquet installé. */
  active?: string
  /** Avancement du téléchargement, 0 à 100. */
  percent: number
  /** Code d'erreur fixe de la phase `failed`. */
  code?: string
  /** Un paquet prêt n'est pas encore le paquet actif : il le devient au prochain démarrage. */
  restart_needed: boolean
}

export const RESOURCE_PACK_POLL_MS = 2_000

export function useResourcePackStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.desktopResources,
    queryFn: () => api.get<ResourcePackStatus>('/desktop/resources'),
    enabled,
    retry: false,
    staleTime: RESOURCE_PACK_POLL_MS,
    refetchInterval: (query) => {
      const phase = query.state.data?.phase
      return phase === 'waiting_game' || phase === 'downloading' || phase === 'verifying' ? RESOURCE_PACK_POLL_MS : false
    },
  })
}

/** Relance le téléchargement du paquet du moteur ; la réponse porte déjà l'état, posé dans le cache. */
export function useResourcePackRetry() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<ResourcePackStatus>('/desktop/resources/retry', {}),
    onSuccess: (status) => qc.setQueryData(queryKeys.desktopResources, status),
    onError: () => void qc.invalidateQueries({ queryKey: queryKeys.desktopResources }),
  })
}
