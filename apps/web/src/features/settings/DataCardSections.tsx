/**
 * Sections de la carte « Données » (lot L22) : la sauvegarde périodique et le plafond mémoire des
 * bases. Chaque section reçoit l'état de saisie de `useDataSettings` ; le texte d'un refus du serveur
 * s'affiche sous le champ qui l'a causé, et la section ne parle que de ce qu'elle sait.
 */
import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { apiErrorCode } from '@/lib/api/client'
import { formatDateTime, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import type { DataBackupStatus, MemoryStatus } from './dataQueries'
import { useRunDataBackup } from './dataQueries'
import {
  attemptFailureText,
  BACKUP_INTERVALS,
  BACKUP_KEEP_MAX,
  BACKUP_KEEP_MIN,
  errorText,
  knownInterval,
  lastBackupSummary,
} from './dataLogic'
import { fill } from './fillTemplate'
import type { DataText } from './dataText'
import type { DataSettingsForm } from './useDataSettings'

const INPUT_CLASS = 'rounded border border-border bg-background px-2 py-1 text-sm text-foreground'

function FieldError({ message }: { message: string | undefined }) {
  if (!message) return null
  return (
    <p role="alert" className="text-xs text-destructive">
      {message}
    </p>
  )
}

function Field({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      <label className="text-sm text-foreground" htmlFor={id}>
        {label}
      </label>
      {children}
    </div>
  )
}

interface SectionProps {
  t: DataText
  locale: Locale
  form: DataSettingsForm
}

/** Dernier jeu de sauvegarde, et dernier passage s'il a échoué. */
function LastBackup({ t, locale, backup }: { t: DataText; locale: Locale; backup: DataBackupStatus | undefined }) {
  const last = backup?.last
  const attempt = backup?.last_attempt
  return (
    <div className="space-y-1 border-t border-border/50 pt-3 text-sm">
      <p className="text-muted-foreground">{t.lastLabel}</p>
      {last ? (
        <p className="text-foreground" data-testid="data-last-backup">
          {lastBackupSummary(t, locale, last)}
        </p>
      ) : (
        <p className="text-foreground">{t.lastNone}</p>
      )}
      {last?.partial && (
        <p role="status" className="text-xs text-warning">
          {t.lastPartial}
        </p>
      )}
      {attempt && !attempt.ok && (
        <p role="alert" className="text-xs text-destructive">
          {fill(t.lastAttemptFailed, {
            date: formatDateTime(attempt.at, intlLocale(locale)),
            reason: attemptFailureText(t, attempt.error_code),
          })}
        </p>
      )}
    </div>
  )
}

interface BackupSectionProps extends SectionProps {
  backup: DataBackupStatus | undefined
  /** Dossier enregistré : sert tant que l'état du serveur n'est pas arrivé. */
  savedDir: string
}

function BackupDirField({ t, form, same }: { t: DataText; form: DataSettingsForm; same: boolean }) {
  return (
    <Field id="data-backup-dir" label={t.backupDirLabel}>
      <div className="flex gap-2">
        <input
          id="data-backup-dir"
          type="text"
          className={`${INPUT_CLASS} min-w-0 flex-1`}
          value={form.value('backup_dir')}
          placeholder={t.backupDirPlaceholder}
          onChange={(e) => form.edit('backup_dir', e.target.value)}
          onBlur={() => form.commit('backup_dir')}
        />
        <Button
          size="sm"
          variant="outline"
          disabled={!form.dirty('backup_dir') || form.saving}
          onClick={() => form.commit('backup_dir')}
        >
          {t.save}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t.backupDirHint}</p>
      <FieldError message={form.errors.backup_dir} />
      {same && (
        <p role="status" className="text-xs text-warning" data-testid="data-same-volume">
          {t.sameVolumeWarning}
        </p>
      )}
    </Field>
  )
}

