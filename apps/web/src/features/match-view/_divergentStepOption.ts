/**
 * _divergentStepOption.ts — L'OPTION ECHARTS COMMUNE AUX DEUX GRAPHES DIVERGENTS de l'onglet
 * Chronologie : « Écart de score » et « Supériorité numérique ».
 *
 * UNE GRANDEUR SIGNÉE AUTOUR DE ZÉRO, PEINTE PAR RÔLE. Au-dessus de zéro, l'équipe du joueur
 * est devant ; en dessous, c'est l'équipe d'en face. Les deux aires prennent les jetons
 * DIVERGENTS du projet (`divergent-pos` / `divergent-neg`), donc la palette d'accessibilité
 * réglée par l'utilisateur — jamais un vert ou un rouge écrit ici. La polarité est portée par la
 * POSITION par rapport à la ligne de zéro autant que par la teinte : la couleur n'est pas seule.
 *
 * DEUX SÉRIES, UNE PAR MOITIÉ. Chacune ne garde que sa part de la valeur (`max(v, 0)` /
 * `min(v, 0)`), en ESCALIER (`step: 'end'` : la valeur tient jusqu'au point suivant — ni le
 * score ni l'effectif ne s'interpolent). Les deux séries partagent les MÊMES abscisses, si bien
 * que le passage de zéro tombe exactement sur une marche, sans triangle d'interpolation.
 *
 * MÊME AXE DES TEMPS ET MÊMES MARGES QUE « Score dans le temps » (`left: 40, right: 16`,
 * abscisses en M:SS depuis le coup d'envoi) : empilés dans l'onglet, les graphes se lisent
 * à la verticale. UN SEUL AXE DE VALEURS, symétrique, aux graduations rondes.
 *
 * L'INFOBULLE LIT L'INSTANT DE LA SOURIS, pas le point le plus proche : sur un escalier, le point
 * le plus proche est parfois le SUIVANT, et lirait un score que le match n'affiche pas encore.
 * Ce qu'elle dit est écrit par l'appelant (`tooltipHtml`, déjà échappé) — ce module ne connaît ni
 * score ni effectif.
 *
 * Zéro dépendance React : testable en pur (`_divergentStepOption.test.ts`).
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  legendEntries,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import { tipNumber } from '@/components/charts/tipNumber'
import { resolveToken } from '@/lib/accessibility'

import { formatClock } from './_scoreCurve'

/** Un point de la série signée : tient jusqu'au suivant. */
export interface DivergentStepPoint {
  /** Millisecondes depuis le COUP D'ENVOI. */
  ms: number
  value: number
}

export interface DivergentStepInput {
  points: readonly DivergentStepPoint[]
  /** Fin de l'axe des temps, en ms depuis le coup d'envoi. */
  endMs: number
  /** Nom de la moitié positive (légende), ex. « Équipe en tête ». */
  positiveName: string
  /** Nom de la moitié négative (légende), ex. « Adversaires en tête ». */
  negativeName: string
  /** Le corps de l'infobulle à l'instant `ms` : HTML DÉJÀ échappé par l'appelant. */
  tooltipHtml: (ms: number) => string
  /** Plus petite demi-étendue de l'axe des valeurs (défaut 4). */
  minExtent?: number
}

/** Au plus trois graduations de part et d'autre de zéro : lisible sur 260 px de hauteur. */
const MAX_TICKS_PER_SIDE = 3

/** Pas candidats : 1, 2, 5, 10, 20, 50, 100… */
function* tickSteps(): Generator<number> {
  for (let decade = 1; ; decade *= 10) {
    yield decade
    yield decade * 2
    yield decade * 5
  }
}

/**
 * symmetricExtent rend la demi-étendue d'un axe symétrique autour de zéro et son pas de
 * graduation : le plus petit pas rond qui tient en `MAX_TICKS_PER_SIDE` graduations, et un
 * maximum multiple de ce pas — les graduations tombent juste, de -max à +max. `floor` est le
 * plancher : un écart de 1 ne gonfle pas l'axe, mais ne l'écrase pas non plus.
 */
