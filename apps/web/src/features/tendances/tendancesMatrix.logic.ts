/**
 * tendancesMatrix.logic — le builder PUR de la matrice « Indicateurs par mois et par
 * horizon » : de la réponse de l'API aux séries du wrapper `Heatmap2DChart`, une grille par
 * groupe d'indicateurs.
 *
 * COLONNES : les douze mois (nom court dans la locale), puis les horizons du plus long au
 * plus court (365, 90, 30, 7 j). LIGNES : un indicateur `in_matrix` par ligne.
 *
 * LA COULEUR ET LE NOMBRE NE DISENT PAS LA MÊME CHOSE. `value` (la couleur) est l'écart-type
 * borné à ± 2,5 ; `detail.count` (le nombre écrit dans la case) est la valeur de l'indicateur,
 * déjà formatée. Une valeur sans écart-type (horizon sans comparaison) prend 0 : case
 * neutre, que l'infobulle explique. Une cellule sans valeur prend `null`.
 *
 * TOUTES LES CASES SONT ÉMISES, dans l'ordre des colonnes puis des lignes : le wrapper déduit
 * ses axes de l'ordre d'apparition des points, en omettre une décalerait les catégories.
 */
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { escapeHtml } from '@/components/charts/_utils'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import type { TrendsHorizonCell, TrendsIndicator, TrendsMonthCell } from '@/lib/api/types'

import { MATRIX_GROUPS, getTendancesText, type MatrixGroup } from './i18n'
import {
  MIN_COMPARE_MATCHES,
  clampZ,
  formatTrendDelta,
  formatTrendValue,
} from './tendances.logic'

/** Horizons de la matrice, dans l'ordre des colonnes. */
const MATRIX_HORIZONS = [365, 90, 30, 7] as const

/** Hauteur d'une ligne de la grille, en pixels. */
const ROW_HEIGHT_PX = 34
/** Marges haute et basse de la grille (axe des colonnes compris), en pixels. */
const GRID_VERTICAL_PX = 64
/** Largeur d'un caractère d'étiquette de ligne, en pixels. */
const LABEL_CHAR_PX = 6.5
/** Bornes de la marge gauche (étiquettes de lignes), en pixels. */
const LEFT_MIN_PX = 120
const LEFT_MAX_PX = 260

/** Ce que l'infobulle d'une case sait d'elle ; `count` est lu par le wrapper pour l'étiquette. */
export interface MatrixCellDetail extends Record<string, unknown> {
  /** Valeur formatée, écrite dans la case. */
  count: string
  kind: 'month' | 'horizon'
  label: string
  period: string
  unit: string
  decimals: number
  value: number
  matches: number
  prevValue?: number
  prevMatches?: number
  /** Vrai quand la période d'avant existe et que la case porte un écart-type. */
  compared: boolean
}

export interface MatrixGrid {
  group: MatrixGroup
  /** Nombre de lignes (hauteur proportionnelle). */
  rows: number
  series: ChartSeries<ChartPointHeatmap>[]
}

/** Nom court du mois d'une clé `AAAA-MM`, dans la locale. */
export function monthShortLabel(monthKey: string, locale: Locale): string {
  return monthFormat(monthKey, locale, { month: 'short' })
}

function monthFormat(
  monthKey: string,
  locale: Locale,
  options: Intl.DateTimeFormatOptions,
): string {
  const [annee, mois] = monthKey.split('-').map(Number)
  const date = new Date(Date.UTC(annee, mois - 1, 1))
  return new Intl.DateTimeFormat(intlLocale(locale), { ...options, timeZone: 'UTC' }).format(date)
}

/** Valeur de la couleur d'une case : écart-type borné, 0 sans écart, `null` sans valeur. */
function colorValue(value: number | undefined, z: number | undefined): number | null {
  if (value == null) return null
  return z == null ? 0 : clampZ(z)
}

function monthPoint(
  ind: TrendsIndicator,
  cell: TrendsMonthCell | undefined,
  monthKey: string,
  label: string,
  locale: Locale,
): ChartPointHeatmap {
  const x = monthShortLabel(monthKey, locale)
  const value = cell?.value
  if (cell == null || value == null) return { x, y: label, value: null }
  const detail: MatrixCellDetail = {
    count: formatTrendValue(value, ind.unit, ind.decimals, locale),
    kind: 'month',
    label,
    period: monthFormat(monthKey, locale, { month: 'long', year: 'numeric' }),
    unit: ind.unit,
    decimals: ind.decimals,
    value,
    matches: cell.matches,
    compared: false,
  }
  return { x, y: label, value: colorValue(value, cell.z), detail }
}