function IntervalField({ t, form }: { t: DataText; form: DataSettingsForm }) {
  return (
    <Field id="data-backup-interval" label={t.intervalLabel}>
      <select
        id="data-backup-interval"
        className={INPUT_CLASS}
        value={knownInterval(form.value('backup_interval'))}
        onChange={(e) => form.commit('backup_interval', e.target.value)}
      >
        {BACKUP_INTERVALS.map((interval) => (
          <option key={interval} value={interval}>
            {t.intervals[interval]}
          </option>
        ))}
      </select>
      <FieldError message={form.errors.backup_interval} />
    </Field>
  )
}

function KeepField({ t, form }: { t: DataText; form: DataSettingsForm }) {
  return (
    <Field id="data-backup-keep" label={t.keepLabel}>
      <input
        id="data-backup-keep"
        type="number"
        inputMode="numeric"
        min={BACKUP_KEEP_MIN}
        max={BACKUP_KEEP_MAX}
        className={`${INPUT_CLASS} w-24`}
        value={form.value('backup_keep')}
        onChange={(e) => form.edit('backup_keep', e.target.value)}
        onBlur={() => form.commit('backup_keep')}
      />
      <p className="text-xs text-muted-foreground">{t.keepHint}</p>
      <FieldError message={form.errors.backup_keep} />
    </Field>
  )
}

export function BackupSection({ t, locale, form, backup, savedDir }: BackupSectionProps) {
  const run = useRunDataBackup()
  const enabled = backup ? backup.enabled : savedDir !== ''
  const running = backup?.running === true

  function runNow() {
    form.setGeneralError(undefined)
    run.mutate(undefined, {
      onError: (err) => form.setGeneralError(errorText(t, apiErrorCode(err)) ?? t.saveFailed),
    })
  }

  return (
    <section className="space-y-3" aria-label={t.backupHeading}>
      <h3 className="text-sm font-semibold text-foreground">{t.backupHeading}</h3>
      <BackupDirField t={t} form={form} same={enabled && backup?.same_volume === true} />
      <IntervalField t={t} form={form} />
      <KeepField t={t} form={form} />
      <div className="flex flex-wrap items-center gap-3">
        <Button size="sm" variant="outline" disabled={!enabled || running || run.isPending} onClick={runNow}>
          {running ? t.running : t.runNow}
        </Button>
        {run.isSuccess && !form.errors.general && (
          <p role="status" className="text-xs text-muted-foreground">
            {t.runStarted}
          </p>
        )}
      </div>
      <FieldError message={form.errors.general} />
      <LastBackup t={t} locale={locale} backup={backup} />
    </section>
  )
}

interface MemorySectionProps extends SectionProps {
  memory: MemoryStatus | undefined
}

export function MemorySection({ t, locale, form, memory }: MemorySectionProps) {
  return (
    <section className="space-y-3" aria-label={t.memoryHeading}>
      <h3 className="text-sm font-semibold text-foreground">{t.memoryHeading}</h3>
      {memory && (
        <p className="text-sm text-foreground" data-testid="data-memory-current">
          {memory.rss_mb > 0
            ? fill(t.memoryCurrent, { value: memory.rss_mb.toLocaleString(intlLocale(locale)) })
            : t.memoryUnavailable}
        </p>
      )}
      <Field id="data-memory-limit" label={t.memoryLimitLabel}>
        <input
          id="data-memory-limit"
          type="number"
          inputMode="numeric"
          min={0}
          className={`${INPUT_CLASS} w-32`}
          value={form.value('memory_limit_mb')}
          onChange={(e) => form.edit('memory_limit_mb', e.target.value)}
          onBlur={() => form.commit('memory_limit_mb')}
        />
        <p className="text-xs text-muted-foreground">{t.memoryLimitHint}</p>
        <FieldError message={form.errors.memory_limit_mb} />
      </Field>
      {memory?.effective_limit && (
        <p className="text-xs text-muted-foreground" data-testid="data-memory-effective">
          {fill(t.memoryEffective, { limit: memory.effective_limit })}
        </p>
      )}
    </section>
  )
}
