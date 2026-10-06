/**
 * tendancesWinLoss.logic — les lignes PURES du graphique « Moyenne par match en défaite et en
 * victoire » : du bloc `win_loss` de l'horizon courant aux lignes de l'haltère.
 *
 * Axe commun : l'écart-type (`z_loss` pour le point des défaites, `z_win` pour celui des
 * victoires). Les valeurs RÉELLES s'écrivent près des points, formatées par indicateur
 * (`WIN_LOSS_FORMATS`) ; le coefficient r et le nombre de matchs passent dans l'infobulle.
 */
import type { TrendsPageResponse, TrendsWinLossBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { MIN_COMPARE_MATCHES, formatTrendDelta, formatTrendValue } from './tendances.logic'
import type { DumbbellRow } from './tendancesDumbbell.logic'

/** Unité et décimales d'une valeur réelle affichée près d'un point. */
export interface WinLossFormat {
  unit: string
  decimals: number
}

/** Format des indicateurs qui n'ont pas de ligne dans `WIN_LOSS_FORMATS`. */
const DEFAULT_FORMAT: WinLossFormat = { unit: 'number', decimals: 1 }

/** Formats par indicateur (clés de l'API). */
export const WIN_LOSS_FORMATS: Readonly<Record<string, WinLossFormat>> = {
  accuracy: { unit: 'ratio', decimals: 3 },
  avg_life_seconds: { unit: 'seconds', decimals: 1 },
  kda: { unit: 'number', decimals: 2 },
  offensive_conversion: { unit: 'number', decimals: 2 },
  defensive_resistance: { unit: 'number', decimals: 2 },
  damage_balance: { unit: 'number', decimals: 0 },
  mmr_gap: { unit: 'number', decimals: 0 },
}

/** Format d'une valeur réelle de l'indicateur `key`. */
export function winLossFormat(key: string): WinLossFormat {
  return WIN_LOSS_FORMATS[key] ?? DEFAULT_FORMAT
}

/** Le bloc de l'horizon, ou `undefined` s'il manque. */
export function winLossBlock(
  data: Pick<TrendsPageResponse, 'win_loss'>,
  horizon: number,
): TrendsWinLossBlock | undefined {
  return (data.win_loss ?? []).find((b) => b.days === horizon)
}

/** Ce que l'état vide dit : le nombre de victoires et défaites, et celui qu'il en faut. */
export function winLossShortfall(block: TrendsWinLossBlock | undefined): {
  n: number
  required: number
} {
  return {
    n: block?.matches ?? 0,
    required: block && block.required > 0 ? block.required : MIN_COMPARE_MATCHES,
  }
}

/** Les lignes de l'haltère, dans l'ordre reçu. Vide quand le bloc est absent ou sans ligne. */
export function buildWinLossRows(
  block: TrendsWinLossBlock | undefined,
  locale: Locale,
  labelOf: (key: string) => string,
): DumbbellRow[] {
  const t = getTendancesText(locale)
  return (block?.rows ?? []).map((row) => {
    const format = winLossFormat(row.key)
    const loss = formatTrendValue(row.loss_mean, format.unit, format.decimals, locale)
    const win = formatTrendValue(row.win_mean, format.unit, format.decimals, locale)
    return {
      label: labelOf(row.key),
      a: row.z_loss,
      b: row.z_win,
      aText: loss,
      bText: win,
      tooltip: [
        t.winLossTooltipLoss(loss),
        t.winLossTooltipWin(win),
        t.winLossTooltipR(formatTrendDelta(row.r, 'number', 2, locale)),
        t.matchCount(row.matches),
      ],
    }
  })
}