function horizonPoint(
  ind: TrendsIndicator,
  cell: TrendsHorizonCell | undefined,
  days: number,
  label: string,
  locale: Locale,
): ChartPointHeatmap {
  const t = getTendancesText(locale)
  const x = t.horizonShort(days)
  const value = cell?.value
  if (cell == null || value == null) return { x, y: label, value: null }
  const detail: MatrixCellDetail = {
    count: formatTrendValue(value, ind.unit, ind.decimals, locale),
    kind: 'horizon',
    label,
    period: t.horizonLong(days),
    unit: ind.unit,
    decimals: ind.decimals,
    value,
    matches: cell.matches,
    prevValue: cell.prev_value,
    prevMatches: cell.prev_matches,
    compared: cell.z != null && cell.prev_value != null,
  }
  return { x, y: label, value: colorValue(value, cell.z), detail }
}

/**
 * Les grilles de la matrice : une par groupe présent (ordre `level`, `results`, `combat`,
 * `style`, `objectives`, `activity`), limitées aux indicateurs `in_matrix`.
 * `labelOf` écrit la ligne d'un indicateur (clé et variante).
 */
export function buildMatrixGrids(
  indicators: readonly TrendsIndicator[] | null | undefined,
  months: readonly string[] | null | undefined,
  locale: Locale,
  labelOf: (key: string, variant?: string) => string,
): MatrixGrid[] {
  const cles = months ?? []
  const grids: MatrixGrid[] = []
  for (const group of MATRIX_GROUPS) {
    const lignes = (indicators ?? []).filter((i) => i.in_matrix && i.group === group)
    if (lignes.length === 0) continue
    const libelles = lignes.map((i) => labelOf(i.key, i.variant))
    const datapoints: ChartPointHeatmap[] = []
    cles.forEach((monthKey, idx) => {
      lignes.forEach((ind, r) => {
        datapoints.push(monthPoint(ind, ind.months?.[idx], monthKey, libelles[r], locale))
      })
    })
    for (const days of MATRIX_HORIZONS) {
      lignes.forEach((ind, r) => {
        const cell = ind.horizons?.find((h) => h.days === days)
        datapoints.push(horizonPoint(ind, cell, days, libelles[r], locale))
      })
    }
    grids.push({ group, rows: lignes.length, series: [{ key: `trends-${group}`, datapoints }] })
  }
  return grids
}

/** Hauteur d'une grille, proportionnelle à son nombre de lignes. */
export function matrixGridHeight(rows: number): number {
  return rows * ROW_HEIGHT_PX + GRID_VERTICAL_PX
}

/** Marge gauche COMMUNE à toutes les grilles : la plus longue étiquette de lignes, bornée. */
export function matrixLeftMargin(grids: readonly MatrixGrid[]): number {
  let plusLongue = 0
  for (const g of grids) {
    for (const p of g.series[0]?.datapoints ?? []) plusLongue = Math.max(plusLongue, p.y.length)
  }
  return Math.min(LEFT_MAX_PX, Math.max(LEFT_MIN_PX, Math.round(plusLongue * LABEL_CHAR_PX + 16)))
}

/** Les lignes de l'infobulle d'une case, texte échappé. */
export function formatMatrixTooltip(point: ChartPointHeatmap, locale: Locale): string {
  const d = point.detail as MatrixCellDetail | undefined
  if (!d) return ''
  const t = getTendancesText(locale)
  const lignes = [
    `<b>${escapeHtml(t.tooltipHeader(d.label, d.period))}</b>`,
    escapeHtml(t.tooltipValue(d.count, d.matches)),
  ]
  if (d.kind === 'horizon') {
    const prevMatches = d.prevMatches ?? 0
    if (d.compared && d.prevValue != null) {
      lignes.push(
        escapeHtml(
          t.tooltipPrevious(
            formatTrendValue(d.prevValue, d.unit, d.decimals, locale),
            prevMatches,
          ),
        ),
        escapeHtml(
          t.tooltipVariation(formatTrendDelta(d.value - d.prevValue, d.unit, d.decimals, locale)),
        ),
      )
    } else if (prevMatches < MIN_COMPARE_MATCHES) {
      lignes.push(escapeHtml(t.tooltipNoComparison(prevMatches, MIN_COMPARE_MATCHES)))
    } else {
      lignes.push(escapeHtml(t.tooltipNoComparisonCurrent(d.matches, MIN_COMPARE_MATCHES)))
    }
  }
  return lignes.join('<br/>')
}
