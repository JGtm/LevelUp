/**
 * Shared helpers + UI atoms des panneaux ex-Lab encore en service.
 *
 * A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — ne restent ici que
 * les briques consommées par DiagnosticsPanel (rendu dans l'onglet admin
 * Données) : formatters + StatusBadge + MetricCard + FileStatusRow + GuardRow.
 * RouteList / JsonViewer / SelectableAssetList / SelectableMedalList sont
 * partis avec les panneaux Resources/Waypoint/Contracts.
 */
import { Badge } from '@/components/ui/badge'
import type { LabFileStatus, LabGuardResult } from '@/lib/api/types'
import { getLabText, type LabText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'

// ─── Formatters ──────────────────────────────────────────────────────────────

export function getStatusVariant(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'ok' || normalized === 'passed' || normalized === 'present') {
    return 'success' as const
  }
  if (
    normalized === 'failed'
    || normalized === 'ko'
    || normalized === 'divergence'
    || normalized === 'divergences'
    || normalized === 'absent'
  ) {
    return 'destructive' as const
  }
  return 'secondary' as const
}

export function translateStatus(status: string, text: LabText) {
  switch (status.toLowerCase()) {
    case 'ok':
      return text.common.statuses.ok
    case 'ko':
      return text.common.statuses.ko
    case 'passed':
      return text.common.statuses.passed
    case 'failed':
      return text.common.statuses.failed
    case 'skipped':
      return text.common.statuses.skipped
    case 'divergence':
    case 'divergences':
      return text.common.statuses.divergence
    default:
      return status
  }
}

export function formatLabDateTime(value: string | null | undefined, locale: Locale, text: LabText) {
  if (!value) {
    return text.common.notAvailable
  }
  return new Date(value).toLocaleString(getLabText(locale).intlLocale)
}

export function formatNumber(value: number | null | undefined, locale: Locale, text: LabText) {
  if (value == null) {
    return text.common.notAvailable
  }
  return value.toLocaleString(getLabText(locale).intlLocale)
}

export function formatDecimal(value: number, locale: Locale) {
  return value.toLocaleString(getLabText(locale).intlLocale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
}

export function formatBytes(value: number | null | undefined, locale: Locale, text: LabText) {
  if (value == null) {
    return text.common.notAvailable
  }
  if (value < 1024) {
    return `${value} B`
  }
  if (value < 1024 * 1024) {
    return `${formatDecimal(value / 1024, locale)} KB`
  }
  return `${formatDecimal(value / (1024 * 1024), locale)} MB`
}

// ─── UI atoms ────────────────────────────────────────────────────────────────

export function StatusBadge({ status, text }: { status: string; text: LabText }) {
  return <Badge variant={getStatusVariant(status)}>{translateStatus(status, text)}</Badge>
}

// P8.13 (revue 2026-04-29) : wrapper léger autour de StatCard variant='metric'.
// Conserve le nom MetricCard pour la rétrocompat des imports lab/.
import { StatCard } from '@/components/cards/StatCard'
export function MetricCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return <StatCard label={label} value={value} hint={hint} variant="metric" />
}

export function FileStatusRow({
  label,
  file,
  locale,
  text,
}: {
  label: string
  file: LabFileStatus
  locale: Locale
  text: LabText
}) {
  return (
    <div className="rounded-xl border border-border bg-muted p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">{label}</p>
          <p className="mt-1 break-all text-xs text-muted-foreground">{file.path}</p>
        </div>
        <Badge variant={file.exists ? 'success' : 'destructive'}>
          {file.exists ? text.common.present : text.common.absent}
        </Badge>
      </div>
      <div className="mt-3 flex flex-wrap gap-4 text-xs text-muted-foreground">
        <span>{text.common.size}: {formatBytes(file.size_bytes, locale, text)}</span>
        <span>{text.common.modified}: {formatLabDateTime(file.modified_at ?? null, locale, text)}</span>
      </div>
    </div>
  )
}

export function GuardRow({
  label,
  result,
  text,
}: {
  label: string
  result: LabGuardResult
  text: LabText
}) {
  return (
    <div className="rounded-xl border border-border p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-foreground">{label}</p>
        <Badge variant={result.passed ? 'success' : 'destructive'}>
          {result.passed ? text.common.statuses.ok : text.common.statuses.ko}
        </Badge>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{result.reason}</p>
      {(result.details ?? []).length > 0 ? (
        <div className="mt-2 space-y-1 text-xs text-muted-foreground">
          {(result.details ?? []).slice(0, 5).map((detail) => (
            <p key={detail}>{detail}</p>
          ))}
        </div>
      ) : null}
    </div>
  )
}
