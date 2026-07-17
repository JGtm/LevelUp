/**
 * Barrel export des helpers de formatage canoniques (revue 2026-04-29 P2.6 + P2.6bis).
 *
 * Usage :
 *   import { formatPercent, formatDate, formatNumber, formatDurationMMSS } from '@/lib/formatters'
 *
 * Convention : tous les helpers acceptent `null | undefined` et renvoient un
 * fallback "—" (ou "-" pour les durées) configurable. Pas de coercition
 * silencieuse (`Number.isNaN` + null check explicites).
 */

export { formatPercent, formatPercentValue, formatPercentInt } from './percent'
export { formatDate, formatDateRange, formatDateShort, formatDateTime, type Locale } from './date'
export { formatNumber, formatNumberFixed, formatRatio, formatKDA } from './number'
export { formatDurationMMSS, formatDurationHMS, formatDurationMinSec, formatDurationMShort } from './duration'
export { displayRatingLabel, formatRankDelta } from './rating'
export { formatOffensiveConversion, formatDefensiveResistance, effectiveDmgPerFrag } from './combatYield'
export {
  dowLabels,
  DOW_LABELS_FR,
  DOW_LABELS_EN,
  HOUR_LABELS,
  calendarChartText,
} from './calendar'
export { intlLocale } from './intlLocale'
export { verificationLinkLabel } from './url'
