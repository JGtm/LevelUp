/**
 * SynthesisOutcomesByGroupChart — synthesis.01 (par carte) et synthesis.02 (par mode).
 *
 * Une seule grille : les issues empilées (axe de gauche, effectif à l'intérieur de chaque segment) et,
 * par-dessus, une courbe du score de performance moyen (axe de droite, 0..100). Les points de la courbe
 * prennent la couleur de leur palier (`perf-tier-N`) ; une colonne sans match noté laisse un trou dans la courbe.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { barLabel } from '@/components/charts/barLabel'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { axisTooltipFormatter, stackTopFirstOrder } from '@/components/charts/axisTooltip'
import { tipNumber, tipNumberAuto } from '@/components/charts/tipNumber'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { useFieldLabel, useFieldMappings, useOutcomeLabel } from '@/lib/i18n/fieldMappings'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'

interface GroupEntry {
  name: string
  wins: number
  losses: number
  ties: number
  unfinished: number
  win_rate?: number
  /** Score de performance moyen (0..100) des matchs notés ; absent = aucun match noté. */
  performance_avg?: number | null
  /** Palier canonique 1..5 de `performance_avg` (0 ou absent = aucun match noté). */
  perf_tier?: number
}

interface Props {
  entries: GroupEntry[]
  title?: string
  /** Texte de la bulle ⓘ du titre ; absent = titre seul. */
  info?: string
  height?: number
}

interface OutcomeLabels { win: string; loss: string; tie: string; dnf: string }

/** Les séries d'issues viennent d'abord ; la courbe de performance est la suivante. */
const OUTCOME_SERIES_COUNT = 4
const PERF_SERIES_INDEX = OUTCOME_SERIES_COUNT

const PERF_MAX = 100
const PERF_TIER_MIN = 1
const PERF_TIER_MAX = 5
const PERF_SYMBOL_SIZE = 8

/**
 * Marge droite : de quoi loger les graduations et le nom de l'axe de performance. Marge basse : les
 * étiquettes inclinées des noms longs ne descendent pas jusqu'à la légende.
 */
const GRID = { left: 40, right: 56, top: 30, bottom: 120, containLabel: false }
const DEFAULT_HEIGHT = 520
const BAR_CATEGORY_GAP = '15%'

/** Palier valide d'une entrée notée, sinon null (colonne sans score). */
function scoredTier(e: GroupEntry): number | null {
  if (e.performance_avg == null || e.perf_tier == null) return null
  return e.perf_tier >= PERF_TIER_MIN && e.perf_tier <= PERF_TIER_MAX ? e.perf_tier : null
}

function outcomeSeries(entries: GroupEntry[], labels: OutcomeLabels) {
  const labelAll = barLabel({ formatter: (p: { value: number }) => tipNumberAuto(p.value) })
  const labelIfNonzero = barLabel({ formatter: (p: { value: number }) => (p.value > 0 ? tipNumberAuto(p.value) : '') })
  const stacked = { type: 'bar', stack: 'total', barCategoryGap: BAR_CATEGORY_GAP }
  return [
    {
      ...stacked,
      name: labels.win,
      data: entries.map((e) => ({ value: e.wins, win_rate: e.win_rate })),
      itemStyle: { color: resolveToken('outcome-win'), opacity: 0.85 },
      label: labelAll,
    },
    {
      ...stacked,
      name: labels.loss,
      data: entries.map((e) => ({ value: e.losses })),
      itemStyle: { color: resolveToken('outcome-loss'), opacity: 0.75 },
      label: labelAll,
    },
    {
      ...stacked,
      name: labels.tie,
      data: entries.map((e) => ({ value: e.ties })),
      itemStyle: { color: resolveToken('outcome-draw'), opacity: 0.7 },
      label: labelIfNonzero,
    },
    {
      ...stacked,
      name: labels.dnf,
      data: entries.map((e) => ({ value: e.unfinished })),
      itemStyle: { color: resolveToken('outcome-dnf'), opacity: 0.6 },
      label: labelIfNonzero,
    },
  ]
}

/**
 * Courbe du score de performance moyen sur l'axe de droite. Trait neutre, points de la couleur de leur
 * palier ; une colonne sans score vaut `null` (trou dans la courbe, jamais relié). Aucune étiquette de
 * valeur : jusqu'à soixante-dix colonnes, elle est dans l'infobulle.
 */
function performanceSeries(entries: GroupEntry[], perfLabel: string, neutral: string, separator: string) {
  return {
    type: 'line',
    name: perfLabel,
    yAxisIndex: 1,
    z: 3,
    connectNulls: false,
    symbol: 'circle',
    symbolSize: PERF_SYMBOL_SIZE,
    lineStyle: { color: neutral, width: 2 },
    itemStyle: { color: neutral, borderColor: separator, borderWidth: 1 },
    label: { show: false },
    data: entries.map((e) => {
      const tier = scoredTier(e)
      return tier === null
        ? null
        : { value: e.performance_avg, itemStyle: { color: resolveToken(`perf-tier-${tier}` as SemanticToken) } }
    }),
  }
}

