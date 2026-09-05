/**
 * SynthesisBipolaireChart — synthesis.05.
 * Diverging bar horizontal : Solo (gauche, info/cyan) vs Escouade (droite, outcome-win/vert).
 * Normalisation : Escouade = +100, Solo = -(|solo|/|squad|)×100, borné [−200, 0].
 * Les labels affichent toujours les valeurs brutes signées (vérité absolue).
 * barGap: "-100%" pour overlap. clip: false pour labels hors bornes.
 */
import { useCallback, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { ComparisonMetricItem } from '@/lib/api/types'
import { formatClockMShort } from '@/lib/formatters'

interface Props {
  metrics: ComparisonMetricItem[]
  /** Labels résolus pour l'axe Y (même ordre que metrics). */
  fieldLabels?: string[]
  title?: ReactNode
  children?: ReactNode
  height?: number
}

function formatMetricValue(key: string, value: number): string {
  switch (key) {
    case 'win_rate':
      // 0-1 côté API (ADR 0006) → multiplier par 100 pour afficher %
      return `${(value * 100).toFixed(1)}%`
    case 'accuracy':
      // 0-100 nativement dans canonical.PlayerSelf (échelle API Halo) → pas de ×100
      return `${value.toFixed(1)}%`
    case 'offensive_conversion':
      return `${Math.round(value * 100)}%`
    case 'defensive_resistance':
      return `${Math.round((value - 1) * 100)}%`
    case 'performance_score':
    case 'match_count':
    case 'ranked_match_count':
    case 'avg_damage_dealt':
    case 'avg_damage_taken':
      return value.toFixed(0)
    case 'avg_life_seconds': {
      // SOUS LA MINUTE, LA MOYENNE SE LIT « 37 s » : le « 0m » d'un format M m SS s
      // n'apporterait rien sur une durée de vie moyenne, et cette décision-là reste ici.
      // L'ÉCRITURE du format, elle, vient de `lib/formatters` (registre 2026-09-05, N3) ;
      // l'arrondi précède la conversion, sans quoi 119,6 s se lirait « 1m60s ».
      const arrondi = Math.round(value)
      return arrondi < 60 ? `${arrondi}s` : formatClockMShort(arrondi * 1000)
    }
    case 'time_played_seconds': {
      const d = Math.floor(value / 86400)
      const h = Math.floor((value % 86400) / 3600)
      const m = Math.floor((value % 3600) / 60)
      const parts: string[] = []
      if (d > 0) parts.push(`${d}j`)
      if (h > 0 || d > 0) parts.push(`${h}h`)
      parts.push(`${m}m`)
      return parts.join(' ')
    }
    default:
      return value.toFixed(2)
  }
}

/**
 * Normalise la barre Solo sur [−100, 0].
 *
 * Cas nominal (deux positifs) : -(solo/squad)×100, plafonné à −100.
 * Deux négatifs : même ratio puis ÷2 → max −50, signalant visuellement
 *   le territoire négatif (performance médiocre des deux).
 * Croisé solo<0 / squad>0 : solo nettement pire → −50 (petite barre).
 * Croisé solo>0 / squad<0 : solo nettement meilleur → −100 (barre pleine).
 * Le plafond à −100 garantit que la barre solo reste dans les bornes de l'axe.
 */
function soloBarValue(solo: number, squad: number): number {
  if (Math.abs(squad) < 1e-9) return 0

  if (solo < 0 && squad > 0) return -50            // solo négatif, squad positif : mauvais
  if (solo >= 0 && squad < 0) return -100           // solo positif, squad négatif : excellent

  const raw = Math.max(-100, -(Math.abs(solo) / Math.abs(squad)) * 100)
  if (solo < 0 && squad < 0) return raw / 2        // deux négatifs : comprimer à 50 %
  return raw                                        // deux positifs : formule standard
}

function buildBipolaireOption(
  metrics: ComparisonMetricItem[],
  fieldLabels: string[],
): EChartsCoreOption {
  if (metrics.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const dps = [...metrics].reverse()
  const labels = dps.map((m, i) => fieldLabels[metrics.length - 1 - i] ?? m.label)

  const soloVals = dps.map((m) => soloBarValue(m.solo_value, m.squad_value))
  const squadVals = dps.map(() => 100)

  const soloTexts = dps.map((m) => formatMetricValue(m.label, m.solo_value))
  const squadTexts = dps.map((m) => formatMetricValue(m.label, m.squad_value))

  const soloColors = dps.map((m) =>
    m.solo_value < 0 ? resolveToken('outcome-loss') : resolveToken('info'),
  )
  const squadColors = dps.map((m) =>
    m.squad_value < 0 ? resolveToken('outcome-loss') : resolveToken('outcome-win'),
  )
  const dot = (c: string) =>
    `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${c};margin-right:6px;vertical-align:middle"></span>`

  return {
    backgroundColor: CHART_BG,
    grid: { left: 24, right: 24, top: 16, bottom: 40, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? (params as { dataIndex?: number; name?: string }[]) : []
        const idx = arr[0]?.dataIndex ?? 0
        const title = escapeHtml(arr[0]?.name ?? labels[idx] ?? '')
        const solo = soloTexts[idx] ?? ''
        const squad = squadTexts[idx] ?? ''
        return [
          `<div style="font-weight:600;margin-bottom:4px">${title}</div>`,
          `<div>${dot(soloColors[idx])}Solo : <b>${solo}</b></div>`,
          `<div>${dot(squadColors[idx])}Escouade : <b>${squad}</b></div>`,
        ].join('')
      },
    },
    legend: { ...getLegendBase(tc), orient: 'horizontal', bottom: 5, left: 'center', top: undefined },
    xAxis: {
      ...axis,
      type: 'value',
      min: -130,
      max: 130,
      axisLabel: { show: false },
      splitLine: { show: false },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: labels,
      axisTick: { alignWithLabel: true },
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
    },
    barCategoryGap: '30%',
    series: [
      {
        name: 'Solo',
        type: 'bar',
        data: soloVals.map((v, i) => ({
          value: v,
          itemStyle: { color: soloColors[i] },
        })),
        clip: false,
        barWidth: '50%',
        barGap: '-100%',
        itemStyle: { color: resolveToken('info') },
        label: {
          show: true,
          position: 'left',
          color: tc.axisLabel,
          fontSize: 10,
          formatter: (p: { dataIndex: number }) => soloTexts[p.dataIndex] ?? '',
        },
        markLine: {
          silent: true,
          symbol: 'none',
          data: [{ xAxis: 0, lineStyle: { color: tc.splitLine, width: 1, type: 'solid' as const }, label: { show: false } }],
        },
      },
      {
        name: 'Escouade',
        type: 'bar',
        data: squadVals.map((v, i) => ({
          value: v,
          itemStyle: { color: squadColors[i] },
        })),
        clip: false,
        barWidth: '50%',
        barGap: '-100%',
        itemStyle: { color: resolveToken('outcome-win') },
        label: {
          show: true,
          position: 'right',
          color: tc.axisLabel,
          fontSize: 10,
          formatter: (p: { dataIndex: number }) => squadTexts[p.dataIndex] ?? '',
        },
      },
    ],
  }
}

export function SynthesisBipolaireChart({ metrics, fieldLabels, title, children, height }: Props) {
  const resolvedLabels = fieldLabels ?? metrics.map((m) => m.label)
  const series: ChartSeries<ComparisonMetricItem>[] = metrics.length > 0
    ? [{ key: 'bipolaire', datapoints: metrics }]
    : []

  // Clés de recalcul stables : on ne rebuild l'option que si le contenu change.
  const metricsKey = JSON.stringify(metrics)
  const labelsKey = JSON.stringify(resolvedLabels)
  const buildOption = useCallback(
    () => buildBipolaireOption(metrics, resolvedLabels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [metricsKey, labelsKey],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<ComparisonMetricItem>[]) => EChartsCoreOption}
      height={height ?? Math.max(240, 36 * metrics.length)}
    >
      {children}
    </ChartCard>
  )
}
