/**
 * HomeFirstSync — accueil d'un joueur sans aucun match synchronisé.
 *
 * Dit l'état réel de la synchronisation (en cours, en attente de connexion, en pause, échec,
 * terminée sans match, rien de lancé) au lieu d'annoncer « en cours » par défaut. L'état se lit
 * dans `firstSyncState.ts` ; chaque bloc ne propose que des actions possibles (reconnecter, lancer,
 * réessayer, actualiser), jamais un bouton que le serveur refuserait.
 */
import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useReconnectXbox } from '@/lib/auth/useReconnectXbox'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { useJobStatus, useStartDeltaSync } from '@/features/setup/queries'
import { syncJobErrorKey } from '@/features/setup/syncJobError'
import { resolveFirstSyncState } from './firstSyncState'

interface HomeFirstSyncProps {
  playerSlug: string
  /** Relit l'accueil (matchs arrivés, ou simple actualisation demandée). */
  refetch: () => void
}

export function HomeFirstSync({ playerSlug, refetch }: HomeFirstSyncProps) {
  const locale = useAppShellStore((s) => s.locale)
  const reauthRequired = useAppShellStore((s) => s.reauthRequired)
  const authState = useAppShellStore((s) => s.authState)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const canWrite = useCanWritePlayer(playerSlug)
  const reconnect = useReconnectXbox()
  const startSync = useStartDeltaSync()
  const { data: job } = useJobStatus(activeSyncJobId ?? '', !!activeSyncJobId)
  const t = (key: HomeManifestKey) => formatMessage(homeManifest, key, locale)

  const { kind } = resolveFirstSyncState({
    reauthRequired,
    authState,
    syncEnabled: currentPlayer?.player_slug === playerSlug ? currentPlayer.sync_enabled : undefined,
    activeJobId: activeSyncJobId,
    jobStatus: job?.status,
  })

  // Un job qui vient de se terminer a pu écrire des matchs : l'accueil se relit une fois par job.
  const refetchedFor = useRef<string | null>(null)
  useEffect(() => {
    if (job?.status === 'succeeded' && activeSyncJobId && refetchedFor.current !== activeSyncJobId) {
      refetchedFor.current = activeSyncJobId
      refetch()
    }
  }, [job?.status, activeSyncJobId, refetch])

  function startNow() {
    startSync.mutate(playerSlug, {
      onSuccess: (started) => setActiveSyncJobId(started.job_id),
      onError: () => toast.error(t('home.first_sync.start_failed')),
    })
  }

  const block = (() => {
    switch (kind) {
      case 'waiting_connection':
        return {
          title: t('home.first_sync.waiting_title'),
          description: t('home.first_sync.waiting_description'),
          actionLabel: t('home.first_sync.waiting_action'),
          onAction: reconnect,
        }
      case 'paused':
        return {
          title: t('home.first_sync.paused_title'),
          description: t('home.first_sync.paused_description'),
        }
      case 'running':
        return {
          title: t('home.first_sync.title'),
          description: t('home.first_sync.description'),
          actionLabel: t('home.first_sync.action'),
          onAction: refetch,
        }
      case 'failed': {
        const cause = formatMessage(
          commonManifest,
          syncJobErrorKey(job?.error?.code ?? 'sync_aborted') ?? 'common.initial_sync.err_internal',
          locale,
        )
        const retry = canWrite && job?.error?.retryable !== false
        return {
          title: t('home.first_sync.failed_title'),
          description: cause,
          actionLabel: retry ? t('home.first_sync.retry_action') : undefined,
          onAction: retry ? startNow : undefined,
        }
      }
      case 'empty':
        return {
          title: t('home.first_sync.empty_title'),
          description: t('home.first_sync.empty_description'),
          actionLabel: t('home.first_sync.action'),
          onAction: refetch,
        }
      case 'idle':
        return {
          title: t('home.first_sync.idle_title'),
          description: canWrite
            ? t('home.first_sync.idle_description')
            : t('home.first_sync.idle_description_readonly'),
          actionLabel: canWrite ? t('home.first_sync.start_action') : undefined,
          onAction: canWrite ? startNow : undefined,
        }
    }
  })()

  return (
    <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
      <div className="mx-auto w-full max-w-lg">
        <EmptyStateCard {...block} />
      </div>
    </div>
  )
}
