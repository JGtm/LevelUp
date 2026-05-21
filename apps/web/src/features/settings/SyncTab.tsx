/**
 * SyncTab — onglet "Synchronisation" de SettingsPage.
 *
 * P8.4 (revue 2026-04-29) : extrait de SettingsPage.tsx (~135L).
 */
import { useEffect } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { apiErrorMessage } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import { useStartSyncAll, useJobStatus } from '@/features/setup/queries'
import { useJobToasts } from '@/features/settings/useJobToasts'
import { WatcherSectionBody } from '@/features/settings/WatcherCard'
import { ToggleRow, type TabProps } from './_settingsShared'
import { BackfillCard } from './BackfillCard'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function SyncTab({ merged, handleChange, t }: TabProps) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const setActiveSyncJobId = useAppShellStore((s) => s.setActiveSyncJobId)
  const startSyncAll = useStartSyncAll()
  const { data: jobStatus } = useJobStatus(activeSyncJobId ?? '', !!activeSyncJobId)
  const locale = useAppShellStore((s) => s.locale)
  const tc = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const syncRunning =
    !!activeSyncJobId &&
    jobStatus?.status !== 'succeeded' &&
    jobStatus?.status !== 'failed' &&
    jobStatus?.status !== 'cancelled' &&
    jobStatus?.status !== 'interrupted'

  useEffect(() => {
    if (
      activeSyncJobId &&
      (jobStatus?.status === 'succeeded' ||
        jobStatus?.status === 'failed' ||
        jobStatus?.status === 'cancelled' ||
        jobStatus?.status === 'interrupted')
    ) {
      setActiveSyncJobId(null)
    }
  }, [jobStatus?.status, activeSyncJobId, setActiveSyncJobId])

  useJobToasts(jobStatus, {
    succeeded: t.syncToastSucceeded,
    succeededWithWarnings: t.syncToastSucceededWithWarnings,
    failed: t.syncToastFailed,
    cancelled: t.syncToastCancelled,
  })

  function handleSync() {
    if (syncRunning) return
    startSyncAll.mutate(undefined, {
      onSuccess: (job) => {
        setActiveSyncJobId(job.job_id)
        toast.info(t.syncToastStarted)
      },
      onError: (err) => {
        toast.error(t.syncToastStartFailed, {
          description: apiErrorMessage(err),
        })
      },
    })
  }

  return (
    <>
      {/* Synchronisation manuelle */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.manualSyncTitle}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">{t.manualSyncDescription}</p>
          <Button
            onClick={handleSync}
            disabled={syncRunning || startSyncAll.isPending}
            className="shrink-0"
          >
            {syncRunning ? t.manualSyncRunning : t.manualSyncButton}
          </Button>
        </CardContent>
      </Card>

      {/* Synchronisation automatique (planifiée + watcher fusionnés) */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.autoSyncTitle}</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-border/50">
          {/* Sous-section : Synchronisation planifiée */}
          <div className="pb-4">
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t.schedulerSectionTitle}
            </h3>
            <div className="space-y-1 divide-y divide-border/30">
              <ToggleRow
                label={t.spnkrAutoSync}
                value={merged.spnkr_auto_sync_enabled ?? false}
                onChange={(v) => handleChange('spnkr_auto_sync_enabled', v)}
              />
              <div
                className={`flex items-center justify-between py-3 text-sm ${
                  merged.spnkr_auto_sync_enabled ? '' : 'opacity-40'
                }`}
              >
                <span>{t.spnkrAutoSyncIntervalMinutes}</span>
                <div className="flex items-center gap-1.5">
                  <input
                    type="number"
                    min={5}
                    max={1440}
                    disabled={!merged.spnkr_auto_sync_enabled}
                    className="w-20 rounded border border-border bg-background px-2 py-1 text-right text-sm disabled:cursor-not-allowed"
                    value={merged.spnkr_auto_sync_interval_minutes ?? 360}
                    onChange={(e) => handleChange('spnkr_auto_sync_interval_minutes', parseInt(e.target.value, 10) || 360)}
                  />
                  <span className="text-muted-foreground">{t.spnkrAutoSyncIntervalMinutesUnit}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Sous-section : Détection de présence Xbox (watcher RTA/XSTS) */}
          <div className="pt-4">
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t.watcherSectionTitle}
            </h3>
            <div className="space-y-1 divide-y divide-border/30">
              <ToggleRow
                label={t.watcherPresenceEnabled}
                value={merged.watcher_presence_enabled ?? false}
                onChange={(v) => handleChange('watcher_presence_enabled', v)}
              />
              <div className="pt-1">
                <WatcherSectionBody enabled={merged.watcher_presence_enabled ?? false} t={t} />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Backfill — Recalcul rétroactif */}
      <BackfillCard t={t} />

      {/* Escouade — amis par défaut */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{tc('common.settings.my_squad_default')}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-3">
            {tc('common.settings.squad_intro')}
            Les joueurs déjà configurés dans l'app apparaissent en priorité.
          </p>
          <GamertagCombobox
            selected={(merged.friend_gamertags as string[] | undefined) ?? []}
            onChange={(v) => handleChange('friend_gamertags', v)}
            allowFreeInput={true}
            placeholder={tc('common.settings.gamertag_search_placeholder')}
          />
        </CardContent>
      </Card>
    </>
  )
}
