/**
 * StepInitialSync — étape 3 du wizard SetupPage : synchronisation initiale.
 *
 * P8.4 (revue 2026-04-29) : extrait de SetupPage.tsx (~160L).
 */
import { useEffect } from 'react'
import { Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { queryKeys } from '@/lib/query/keys'
import { useStartInitialSync, useJobStatus } from './queries'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

interface StepInitialSyncProps {
  playerSlug: string
}

export function StepInitialSync({ playerSlug }: StepInitialSyncProps) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const currentJobId = useSetupFlowStore((s) => s.currentJobId)
  const setCurrentJobId = useSetupFlowStore((s) => s.setCurrentJobId)
  const startSync = useStartInitialSync()
  const queryClient = useQueryClient()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Reprendre depuis le job actif connu (session serveur) ou le job local du store
  const resolvedJobId = activeSyncJobId ?? currentJobId

  const { data: job } = useJobStatus(resolvedJobId ?? '', !!resolvedJobId)

  function handleStart() {
    startSync.mutate(
      { player_slug: playerSlug, max_matches: 200 },
      {
        onSuccess: (j) => {
          setCurrentJobId(j.job_id)
          queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
        },
      },
    )
  }

  // Quand la sync réussit : invalider le bootstrap pour passer à "ready"
  useEffect(() => {
    if (job?.status === 'succeeded') {
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
    }
  }, [job?.status, queryClient])

  const errorMessages: Record<string, string> = {
    sync_auth_expired: "Votre session Halo a expiré. Relancez pour renouveler l'authentification.",
    sync_halo_api_error: "L'API Halo est temporairement indisponible. Veuillez réessayer.",
    sync_db_error: "Erreur interne lors de l'enregistrement des données. Contactez le support.",
    sync_aborted: "La synchronisation a été interrompue. Vous pouvez relancer.",
    internal_error: "Erreur inattendue. Veuillez réessayer.",
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t('common.setup.initial_sync_title')}</h2>
      <p className="text-sm text-muted-foreground">
        {t('common.setup.initial_sync_intro')}
      </p>
      <p className="text-xs text-muted-foreground">
        {t('common.setup.session_rules_note')}{' '}
        <Link to="/settings" search={{ tab: 'analyse' }} className="underline hover:text-foreground">
          {t('common.setup.settings_analysis')}
        </Link>{' '}
        {t('common.setup.after_sync_suffix')}
      </p>

      {!resolvedJobId && (
        <Button onClick={handleStart} disabled={startSync.isPending}>
          {startSync.isPending ? 'Démarrage…' : 'Lancer la synchronisation'}
        </Button>
      )}

      {job && (
        <div className="space-y-3">
          {/* Barre de progression */}
          {job.progress_pct != null && (
            <div className="space-y-1">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{job.phase_label ?? job.current_step ?? '…'}</span>
                <span>{job.progress_pct} %</span>
              </div>
              <div className="h-2 w-full rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary transition-all duration-500"
                  style={{ width: `${job.progress_pct}%` }}
                />
              </div>
            </div>
          )}

          {/* Compteurs métier */}
          {job.matches_done != null && job.matches_total != null && (
            <p className="text-sm text-muted-foreground">
              {job.matches_done} / {job.matches_total} {t('common.setup.matches_retrieved')}
            </p>
          )}

          {/* ETA */}
          {job.eta_seconds != null && job.status === 'running' && (
            <p className="text-xs text-muted-foreground">
              {t('common.setup.time_remaining_about')} {Math.ceil(job.eta_seconds / 60)} min
            </p>
          )}

          {/* Warnings */}
          {job.warnings.length > 0 && (
            <ul className="text-xs text-warning space-y-0.5">
              {job.warnings.map((w) => <li key={w}>⚠️ {w}</li>)}
            </ul>
          )}

          {/* Résultat réussi */}
          {job.status === 'succeeded' && (
            <div className="space-y-2">
              <p className="text-success font-medium">
                {t('common.setup.sync_complete')}
                {job.result?.matches_imported != null && (
                  <span className="font-normal text-muted-foreground">
                    {' '}{Number(job.result.matches_imported)} {t('common.setup.matches_imported')}
                  </span>
                )}
              </p>
              <Button onClick={() => queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })}>
                {t('common.setup.open_app')}
              </Button>
            </div>
          )}

          {/* Interrompu (redémarrage serveur) */}
          {job.status === 'interrupted' && (
            <div className="space-y-2">
              <p className="text-warning font-medium">
                {t('common.setup.sync_interrupted')}
              </p>
              <Button
                variant="outline"
                onClick={() => {
                  setCurrentJobId(null)
                  handleStart()
                }}
              >
                {t('common.setup.resume_sync')}
              </Button>
            </div>
          )}

          {/* Erreur */}
          {job.status === 'failed' && (
            <div className="space-y-2">
              <p className="text-destructive font-medium">{t('common.setup.sync_failed')}</p>
              {job.error?.code && (
                <p className="text-sm text-muted-foreground">
                  {errorMessages[job.error.code] ?? job.error.message}
                </p>
              )}
              {job.error?.retryable !== false && (
                <Button
                  variant="outline"
                  onClick={() => {
                    setCurrentJobId(null)
                    handleStart()
                  }}
                >
                  Réessayer
                </Button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
