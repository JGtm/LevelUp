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
    extraCssText: 'border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.4)', // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
  } as const
}

/**
 * Symbole de série CACHÉ au repos et RÉVÉLÉ au survol du point (à spread dans une
 * série `line`). ECharts n'affiche le symbole emphasé sous `showSymbol: false` que
 * si `symbol` reste défini : avec `symbol: 'none'` le canvas n'offre aucune prise
 * et le graphe se lit comme une image figée (v7.3 lot 2, item 2.3c).
 *
 * Source unique des 4 séries concernées (écart cumulé au FDA attendu × 2 surfaces,
 * médiane et courbe d'équipe du profil d'intensité) — ne pas ré-écrire le trio
 * showSymbol/symbol/emphasis à la main dans un builder.
 */
export function hoverRevealSymbol(color: string, size = 7) {
  return {
    showSymbol: false,
    symbol: 'circle',
    symbolSize: size,
    itemStyle: { color },
    emphasis: { scale: 1.6 },
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
  if (!m) return `rgba(0,0,0,${alpha})` // color-allow: 2026-09-06 (ronde 2, N3) — REPLI d'un helper de CONVERSION hex -> rgba : la valeur ne s'affiche que si l'entree n'est pas un hex, elle ne nomme aucune couleur de charte
  return `rgba(${parseInt(m[1], 16)},${parseInt(m[2], 16)},${parseInt(m[3], 16)},${alpha})` // color-allow: 2026-09-06 (ronde 2, N3) — CONVERSION d'une couleur DEJA resolue (hex du theme) en rgba pour son alpha : ce helper ne nomme aucune couleur
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
 * stackedAxisExtent — borne min/max d'un axe Y à barres EMPILÉES, calculée sur
 * l'INTÉGRALITÉ des séries fournies plutôt que sur les séries actuellement
 * visibles.
 *
 * Bug corrigé (retours utilisateur 2026-08-29, item 5) : un axe Y sans min/max
 * explicite est recalculé par ECharts sur les séries VISIBLES à chaque rendu
 * (`notMerge`) — masquer/afficher un type (bouton légende « Bonus ») ou un
 * joueur fait donc bouger l'échelle entière d'un clic à l'autre, cassant la
 * comparabilité visuelle. En fixant `yAxis.min`/`max` sur l'extent COMPLET
 * (bonus et joueurs masqués INCLUS), l'échelle devient stable — seule la
 * visibilité des barres change (ça corrige aussi le rescale au masquage d'un
 * joueur : même cause).
 *
 * `positiveStacks` / `negativeStacks` : chaque élément est une PILE, c.-à-d. un
 * ensemble de séries empilées ensemble (même `stack` ECharts — typiquement un
 * joueur) ; une pile est un tableau de séries, une série un tableau de valeurs
 * par index x (`null`/`undefined` comptent pour 0, comme ECharts). Une pile
 * d'une seule série représente une barre non empilée (ex. les morts de
 * TimeseriesKdaTrend, sans `stack`). `negativeStacks` attend des valeurs DÉJÀ
 * NÉGATIVES — les mêmes tableaux que ceux passés en `data` à la série ECharts
 * (ex. `-p.deaths`) — aucune transformation de signe supplémentaire :
 *   - `max` = la plus grande somme empilée sur `positiveStacks`, toutes piles
 *     et index confondus (0 si aucune pile ou aucune valeur positive).
 *   - `min` = la plus grande somme empilée (en valeur absolue) sur
 *     `negativeStacks` — 0 si omis (axe qui ne descend jamais sous zéro, ex.
 *     TimeseriesKdaTrend où les morts restent positives).
 *
 * Marge : la borne est arrondie à la dizaine ENTIÈRE la plus proche vers
 * l'extérieur — même principe que `oneLifeWindowBoundsForData` côté « une vie »
 * (`lib/charts/oneLifeWindow.ts`) — pour qu'aucune barre ne touche le bord du
 * cadre. Les deux vivent dans des modules distincts (celui-ci générique, l'autre
 * spécifique au domaine « une vie ») : même règle d'arrondi, pas de dépendance
 * croisée entre un util de charts générique et une logique métier.
 */
export function stackedAxisExtent(
  positiveStacks: Array<Array<Array<number | null | undefined>>>,
  negativeStacks: Array<Array<Array<number | null | undefined>>> = [],
): { min: number; max: number } {
  const max = extremeStackedSum(positiveStacks, Math.max)
  const min = extremeStackedSum(negativeStacks, Math.min)
  return {
    max: max > 0 ? Math.ceil(max / 10) * 10 : 0,
    min: min < 0 ? Math.floor(min / 10) * 10 : 0,
  }
}

/** Somme empilée la plus extrême (`Math.max`/`Math.min`) sur un ensemble de piles. */
function extremeStackedSum(
  stacks: Array<Array<Array<number | null | undefined>>>,
  pick: (a: number, b: number) => number,
): number {
  let extreme = 0
  for (const stack of stacks) {
    const length = stack.reduce((m, series) => Math.max(m, series.length), 0)
    for (let i = 0; i < length; i++) {
      let sum = 0
      for (const series of stack) sum += series[i] ?? 0
      extreme = pick(extreme, sum)
    }
  }
  return extreme
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
