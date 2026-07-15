/**
 * _utils.ts — Helpers partagés des wrappers ECharts Squad V2.
 *
 * Conformément au PLAN_META_FOUNDATIONS_GO § 5.2 : tous les wrappers
 * (BarStacked, BarGrouped, TimeseriesLine, Heatmap2D, Radar) consomment
 * `ChartSeries<T>[]` (mirror du domain.ChartSeries[T] Go) et reposent
 * sur les helpers ci-dessous pour cohérence visuelle.
 *
 * Couleurs : aucun rgba(255,255,255,X) en dur. Tout passe par
 * `getEChartsThemeColors()` qui résout les CSS vars sémantiques au
 * runtime (light/dark) — voir `apps/web/src/lib/echarts/themeColors.ts`.
 *
 * Pattern d'usage côté builder :
 *   const tc = getEChartsThemeColors()
 *   return {
 *     tooltip: getTooltipBase(tc),
 *     legend:  getLegendBase(tc),
 *     xAxis:   getCategoricalXAxis(cats, tc),
 *     yAxis:   { ...getAxisBase(tc), type: 'value' },
 *     ...
 *   }
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatNumberFixed } from '@/lib/formatters'

export const CHART_BG = 'transparent'

/**
 * escapeHtml — échappe les métacaractères HTML avant interpolation dans un
 * tooltip ECharts (rendu en innerHTML par défaut, sans sanitisation). OBLIGATOIRE
 * sur toute donnée NON CONSTANTE interpolée dans un formatter de tooltip (noms de
 * cartes UGC, gamertags, labels d'assets) — sinon XSS stocké (audit sécurité
 * 2026-07 #9). Source unique : ne jamais redéfinir localement (garde-rail
 * escapeHtml.test.ts).
 */
export function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/** Base axis style (axes X/Y). À spread avant les overrides spécifiques. */
export function getAxisBase(tc: EChartsThemeColors) {
  return {
    axisLine: { lineStyle: { color: tc.axisLine } },
    axisTick: { show: false },
    splitLine: { lineStyle: { color: tc.splitLine } },
    axisLabel: { color: tc.axisLabel, fontSize: 10 },
  } as const
}

/**
 * Socle `grid` ECharts commun (marges + containLabel) pour les charts timeseries.
 * `overrides` ajuste ponctuellement une marge (ex: axe secondaire plus large).
 * Factorisé H6 (2026-07-04) — remplace ~8 littéraux grid identiques/quasi-identiques.
 */
export function getGridBase(overrides: Record<string, number | boolean> = {}) {
  return { top: 16, right: 16, bottom: 64, left: 48, containLabel: true, ...overrides }
}

/** Base tooltip style. À spread avant `formatter` ou `trigger`. */
export function getTooltipBase(tc: EChartsThemeColors) {
  return {
    backgroundColor: tc.tooltipBg,
    borderColor: tc.tooltipBorder,
    textStyle: { color: tc.text, fontSize: 11 },
    extraCssText: 'border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.4)',
  } as const
}

/** Base legend style (bas du chart). */
export function getLegendBase(tc: EChartsThemeColors) {
  return {
    bottom: 0,
    textStyle: { color: tc.axisLabel, fontSize: 10 },
    itemWidth: 12,
    itemHeight: 8,
  } as const
}

/**
 * Couleurs des outcomes Halo (win/loss/tie/dnf).
 * Résolu côté composant via resolveToken (pas de hex direct).
 */
export function outcomeColor(outcome: string | undefined): string {
  switch (outcome) {
    case 'win':
      return resolveToken('outcome-win')
    case 'loss':
      return resolveToken('outcome-loss')
    case 'tie':
      return resolveToken('outcome-draw')
    case 'dnf':
      return resolveToken('outcome-dnf')
    default:
      return resolveToken('chart-series-1')
  }
}

/**
 * Couleurs séries (chart-series-1..8) cyclées modulo 8.
 * Pour wrappers multi-séries (ex. TimeseriesLine, Radar).
 */
export function seriesColor(index: number): string {
  const tokens: SemanticToken[] = [
    'chart-series-1',
    'chart-series-2',
    'chart-series-3',
    'chart-series-4',
    'chart-series-5',
    'chart-series-6',
    'chart-series-7',
    'chart-series-8',
  ]
  return resolveToken(tokens[index % tokens.length])
}

/**
 * hexToRgba — convertit un hex `#RRGGBB` (ou `RRGGBB`) en `rgba(r,g,b,alpha)`.
 *
 * Usage : générer un fond/trait/opacité tinté à partir d'un hex DÉJÀ résolu via
 * token (`resolveToken(...)`), dans un contexte canvas/ECharts qui n'accepte pas
 * les CSS vars (fond de badge encadré, intensité d'un histogramme momentum).
 * C'est un alpha-mix STRUCTUREL, pas un choix sémantique : la couleur reste
 * celle du token, seule la transparence varie.
 *
 * NB : distinct de la variante `color-mix(...)` de
 * `components/ui/match-card-presentation.ts`, qui opère sur une CSS var en
 * contexte DOM (pas un hex résolu) — les deux coexistent volontairement.
 * Source unique : ne jamais redéfinir localement (garde-rail hex-alpha.guard.test.ts).
 */
export function hexToRgba(hex: string, alpha: number): string {
  const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex)
  if (!m) return `rgba(0,0,0,${alpha})`
  return `rgba(${parseInt(m[1], 16)},${parseInt(m[2], 16)},${parseInt(m[3], 16)},${alpha})`
}

/**
 * Composant générique d'axe X catégoriel — paramètre les ticks selon le
 * nombre de catégories (rotation labels au-delà de 60 entrées).
 */
export function getCategoricalXAxis(
  categories: string[],
  tc: EChartsThemeColors = getEChartsThemeColors(),
): EChartsCoreOption['xAxis'] {
  const n = categories.length
  const base = getAxisBase(tc)
  return {
    ...base,
    type: 'category',
    data: categories,
    axisLabel: {
      ...base.axisLabel,
      interval: tickInterval(n) - 1,
      rotate: n > 60 ? 30 : n > 20 ? 15 : 0,
    },
  }
}

/**
 * Calcule l'intervalle de tick pour un nombre N de points (cap visuel).
 * Utilisé pour éviter les axes surchargés.
 */
export function tickInterval(n: number): number {
  if (n <= 10) return 1
  if (n <= 30) return 2
  if (n <= 60) return 5
  if (n <= 120) return 10
  return Math.ceil(n / 12)
}

/**
 * Format date FR court (DD/MM) pour les axes timeseries.
 * Source unique : `lib/formatters/date.ts` (réexport, plus de duplication).
 */
export { formatDateShort } from '@/lib/formatters'

/**
 * Format chiffre arrondi (1 décimale par défaut) pour tooltips/axes.
 * Délègue à `formatNumberFixed` (lib/formatters) en conservant le fallback
 * historique "-" des axes ECharts (le barrel renvoie "—").
 */
export function formatNumber(v: number, decimals = 1): string {
  return formatNumberFixed(v, decimals, '-')
}

/** Re-export pour les builders qui veulent récupérer le helper themeColors. */
export { getEChartsThemeColors }
export type { EChartsThemeColors }
