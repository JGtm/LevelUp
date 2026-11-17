/**
 * optionRead.testutil.ts — LA LECTURE TYPÉE D'UNE OPTION ECHARTS DANS UN TEST, sans `any`.
 *
 * Les fabriques d'option (`buildLeadShareOption`, `buildDivergentStepOption`…) rendent un `ChartOption`
 * ouvert ; un test, lui, lit des chemins précis (`o.series[0].markLine.data`, `o.xAxis.axisLabel.formatter(…)`).
 * `OptionRead` décrit ces chemins une fois : tous requis, parce que le test affirme à l'exécution ce qu'il lit
 * (un chemin absent fait échouer l'assertion, pas la compilation). `D` est le type d'un point de série.
 *
 * Réservé aux tests : un code de production n'a pas à relire sa propre option par chemin.
 */

/** Un axe, tel que les graphes de la Mainmise et de l'écart de score le posent. */
interface AxisRead {
  type: string
  min: number
  max: number
  interval: number
  splitLine: { show: boolean }
  axisLabel: { color: string; formatter: (value: number) => string }
}

/** Une série : aire en escalier, barres ou ligne. */
interface SeriesRead<D> {
  type: string
  step: string
  connectNulls: boolean
  data: D[]
  areaStyle: { color: string }
  itemStyle: { color: string }
  markLine: { data: unknown[]; lineStyle: { type: string; color: string } }
}

interface LegendRead {
  bottom: number
  data: Array<{ name: string; itemStyle: { color: string } }>
  textStyle: { color: string }
}

export interface OptionRead<D = unknown> {
  grid: object
  xAxis: AxisRead
  yAxis: AxisRead
  series: SeriesRead<D>[]
  legend: LegendRead
  tooltip: { trigger: string; formatter: (params: unknown) => string }
}

/** Lit une option rendue par une fabrique comme un `OptionRead`. */
export function readOption<D = unknown>(option: unknown): OptionRead<D> {
  return option as OptionRead<D>
}
