/**
 * TanStack Query — régénération des rejeux en retard après une mise à jour (application de bureau,
 * capacité `can_regenerate_replays`) : décompte et état de la file (GET /desktop/replay-regen), démarrage,
 * pause, reprise et recomptage (POST .../start, .../pause, .../resume, .../rescan).
 *
 * L'état se relit toutes les 3 s tant que la file tourne, et toutes les 4 s tant que le serveur n'a pas fini de
 * compter (le décompte lit chaque rejeu rangé, en arrière-plan). Rien ne démarre sans le clic de l'utilisateur.
 *
 * Type écrit à la main : `openapi-gen` décrit le routeur du profil démo, qui ne monte aucune route sous capacité ;
 * il entrera dans `generated.ts` avec le routeur de contrat (liste MANUAL_CAPABILITY_TYPES de
 * lib/api/response-types.guard.test.ts, qui le rappelle dès qu'il y entre).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

export interface ReplayRegenStatus {
  /** Le décompte est fait ; faux tant que `stale` n'est pas connu. */
  checked: boolean
  /** Rejeux en retard et régénérables ; pendant la file : ce qu'il en reste. */
  stale: number
  /** La file est active (en pause comprise). */
  running: boolean
  /** Rejeux traités par la file courante ou la dernière. */
  done: number
  /** Rejeux que la file courante ou la dernière devait traiter ; 0 tant qu'elle prépare sa liste. */
  total: number
  /** En pause à la demande de l'utilisateur. */
  paused: boolean
  /** La file attend la fin de la partie (mode jeu). */
  waiting_for_game: boolean
  /** Rejeux que la file n'a pas pu régénérer. */
  failed: number
  /** La file compresse, à sa fin, les rejeux bruts qu'elle n'a pas pu refaire (film absent de l'archive). */
  compressing: boolean
  /** Rejeux bruts compressés par la file courante ou la dernière. */
  compressed: number
  /** Rejeux bruts que la compression n'a pas pu convertir (ils restent lisibles). */
  compress_failed: number
}

/** Périodes de relecture, en ms. */
export const REPLAY_REGEN_RUNNING_POLL_MS = 3_000
export const REPLAY_REGEN_CHECKING_POLL_MS = 4_000

export function useReplayRegenStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.replayRegen,
    queryFn: () => api.get<ReplayRegenStatus>('/desktop/replay-regen'),
    enabled,
    staleTime: REPLAY_REGEN_RUNNING_POLL_MS,
    refetchInterval: (query) => {
      const data = query.state.data
      if (data?.running) return REPLAY_REGEN_RUNNING_POLL_MS
      return data && !data.checked ? REPLAY_REGEN_CHECKING_POLL_MS : false
    },
  })
}

export type ReplayRegenCommand = 'start' | 'pause' | 'resume' | 'rescan'

/** Envoie une commande à la file ; la réponse porte déjà le nouvel état, posé dans le cache. */
export function useReplayRegenCommand() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (command: ReplayRegenCommand) =>
      api.post<ReplayRegenStatus>(`/desktop/replay-regen/${command}`, {}),
    onSuccess: (status) => qc.setQueryData(queryKeys.replayRegen, status),
    // Une commande refusée (pause sans file) montre l'état réel du serveur plutôt que l'état supposé.
    onError: () => void qc.invalidateQueries({ queryKey: queryKeys.replayRegen }),
  })
}
