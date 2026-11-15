/**
 * État de la synchronisation du joueur affiché (GET /players/{slug}/sync/status) : ce qui tourne, quelle
 * qu'en soit la source, le job actif ou le dernier job conservé, le délai restant du sync manuel.
 * La clé porte le joueur : l'accueil d'un joueur ne lit jamais l'état d'un autre.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { PlayerSyncStatus } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

const POLL_RUNNING_MS = 3_000
const POLL_COOLDOWN_MS = 15_000

export function usePlayerSyncStatus(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.playerSyncStatus(playerSlug),
    queryFn: () => api.get<PlayerSyncStatus>(`/players/${playerSlug}/sync/status`),
    enabled: !!playerSlug,
    staleTime: 0,
    // Interroge tant qu'une synchronisation tourne (fin de job), puis tant qu'un délai court (le bouton
    // revient à son terme) ; au repos, une seule lecture.
    refetchInterval: (query) => {
      const status = query.state.data
      if (!status) return false
      if (status.syncing) return POLL_RUNNING_MS
      if (status.cooldown_retry_after_s > 0) return POLL_COOLDOWN_MS
      return false
    },
  })
}
