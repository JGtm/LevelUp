/**
 * État de la synchronisation du joueur affiché (GET /players/{slug}/sync/status) : ce qui tourne, quelle
 * qu'en soit la source, le job actif ou le dernier job conservé, le délai restant du sync manuel.
 * La clé porte le joueur : l'accueil d'un joueur ne lit jamais l'état d'un autre.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import type { PlayerSyncStatus } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

const POLL_RUNNING_MS = 3_000
const POLL_COOLDOWN_MS = 15_000

/**
 * Délai avant la prochaine lecture de l'état : 3 s tant qu'une synchronisation tourne (fin de job), 15 s
 * tant qu'un délai court (le bouton revient à son terme) ; au repos, rien, sauf `pollIdle` (15 s : l'écran
 * qui propose un bouton voit une synchronisation lancée ailleurs, par l'automatique ou depuis l'accueil).
 */
export function syncStatusPollInterval(status: PlayerSyncStatus | undefined, pollIdle: boolean): number | false {
  if (!status) return false
  if (status.syncing) return POLL_RUNNING_MS
  if (status.cooldown_retry_after_s > 0 || pollIdle) return POLL_COOLDOWN_MS
  return false
}

export function usePlayerSyncStatus(playerSlug: string, options: { pollIdle?: boolean } = {}) {
  const pollIdle = options.pollIdle === true
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug),
    queryFn: () => api.get<PlayerSyncStatus>(`/players/${playerSlug}/sync/status`),
    enabled: !!playerSlug,
    staleTime: 0,
    refetchInterval: (query) => syncStatusPollInterval(query.state.data, pollIdle),
  })
}
