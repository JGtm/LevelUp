/**
 * tendancesMedals.logic — les lignes PURES du graphique « Médailles par match » : du bloc
 * `medals` de l'horizon courant aux lignes de l'haltère.
 *
 * Point B = taux de l'horizon ; point A = taux de la période d'avant, seulement quand le bloc
 * est comparé. Dans un bloc comparé, un `prev_rate` absent est un taux d'avant NUL (l'API
 * omet le zéro) : le point A tombe à 0 et la variation se dit « nouvelle ».
 */
import type { TrendsMedalsBlock, TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { formatTrendValue } from './tendances.logic'
import type { DumbbellRow } from './tendancesDumbbell.logic'

/** Décimales d'un taux par match. */
const RATE_DECIMALS = 2

/** Le bloc de l'horizon, ou `undefined` s'il manque. */
export function medalsBlock(
  data: Pick<TrendsPageResponse, 'medals'>,
  horizon: number,
): TrendsMedalsBlock | undefined {
  return (data.medals ?? []).find((b) => b.days === horizon)
}

/** Variation en pourcentage, signée ; « nouvelle » quand le taux d'avant est nul. */
function variationText(rate: number, prev: number, locale: Locale): string {
  const t = getTendancesText(locale)
  if (!(prev > 0)) return t.medalsNew
  const pct = Math.round(((rate - prev) / prev) * 100)
  const sign = pct > 0 ? '+' : pct < 0 ? '\u2212' : ''
  return `${sign}${formatTrendValue(Math.abs(pct) / 100, 'ratio', 2, locale)}`
}

/** Les lignes de l'haltère, dans l'ordre reçu (déjà les dix retenues par l'API). */
export function buildMedalRows(
  block: TrendsMedalsBlock | undefined,
  locale: Locale,
  names: { current: string; previous: string },
): DumbbellRow[] {
  const t = getTendancesText(locale)
  const compared = block?.compared === true
  return (block?.rows ?? []).map((row) => {
    const rate = formatTrendValue(row.rate, 'number', RATE_DECIMALS, locale)
    const prev = compared ? (row.prev_rate ?? 0) : null
    const tooltip = [t.medalsTooltipCurrent(names.current, rate)]
    if (prev != null) {
      tooltip.push(
        t.medalsTooltipCurrent(
          names.previous,
          formatTrendValue(prev, 'number', RATE_DECIMALS, locale),
        ),
        t.tooltipVariation(variationText(row.rate, prev, locale)),
      )
    }
    return { label: row.name, a: prev, b: row.rate, bText: rate, tooltip }
  })
}
