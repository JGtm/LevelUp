/**
 * useSyncCardLaunch — le lancement manuel de la carte Synchronisation (`POST /players/{slug}/sync`) et la
 * fin du job lancé d'ici : à la fin, le job n'est plus « actif » pour le reste de l'application et l'état
 * du joueur se relit. Un refus (délai, job déjà en vol) se dit par un toast et relit aussi l'état, qui a
 * changé.
 */
import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { apiErrorCode } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useStartDeltaSync } from '@/features/setup/queries'
import type { PlayerSyncStatus } from '@/lib/api/types'
import { jobEnded } from './syncCardLogic'
import type { SyncCardText } from './syncCardText'

interface SyncCardLaunchArgs {
  playerSlug: string
  titleSlug: string
  status: PlayerSyncStatus | undefined
  t: SyncCardText
}

export function useSyncCardLaunch({ playerSlug, titleSlug, status, t }: SyncCardLaunchArgs) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const startSync = useStartDeltaSync()
  const queryClient = useQueryClient()
  const [starting, setStarting] = useState(false)

  const ended = jobEnded(status, activeSyncJobId)
  useEffect(() => {
    if (!ended) return
    setActiveSyncJobId(null)
    void queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })
  }, [ended, playerSlug, titleSlug, queryClient, setActiveSyncJobId])

  async function start() {
    setStarting(true)
    try {
      const started = await startSync.mutateAsync(playerSlug)
      setActiveSyncJobId(started.job_id)
      toast.info(t.toastStarted)
    } catch (err) {
      const code = apiErrorCode(err)
      if (code === 'sync_cooldown') toast.error(t.toastCooldown)
      else if (code === 'sync_already_active') toast.error(t.toastAlreadyActive)
      else toast.error(t.toastFailed)
    } finally {
      await queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })
      setStarting(false)
    }
  }

  return { starting, start }
}
