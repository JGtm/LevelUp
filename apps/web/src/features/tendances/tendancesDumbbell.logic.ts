/**
 * tendancesDumbbell.logic — le builder PUR des graphiques en haltères de la page Tendances
 * (« Moyenne par match en défaite et en victoire », « Médailles par match »), sur le modèle de
 * `features/session-detail/SessionMmrDumbbell` : une série `custom` pour le segment, deux séries
 * `scatter` pour les points, un axe Y de catégories inversé (première ligne en haut).
 *
 * DIFFÉRENCES AVEC LE MODÈLE. Noms et couleurs des deux séries viennent de l'appelant (les
 * couleurs en JETONS, résolus ICI à chaque construction : l'option suit le thème et la palette
 * d'accessibilité). Une ligne dont un point manque n'a pas de segment. Un texte peut s'écrire
 * au-dessus de chaque point, un repère vertical en pointillé peut se poser, et l'axe X est
 * masqué ou formaté selon la fonction fournie.
 *
 * LÉGENDE. En bas, centrée, limitée aux séries qui ont au moins un point.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'

/** Une ligne de l'haltère. `a` et `b` sont les abscisses des deux points (`null` = pas de point). */
export interface DumbbellRow {
  label: string
  a: number | null
  b: number | null
  /** Texte écrit au-dessus du point A, ou du point B. */
  aText?: string
  bText?: string
  /** Lignes de l'infobulle (déjà traduites), sous le libellé de la ligne. */
  tooltip: string[]
}

export interface TendancesDumbbellInput {
  rows: readonly DumbbellRow[]
  nameA: string
  nameB: string
  colorA: SemanticToken
  colorB: SemanticToken
  /** Repère vertical en pointillé. */
  reference?: number
  /** Formate une graduation de l'axe X ; `null` masque les étiquettes de l'axe. */
  xAxisLabel: ((value: number) => string) | null
}

/** Hauteur d'une ligne de l'haltère, en pixels. */
const ROW_HEIGHT_PX = 30
/** Marges haute et basse (axe X et légende compris), en pixels. */
const VERTICAL_PADDING_PX = 72

/** Hauteur proportionnelle au nombre de lignes. */
export function dumbbellHeight(rows: number): number {
  return Math.max(160, rows * ROW_HEIGHT_PX + VERTICAL_PADDING_PX)
}

interface PointDatum {
  value: [number, number]
  text?: string
}

function pointsOf(
  rows: readonly DumbbellRow[],
  pick: (row: DumbbellRow) => { x: number | null; text?: string },
): PointDatum[] {
  const data: PointDatum[] = []
  rows.forEach((row, index) => {
    const { x, text } = pick(row)
    if (x != null) data.push({ value: [x, index], text })
  })
  return data
}

function pointSeries(
  name: string,
  color: string,
  data: PointDatum[],
  textColor: string,
  options: object,
): object {
  return {
    name,
    type: 'scatter',
    symbolSize: 11,
    itemStyle: { color },
    data,
    z: 2,
    label: {
      show: true,
      position: 'top',
      color: textColor,
      fontSize: 10,
      formatter: (p: { data: PointDatum }) => p.data.text ?? '',
    },
    ...options,
  }
}

interface TooltipParam {
  seriesType?: string
  value: number[]
}

function tooltipFormatter(rows: readonly DumbbellRow[]) {
  return (raw: unknown): string => {
    const items = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
    const point = items.find((p) => p.seriesType === 'scatter')
    const row = point ? rows[point.value[1]] : undefined
    if (!row) return ''
    return [`<b>${escapeHtml(row.label)}</b>`, ...row.tooltip.map(escapeHtml)].join('<br/>')
  }
}

/** Marge au-delà du point le plus éloigné du repère, en part de cet écart. */
const X_PADDING_RATIO = 0.12

/**
 * Bornes de l'axe X d'un haltère À REPÈRE : SYMÉTRIQUES autour du repère (au centre), le
 * point le plus éloigné près du bord, une marge pour les valeurs écrites. `null` sans repère
 * ou sans point : l'axe s'ajuste alors aux points (`scale`), graduations arrondies.
 */
