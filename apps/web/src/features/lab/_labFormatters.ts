/**
 * Formatters partagés des panneaux ex-Lab — extraits de _labShared.tsx pour que
 * ce dernier n'exporte que des composants (react-refresh/only-export-components).
 */
import { getLabText, type LabLocale, type LabText } from './i18n'

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

export function formatLabDateTime(value: string | null | undefined, locale: LabLocale, text: LabText) {
  if (!value) {
    return text.common.notAvailable
  }
  return new Date(value).toLocaleString(getLabText(locale).intlLocale)
}

export function formatNumber(value: number | null | undefined, locale: LabLocale, text: LabText) {
  if (value == null) {
    return text.common.notAvailable
  }
  return value.toLocaleString(getLabText(locale).intlLocale)
}

export function formatDecimal(value: number, locale: LabLocale) {
  return value.toLocaleString(getLabText(locale).intlLocale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
}

export function formatBytes(value: number | null | undefined, locale: LabLocale, text: LabText) {
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