export function symmetricExtent(maxAbs: number, floor = 4): { max: number; interval: number } {
  const wanted = Math.max(maxAbs, floor)
  for (const step of tickSteps()) {
    const ticks = Math.ceil(wanted / step)
    if (ticks <= MAX_TICKS_PER_SIDE) return { max: ticks * step, interval: step }
  }
  return { max: wanted, interval: wanted }
}

/** Le signe explicite d'une valeur d'axe : « +2 », « 0 », « −2 » (moins typographique). */
function signed(v: number): string {
  return tipNumber(v, 0, true)
}

/** buildDivergentStepOption — l'option ECharts des deux graphes divergents (cf. l'en-tête). */
export function buildDivergentStepOption(
  input: DivergentStepInput,
  tc: EChartsThemeColors,
): EChartsCoreOption {
  const { points, endMs, positiveName, negativeName, tooltipHtml } = input
  const posColor = resolveToken('divergent-pos')
  const negColor = resolveToken('divergent-neg')
  const { max, interval } = symmetricExtent(
    points.reduce((m, p) => Math.max(m, Math.abs(p.value)), 0),
    input.minExtent,
  )
  const area = (name: string, color: string, keep: (v: number) => number) => ({
    type: 'line' as const,
    name,
    step: 'end' as const,
    // Aucune marque de point : un escalier de plusieurs centaines de marches en deviendrait illisible.
    showSymbol: false,
    symbol: 'none',
    data: points.map((p) => [p.ms, keep(p.value)]),
    // Le contour est celui de l'aire : pas de trait propre, donc pas de double bord.
    lineStyle: { width: 0 },
    itemStyle: { color },
    areaStyle: { color, opacity: 0.75, origin: 0 },
    z: 3,
  })
  const positive = area(positiveName, posColor, (v) => Math.max(v, 0))
  const negative = area(negativeName, negColor, (v) => Math.min(v, 0))
  return {
    backgroundColor: CHART_BG,
    // `bottom` laisse la place à la légende posée sous le graphe ; `left` / `right` sont ceux
    // de « Score dans le temps » : les abscisses des graphes empilés tombent à la verticale.
    grid: getGridBase({ bottom: 44, left: 40, right: 16, top: 12 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line', lineStyle: { color: tc.splitLine } },
      formatter: (params: unknown): string => {
        const first = (Array.isArray(params) ? params[0] : params) as
          | { axisValue?: unknown; value?: [number, number] }
          | undefined
        const ms = typeof first?.axisValue === 'number' ? first.axisValue : (first?.value?.[0] ?? 0)
        return tooltipHtml(ms)
      },
    },
    legend: {
      ...getLegendBase(tc),
      left: 'center',
      icon: 'roundRect',
      // La légende nomme les deux moitiés ; en masquer une ne dirait rien de plus que le zéro.
      selectedMode: false,
      data: legendEntries([
        { name: positiveName, color: posColor },
        { name: negativeName, color: negColor },
      ]),
    },
    xAxis: {
      ...getAxisBase(tc),
      type: 'value',
      min: 0,
      max: endMs,
      axisLabel: { color: tc.axisLabel, fontSize: 9, formatter: (v: number) => formatClock(v) },
      splitLine: { show: false },
    },
    yAxis: {
      ...getAxisBase(tc),
      type: 'value',
      min: -max,
      max,
      interval,
      axisLabel: { color: tc.axisLabel, fontSize: 9, formatter: signed },
      splitLine: { show: true, lineStyle: { color: tc.splitLine, opacity: 0.35 } },
    },
    series: [
      {
        ...positive,
        // LA LIGNE DE ZÉRO, UNE FOIS : la frontière entre « devant » et « derrière », à l'encre d'axe.
        markLine: {
          silent: true,
          symbol: ['none', 'none'],
          label: { show: false },
          data: [{ yAxis: 0 }],
          lineStyle: { color: tc.axisLine, width: 1, type: 'solid' },
        },
      },
      negative,
    ],
  }
}
