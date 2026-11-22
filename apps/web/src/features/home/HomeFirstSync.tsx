/**
 * HomeFirstSync — accueil d'un joueur sans aucun match synchronisé.
 *
 * Dit l'état réel de la synchronisation (en cours, en attente de connexion, en pause, échec,
 * terminée sans match, rien de lancé) au lieu d'annoncer « en cours » par défaut. L'état se lit
 * dans `firstSyncState.ts` à partir de ce que le SERVEUR dit du joueur affiché
 * (`GET /players/{slug}/sync/status`) ; chaque bloc ne propose que des actions possibles
 * (reconnecter, lancer, réessayer, actualiser), jamais un bouton que le serveur refuserait : pendant
 * le délai du sync manuel, ni « Lancer » ni « Réessayer », et le délai restant est dit.
 */
import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { apiErrorCode } from '@/lib/api/client'
import { useReconnectXbox } from '@/lib/auth/useReconnectXbox'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { commonManifest } from '@/lib/i18n/generated/common'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useStartDeltaSync } from '@/features/setup/queries'
import { syncJobErrorKey } from '@/features/setup/syncJobError'
import { resolveFirstSyncState } from './firstSyncState'
import { usePlayerSyncStatus } from '@/lib/players/playerSyncStatus'

interface HomeFirstSyncProps {
  playerSlug: string
  /** Relit l'accueil (matchs arrivés, ou simple actualisation demandée). */
  refetch: () => void
}

export function HomeFirstSync({ playerSlug, refetch }: HomeFirstSyncProps) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const reauthRequired = useAppShellStore((s) => s.reauthRequired)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const canWrite = useCanWritePlayer(playerSlug)
  const reconnect = useReconnectXbox()
  const startSync = useStartDeltaSync()
  const queryClient = useQueryClient()
  const { data: serverStatus, isError: statusFailed, refetch: refetchStatus } = usePlayerSyncStatus(playerSlug)
  const ownHaloXuid = useAppShellStore((s) => s.ownHaloXuid)
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)

  // Le serveur ne sert que les jobs du joueur demandé ; le client ne lit pourtant jamais le job d'un autre.
  const status = serverStatus?.player_slug === playerSlug ? serverStatus : undefined
  // Une panne, ou la réponse d'un autre joueur : l'écran le dit au lieu de rester vide.
  const statusError = (statusFailed && !serverStatus) || (serverStatus !== undefined && status === undefined)
  const displayed = currentPlayer?.player_slug === playerSlug ? currentPlayer : undefined
  const job = status?.job && (!status.job.player_slug || status.job.player_slug === playerSlug) ? status.job : undefined
  const cooldownSec = status?.cooldown_retry_after_s ?? 0
  const canStart = canWrite && cooldownSec <= 0

  const { kind } = resolveFirstSyncState({
    reauthRequired,
    canWrite,
    isOwnAccount: ownHaloXuid !== '' && displayed?.xuid === ownHaloXuid,
    syncEnabled: currentPlayer?.player_slug === playerSlug ? currentPlayer.sync_enabled : undefined,
    server: status ? { syncing: status.syncing, jobStatus: job?.status } : null,
    statusError,
  })

  // Un job qui vient de se terminer a pu écrire des matchs : l'accueil se relit une fois par job.
  const refetchedFor = useRef<string | null>(null)
  useEffect(() => {
    if (job?.status === 'succeeded' && refetchedFor.current !== job.job_id) {
      refetchedFor.current = job.job_id
      refetch()
    }
  }, [job?.status, job?.job_id, refetch])

  function startNow() {
    startSync.mutate(playerSlug, {
      onSuccess: (started) => {
        setActiveSyncJobId(started.job_id)
        void queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })
      },
      onError: (err) => {
        const code = apiErrorCode(err)
        if (code === 'sync_cooldown') toast.error(t('home.first_sync.start_cooldown'))
        else if (code === 'sync_already_active') toast.error(t('home.first_sync.start_already_active'))
        else toast.error(t('home.first_sync.start_failed'))
        // Le refus dit que l'état affiché est périmé : le relire montre l'état réel (délai, en cours).
        void queryClient.invalidateQueries({ queryKey: queryKeys.playerSyncStatus(playerSlug, titleSlug) })
      },
    })
  }

  const cooldownNote =
    canWrite && cooldownSec > 0
      ? ` ${t('home.first_sync.cooldown_note', { minutes: Math.ceil(cooldownSec / 60) })}`
      : ''

  const block = (() => {
    switch (kind) {
      case 'loading':
        return null
      case 'status_error':
        return {
          title: t('home.first_sync.status_error_title'),
          description: t('home.first_sync.status_error_description'),
          actionLabel: t('home.first_sync.retry_action'),
          onAction: () => void refetchStatus(),
        }
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
        const retry = canStart && job?.error?.retryable !== false
        return {
          title: t('home.first_sync.failed_title'),
          description: `${cause}${cooldownNote}`,
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
            ? `${t('home.first_sync.idle_description')}${cooldownNote}`
            : t('home.first_sync.idle_description_readonly'),
          actionLabel: canStart ? t('home.first_sync.start_action') : undefined,
          onAction: canStart ? startNow : undefined,
        }
    }
  })()

  if (block === null) return null

  return (
    <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
      <div className="mx-auto w-full max-w-lg">
        <EmptyStateCard {...block} />
      </div>
    </div>
  )
}
