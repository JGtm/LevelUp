/**
 * Libellés calendaires localisés (jours de semaine, heures) + textes d'axes
 * pour les charts temporels (heatmaps activité, timeseries par jour).
 *
 * SOURCE UNIQUE (CLAUDE.md n°6) : les tableaux `['Lun', 'Mar', ...]` étaient
 * dupliqués dans 4 fichiers (explorer/synthesis heatmaps, timeseries adapters,
 * lab showcase) — un seul en FR, cassant le bilinguisme (règle n°1). Centralisé
 * ici ; garde-rail `no_hardcoded_dow_labels` (test grep) interdit le littéral
 * ailleurs. Toute nouvelle heatmap/axe jour consomme `dowLabels(locale)`.
 */
import type { ManifestLocale } from '@/lib/i18n/format'

export const DOW_LABELS_FR = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'] as const
export const DOW_LABELS_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const

/** Abréviations des 7 jours (lundi-first, aligné sur l'index `dow` backend). */
export function dowLabels(locale: ManifestLocale): readonly string[] {
  return locale === 'en' ? DOW_LABELS_EN : DOW_LABELS_FR
}

/** Étiquettes des 24 heures ("00h".."23h") — notation 24 h commune FR/EN. */
export const HOUR_LABELS: readonly string[] = Array.from(
  { length: 24 },
  (_, h) => `${String(h).padStart(2, '0')}h`,
)

export interface CalendarChartText {
  hourAxis: string
  dayAxis: string
  winRate: string
  wins: string
  matches: string
  commonMatches: string
  noCommonMatch: string
}

const CALENDAR_CHART_TEXT: Record<ManifestLocale, CalendarChartText> = {
  fr: {
    hourAxis: 'Heure',
    dayAxis: 'Jour',
    winRate: 'Taux de victoire',
    wins: 'Victoires',
    matches: 'Matchs',
    commonMatches: 'Matchs communs',
    noCommonMatch: 'Aucun match commun',
  },
  en: {
    hourAxis: 'Hour',
    dayAxis: 'Day',
    winRate: 'Win rate',
    wins: 'Wins',
    matches: 'Matches',
    commonMatches: 'Common matches',
    noCommonMatch: 'No common match',
  },
}

/** Textes localisés des axes + tooltips des heatmaps activité (heure × jour). */
export function calendarChartText(locale: ManifestLocale): CalendarChartText {
  return CALENDAR_CHART_TEXT[locale] ?? CALENDAR_CHART_TEXT.fr
}
