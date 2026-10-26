/**
 * SynthesisOutcomesByGroupChart — synthesis.01 (par carte) et synthesis.02 (par mode).
 *
 * Deux grilles superposées sur le MÊME axe des catégories : en haut, une barre par colonne du score
 * de performance moyen (0..100, couleur du palier `perf-tier-N`) ; en bas, les issues empilées avec
 * leur effectif à l'intérieur de chaque segment. Une colonne sans match noté montre, à la place de la
 * barre, un emplacement en pointillé sans valeur.
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

/** Nombre de séries d'issues ; les séries de performance viennent après, aux indices suivants. */
const OUTCOME_SERIES_COUNT = 4
const PERF_SERIES_INDEX = OUTCOME_SERIES_COUNT
const PERF_PLACEHOLDER_INDEX = OUTCOME_SERIES_COUNT + 1

const PERF_MAX = 100
const PERF_TIER_MIN = 1
const PERF_TIER_MAX = 5

/**
 * Découpe verticale (px) : bande de performance, espace des valeurs au-dessus des colonnes, puis
 * les issues empilées sur la hauteur qu'elles avaient avant l'ajout de la bande (230 px).
 */
const GRID_TOP = 30
const PERF_BAND_HEIGHT = 64
const PERF_BAND_GAP = 26
const STACK_HEIGHT = 230
const GRID_BOTTOM = 100
const CHART_HEIGHT = GRID_TOP + PERF_BAND_HEIGHT + PERF_BAND_GAP + STACK_HEIGHT + GRID_BOTTOM

const GRID_SIDES = { left: 40, right: 20 }
const BAR_CATEGORY_GAP = '15%'

/** Palier valide d'une entrée notée, sinon null (colonne sans score). */
function scoredTier(e: GroupEntry): number | null {
  if (e.performance_avg == null || e.perf_tier == null) return null
  return e.perf_tier >= PERF_TIER_MIN && e.perf_tier <= PERF_TIER_MAX ? e.perf_tier : null
}

function outcomeSeries(entries: GroupEntry[], labels: OutcomeLabels) {
  const labelAll = barLabel({ formatter: (p: { value: number }) => tipNumberAuto(p.value) })
  const labelIfNonzero = barLabel({ formatter: (p: { value: number }) => (p.value > 0 ? tipNumberAuto(p.value) : '') })
  const stacked = { type: 'bar', stack: 'total', xAxisIndex: 1, yAxisIndex: 1, barCategoryGap: BAR_CATEGORY_GAP }
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
 * Les deux séries de la bande de performance, empilées dans la même pile : par colonne, soit la
 * barre du score (couleur du palier, valeur entière au-dessus), soit l'emplacement en pointillé
 * pleine hauteur — jamais les deux.
 */
function performanceSeries(entries: GroupEntry[], perfLabel: string, neutral: string, textColor: string) {
  const band = { type: 'bar', stack: 'perf', xAxisIndex: 0, yAxisIndex: 0, name: perfLabel, barCategoryGap: BAR_CATEGORY_GAP }
  return [
    {
      ...band,
      data: entries.map((e) => {
        const tier = scoredTier(e)
        return tier === null
          ? null
          : { value: e.performance_avg, itemStyle: { color: resolveToken(`perf-tier-${tier}` as SemanticToken) } }
      }),
      label: {
        show: true,
        position: 'top',
        color: textColor,
        formatter: (p: { value: number }) => tipNumber(p.value),
      },
    },
    {
      ...band,
      data: entries.map((e) => (scoredTier(e) === null ? PERF_MAX : null)),
      itemStyle: { color: 'transparent', borderColor: neutral, borderWidth: 1, borderType: 'dashed' },
      label: { show: false },
    },
  ]
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
  const series = [...outcomes, ...performanceSeries(entries, perfLabel, tc.axisLabel, tc.text)]
  const gridLine = { show: true, lineStyle: { color: tc.splitLine } }
  const stackOrder = stackTopFirstOrder(series)

  return {
    backgroundColor: CHART_BG,
    grid: [
      { ...GRID_SIDES, top: GRID_TOP, height: PERF_BAND_HEIGHT, containLabel: false },
      { ...GRID_SIDES, top: GRID_TOP + PERF_BAND_HEIGHT + PERF_BAND_GAP, bottom: GRID_BOTTOM, containLabel: false },
    ],
    // Un seul curseur sur les deux grilles : le survol d'une colonne montre la perf ET les issues.
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // La performance d'abord (grille du haut), puis une ligne par issue présente : un segment à
      // zéro n'est ni dessiné ni étiqueté. Colonne sans score : « — ».
      formatter: axisTooltipFormatter({
        value: (r) =>
          r.seriesIndex === PERF_PLACEHOLDER_INDEX ? '—' : r.seriesIndex === PERF_SERIES_INDEX ? tipNumber(r.value) : tipNumberAuto(r.value),
        order: (a, b) => {
          const pa = a.seriesIndex >= PERF_SERIES_INDEX
          const pb = b.seriesIndex >= PERF_SERIES_INDEX
          return pa === pb ? stackOrder(a, b) : pa ? -1 : 1
        },
        skip: (r) => r.seriesIndex < OUTCOME_SERIES_COUNT && r.value === 0,
      }),
    },
    legend: {
      ...getLegendBase(tc),
      data: [labels.win, labels.loss, labels.tie, labels.dnf],
      orient: 'horizontal',
      bottom: 8,
      left: 'center',
      right: undefined,
      top: undefined,
    },
    xAxis: [
      // Grille du haut : mêmes catégories, étiquettes masquées (celles du bas font foi).
      { ...axis, type: 'category', gridIndex: 0, data: categories, splitLine: gridLine, axisLabel: { show: false } },
      {
        ...axis,
        type: 'category',
        gridIndex: 1,
        data: categories,
        splitLine: gridLine,
        axisLabel: { ...axis.axisLabel, rotate: 45 },
      },
    ],
    yAxis: [
      {
        ...axis,
        type: 'value',
        gridIndex: 0,
        min: 0,
        max: PERF_MAX,
        interval: PERF_MAX / 2,
        name: perfLabel,
        nameTextStyle: { align: 'left' },
        splitLine: gridLine,
      },
      { ...axis, type: 'value', gridIndex: 1, name: yAxisLabel, splitLine: gridLine },
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
      height={height ?? CHART_HEIGHT}
    />
  )
}