/** Paramètre d'une série tel que l'infobulle par axe le reçoit ; `unscored` marque la ligne de performance sans score. */
type TipParam = { seriesIndex?: number; dataIndex?: number; value?: unknown; unscored?: boolean }

/**
 * Garantit une ligne de performance par colonne survolée : ECharts omet ou vide la série d'une colonne
 * sans score, l'infobulle doit pourtant l'annoncer (« — »).
 */
function withPerformanceRow(params: unknown, perfLabel: string, neutral: string): unknown[] {
  const list = (Array.isArray(params) ? params : [params]) as (TipParam & Record<string, unknown>)[]
  if (list.length === 0 || list[0] == null) return []
  const perf = list.find((p) => p.seriesIndex === PERF_SERIES_INDEX)
  if (perf && typeof perf.value === 'number' && Number.isFinite(perf.value)) return list
  const unscored = {
    ...list[0],
    seriesName: perfLabel,
    seriesIndex: PERF_SERIES_INDEX,
    color: neutral,
    value: 0,
    unscored: true,
  }
  return [...list.filter((p) => p.seriesIndex !== PERF_SERIES_INDEX), unscored]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildOutcomesOption(
  entries: GroupEntry[],
  labels: OutcomeLabels,
  yAxisLabel: string,
  perfLabel: string,
): EChartsCoreOption {
  if (entries.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const categories = entries.map((e) => e.name)
  const outcomes = outcomeSeries(entries, labels)
  const series = [...outcomes, performanceSeries(entries, perfLabel, tc.axisLabel, tc.card)]
  const gridLine = { show: true, lineStyle: { color: tc.splitLine } }
  const stackOrder = stackTopFirstOrder(series)
  const tipFormatter = axisTooltipFormatter({
    value: (r) => ((r.param as TipParam).unscored ? '—' : r.seriesIndex === PERF_SERIES_INDEX ? tipNumber(r.value) : tipNumberAuto(r.value)),
    // La performance d'abord, puis une ligne par issue présente : un segment à zéro n'est ni dessiné
    // ni étiqueté.
    order: (a, b) => {
      const pa = a.seriesIndex === PERF_SERIES_INDEX
      const pb = b.seriesIndex === PERF_SERIES_INDEX
      return pa === pb ? stackOrder(a, b) : pa ? -1 : 1
    },
    skip: (r) => r.seriesIndex < OUTCOME_SERIES_COUNT && r.value === 0,
  })

  return {
    backgroundColor: CHART_BG,
    grid: GRID,
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => tipFormatter(withPerformanceRow(params, perfLabel, tc.axisLabel)),
    },
    legend: { ...getLegendBase(tc), orient: 'horizontal', bottom: 8, left: 'center', right: undefined, top: undefined },
    xAxis: {
      ...axis,
      type: 'category',
      data: categories,
      splitLine: gridLine,
      axisLabel: { ...axis.axisLabel, rotate: 45 },
    },
    yAxis: [
      { ...axis, type: 'value', name: yAxisLabel, splitLine: gridLine },
      {
        ...axis,
        type: 'value',
        position: 'right',
        min: 0,
        max: PERF_MAX,
        interval: PERF_MAX / 4,
        name: perfLabel,
        nameTextStyle: { align: 'right' },
        splitLine: { show: false },
      },
    ],
    series,
  }
}

type Pt = { name: string }

export function SynthesisOutcomesByGroupChart({ entries, title, info, height }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const winLabel  = useOutcomeLabel('win')
  const lossLabel = useOutcomeLabel('loss')
  const tieLabel  = useOutcomeLabel('tie')
  const dnfLabel  = useOutcomeLabel('dnf')
  const perfLabel = useFieldLabel('performance_score')
  const locale = useAppShellStore((st) => st.locale)
  const yAxisLabel =
    fieldMappings?.fields['match_count']?.label ?? formatMessage(commonManifest, 'common.charts.axis_matches', locale)

  const series: ChartSeries<Pt>[] = entries.length > 0
    ? [{ key: 'outcomes', datapoints: entries.map((e) => ({ name: e.name })) }]
    : []

  // Clé de recalcul stable : on ne rebuild l'option que si le contenu change.
  const entriesKey = JSON.stringify(entries)
  const buildOption = useCallback(
    () =>
      buildOutcomesOption(
        entries,
        { win: winLabel, loss: lossLabel, tie: tieLabel, dnf: dnfLabel },
        yAxisLabel,
        perfLabel,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [entriesKey, winLabel, lossLabel, tieLabel, dnfLabel, yAxisLabel, perfLabel],
  )

  return (
    <ChartCard
      title={title && info ? titleWithInfo(<p>{info}</p>)(title) : title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? DEFAULT_HEIGHT}
    />
  )
}
