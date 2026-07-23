/**
 * BackupTab — contenu de la section « Sauvegarde » (monté par AdminBackupSection,
 * page admin Système ; le titre de section vit hors carte). Affiche le statut de la
 * dernière sauvegarde restic des bases DuckDB et permet de déclencher une sauvegarde
 * manuelle. La PLANIFICATION est externe (systemd timers côté serveur) : l'application
 * n'embarque plus de planificateur.
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { useBackupStatus, useRunBackup } from '@/features/settings/queries'
import type { getSettingsText } from '@/features/settings/i18n'
import type { IntegrityResult } from '@/lib/api/types'

interface BackupTabProps {
  t: ReturnType<typeof getSettingsText>
  /** Mode démo : sauvegarde figée (bouton désactivé), cf. SettingsPage. */
  frozen?: boolean
}

function statusBadge(enabled: boolean, available: boolean, t: BackupTabProps['t']) {
  // Non configurées (état neutre) : les sauvegardes ne sont pas activées. « Restic
  // introuvable » (erreur) est réservé au cas où elles SONT activées mais que le
  // binaire manque (enabled && !available) — cf. B3.
  if (!enabled) return { label: t.backupStatusNotConfigured, cls: 'bg-muted text-muted-foreground' }
  if (!available) return { label: t.backupStatusResticMissing, cls: 'bg-destructive/20 text-destructive' }
  return { label: t.backupStatusEnabled, cls: 'bg-green-500/20 text-green-700 dark:text-green-400' } // color-allow: badge état système success (vert) — exception CLAUDE.md §20
}

function IntegrityBadge({ result }: { result: IntegrityResult }) {
  if (result.ok) {
    return (
      <span className="rounded bg-green-500/15 px-1.5 py-0.5 text-xs font-mono text-green-700 dark:text-green-400"> {/* color-allow: badge état système success — exception CLAUDE.md §20 */}
        ✓
      </span>
    )
  }
  return (
    <span
      title={result.detail}
      className="cursor-help rounded bg-amber-500/15 px-1.5 py-0.5 text-xs font-mono text-amber-700 dark:text-amber-400" /* color-allow: badge état système warning — exception CLAUDE.md §20 */
    >
      ⚠
    </span>
  )
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

export function BackupTab({ t, frozen }: BackupTabProps) {
  const { data: status, isLoading } = useBackupStatus()
  const runBackup = useRunBackup()

  const badge = status ? statusBadge(status.enabled, status.available, t) : null

  const lastResult = runBackup.data
  let runFeedback: string | null = null
  if (runBackup.isPending) runFeedback = t.backupRunning
  else if (runBackup.isError) runFeedback = t.backupRunError
  else if (lastResult) runFeedback = lastResult.skipped ? t.backupRunSkipped : t.backupRunDone

  return (
    // En démo, la sauvegarde manuelle est figée (cf. SyncTab) : <fieldset disabled>
    // neutralise le bouton « Lancer ». Le statut reste lisible. Le titre de la
    // section (« Sauvegarde ») est porté hors carte par SectionHeader côté admin :
    // une seule carte compacte statut + configuration + action.
    <fieldset
      disabled={frozen}
      className={`m-0 min-w-0 border-0 p-0 ${frozen ? 'opacity-60' : ''}`}
    >
      <Card>
        <CardContent className="space-y-4 pt-6">
          {/* Statut + badge d'état */}
          <div className="flex items-start justify-between gap-3">
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

                {status.integrity_checks && Object.keys(status.integrity_checks).length > 0 && (
                  <>
                    <dt className="text-muted-foreground">{t.backupIntegrityLabel}</dt>
                    <dd className="flex flex-wrap gap-1">
                      {Object.entries(status.integrity_checks).map(([key, result]) => (
                        <span key={key} className="flex items-center gap-1">
                          <span className="text-xs text-muted-foreground font-mono">{key}</span>
                          <IntegrityBadge result={result} />
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
            {badge && (
              <span
                className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${badge.cls}`}
              >
                {badge.label}
              </span>
            )}
          </div>

          {/* CTA d'activation — état neutre « non configurées » : explique le geste
              concret (binaire restic + BACKUP_ENABLED + dépôt restic + doc). */}
          {status && !status.enabled && (
            <div className="rounded-md border border-border bg-muted/40 p-3 text-xs leading-relaxed text-muted-foreground">
              {t.backupNotConfiguredHint}
            </div>
          )}

          {/* Configuration (lecture seule) — repliée dans la même carte */}
          {status?.config && (
            <div className="space-y-1 border-t pt-3">
              <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t.backupConfigTitle}
              </div>
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
            </div>
          )}

          {/* Action */}
          <div className="flex items-center gap-3 border-t pt-3">
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
    </fieldset>
  )
}
