/**
 * timeseriesLeadShare.logic — LA MAINMISE PAR MOIS : une colonne par mois, de la population solo
 * complète. Moyenne des matchs MESURÉS du mois (calculée par le serveur) ; un mois sans match
 * mesuré garde sa colonne en pointillé, jamais un zéro.
 */
import type { LeadSharePeriodPoint } from '@/lib/api/types'
import { formatTemplate, type LeadSharePoint } from '@/lib/charts/leadShare/model'
import type { LeadShareText } from '@/lib/charts/leadShare/text'
import type { Locale } from '@/lib/i18n/locale'
import { intlLocale } from '@/lib/formatters/intlLocale'

/** « janv. 26 » : le mois court et l'année sur deux chiffres, dans la langue du shell. */
export function formatMonthShort(startDate: string, locale: Locale): string {
  const d = new Date(`${startDate}T00:00:00Z`)
  return new Intl.DateTimeFormat(intlLocale(locale), { month: 'short', year: '2-digit', timeZone: 'UTC' }).format(d)
}

export function leadShareMonthPoints(
  months: readonly LeadSharePeriodPoint[] | null | undefined,
  text: LeadShareText,
  locale: Locale,
): LeadSharePoint[] {
  return (months ?? []).map((m) => ({
    key: m.month,
    label: formatMonthShort(m.start_date, locale),
    share: m.lead_share ?? null,
    details: [formatTemplate(text.measuredOfTotal, { measured: m.measured_matches, total: m.matches })],
  }))
}