export function dumbbellXBounds(
  rows: readonly DumbbellRow[],
  reference: number | undefined,
): { min: number; max: number } | null {
  const xs = rows.flatMap((r) => [r.a, r.b]).filter((x): x is number => x != null)
  if (reference == null || xs.length === 0) return null
  const half = Math.max(...xs.map((x) => Math.abs(x - reference))) || 1
  const span = half * (1 + X_PADDING_RATIO)
  return { min: reference - span, max: reference + span }
}

/** Axes de l'haltère : X en valeurs (étiquettes selon la fonction fournie), Y en catégories inversées. */
function dumbbellAxes(
  rows: readonly DumbbellRow[],
  input: Pick<TendancesDumbbellInput, 'xAxisLabel' | 'reference'>,
  axis: ReturnType<typeof getAxisBase>,
) {
  const { xAxisLabel } = input
  const bounds = dumbbellXBounds(rows, input.reference)
  return {
    xAxis: {
      ...axis,
      type: 'value',
      ...(bounds ?? { scale: true }),
      axisLabel: xAxisLabel ? { ...axis.axisLabel, formatter: xAxisLabel } : { show: false },
    },
    yAxis: {
      ...axis,
      type: 'category',
      inverse: true,
      data: rows.map((r) => r.label),
      axisLabel: { ...axis.axisLabel, interval: 0 },
    },
  }
}

/** Segments de liaison [ligne, a, b] : seules les lignes aux deux points en ont un. */
function segmentsOf(rows: readonly DumbbellRow[]): number[][] {
  const segments: number[][] = []
  rows.forEach((r, index) => {
    if (r.a != null && r.b != null) segments.push([index, r.a, r.b])
  })
  return segments
}

/**
 * Série `custom` du segment, sous les points ; muette, hors légende.
 *
 * `encode` EST OBLIGATOIRE : sans lui, la première dimension de [ligne, a, b] (la LIGNE) part
 * sur l'axe X, dont l'étendue s'étirait jusqu'au nombre de lignes — tous les points tassés
 * dans un coin du graphe.
 */
function segmentSeries(segments: number[][], linkColor: string): object {
  return {
    type: 'custom',
    silent: true,
    z: 1,
    encode: { x: [1, 2], y: 0 },
    data: segments,
    renderItem: (
      _params: unknown,
      api: { value: (i: number) => number; coord: (v: number[]) => number[] },
    ) => {
      const index = api.value(0)
      const a = api.coord([api.value(1), index])
      const b = api.coord([api.value(2), index])
      return {
        type: 'line',
        shape: { x1: a[0], y1: a[1], x2: b[0], y2: b[1] },
        style: { stroke: linkColor, lineWidth: 2 },
      }
    },
  }
}

/** Option ECharts d'un graphique en haltères ; sans ligne, option vide (le cadre dit « vide »). */
export function buildTendancesDumbbellOption(input: TendancesDumbbellInput): EChartsCoreOption {
  const { rows } = input
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const colorA = resolveToken(input.colorA)
  const colorB = resolveToken(input.colorB)
  const linkColor = resolveToken('divergent-neutral')

  const dataA = pointsOf(rows, (r) => ({ x: r.a, text: r.aText }))
  const dataB = pointsOf(rows, (r) => ({ x: r.b, text: r.bText }))

  const referenceMark =
    input.reference != null
      ? {
          markLine: {
            silent: true,
            symbol: 'none',
            label: { show: false },
            lineStyle: { type: 'dashed', width: 1, color: tc.axisLabel },
            data: [{ xAxis: input.reference }],
          },
        }
      : {}

  const entries = [
    ...(dataA.length > 0 ? [{ name: input.nameA, color: colorA }] : []),
    ...(dataB.length > 0 ? [{ name: input.nameB, color: colorB }] : []),
  ]

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 16, bottom: 40, left: 8, right: 48 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: tooltipFormatter(rows),
    },
    legend: { ...getLegendBase(tc), left: 'center', data: legendEntries(entries) },
    ...dumbbellAxes(rows, input, axis),
    series: [
      segmentSeries(segmentsOf(rows), linkColor),
      pointSeries(input.nameA, colorA, dataA, tc.text, {}),
      pointSeries(input.nameB, colorB, dataB, tc.text, { symbol: 'diamond', ...referenceMark }),
    ],
  }
}
