/**
 * BackupTab — onglet "Sauvegarde" de SettingsPage.
 * Affiche le statut du scheduler DuckDB/Restic et permet de déclencher
 * une sauvegarde manuelle.
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useBackupStatus, useRunBackup } from '@/features/settings/queries'
import type { getSettingsText } from '@/features/settings/i18n'

interface BackupTabProps {
  t: ReturnType<typeof getSettingsText>
}

function statusBadge(enabled: boolean, available: boolean, t: BackupTabProps['t']) {
  if (!enabled) return { label: t.backupStatusDisabled, cls: 'bg-muted text-muted-foreground' }
  if (!available) return { label: t.backupStatusResticMissing, cls: 'bg-destructive/20 text-destructive' }
  return { label: t.backupStatusEnabled, cls: 'bg-green-500/20 text-green-700 dark:text-green-400' }
}

function fmtDuration(ms: number) {
  if (ms < 1000) return `${ms} ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)} s`
  return `${Math.round(ms / 60_000)} min`
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export function BackupTab({ t }: BackupTabProps) {
  const { data: status, isLoading } = useBackupStatus()
  const runBackup = useRunBackup()

  const badge = status ? statusBadge(status.enabled, status.available, t) : null

  const lastResult = runBackup.data
  let runFeedback: string | null = null
  if (runBackup.isPending) runFeedback = t.backupRunning
  else if (runBackup.isError) runFeedback = t.backupRunError
  else if (lastResult) runFeedback = lastResult.skipped ? t.backupRunSkipped : t.backupRunDone

  return (
    <>
      {/* Statut */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-base">{t.backupTitle}</CardTitle>
          {badge && (
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${badge.cls}`}>
              {badge.label}
            </span>
          )}
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <p className="text-sm text-muted-foreground">{t.backupNever}</p>
          ) : status?.last_backup_at ? (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt className="text-muted-foreground">{t.backupLastBackup}</dt>
              <dd>{fmtDate(status.last_backup_at)}</dd>

              {status.last_snapshot_id && (
                <>
                  <dt className="text-muted-foreground">{t.backupSnapshotId}</dt>
                  <dd className="font-mono text-xs">{status.last_snapshot_id.slice(0, 12)}</dd>
                </>
              )}

              {status.last_exported && status.last_exported.length > 0 && (
                <>
                  <dt className="text-muted-foreground">{t.backupDatabases}</dt>
                  <dd className="flex flex-wrap gap-1">
                    {status.last_exported.map((key) => (
                      <span key={key} className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                        {key}
                      </span>
                    ))}
                  </dd>
                </>
              )}

              {status.last_duration_ms != null && (
                <>
                  <dt className="text-muted-foreground">{t.backupDuration}</dt>
                  <dd>{fmtDuration(status.last_duration_ms)}</dd>
                </>
              )}
            </dl>
          ) : (
            <p className="text-sm text-muted-foreground">{t.backupNever}</p>
          )}

          <div className="flex items-center gap-3 pt-1">
            <Button
              size="sm"
              disabled={runBackup.isPending || !status?.available}
              onClick={() => runBackup.mutate()}
            >
              {runBackup.isPending ? t.backupRunning : t.backupRunButton}
            </Button>
            {runFeedback && !runBackup.isPending && (
              <span
                className={`text-xs ${runBackup.isError ? 'text-destructive' : 'text-muted-foreground'}`}
              >
                {runFeedback}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Configuration (lecture seule) */}
      {status?.config && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">{t.backupConfigTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt className="text-muted-foreground">{t.backupConfigInterval}</dt>
              <dd className="font-mono">{status.config.interval}</dd>
              <dt className="text-muted-foreground">{t.backupConfigRetention}</dt>
              <dd>
                {t.backupConfigRetentionValue
                  .replace('{daily}', String(status.config.keep_daily))
                  .replace('{weekly}', String(status.config.keep_weekly))
                  .replace('{monthly}', String(status.config.keep_monthly))}
              </dd>
            </dl>
          </CardContent>
        </Card>
      )}
    </>
  )
}
