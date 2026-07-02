/**
 * SessionOcdrBars — Rendement offensif (OC) et Résistance défensive (DR) par match.
 *
 * Barres groupées par match (X = #N + carte), normalisées en % de P80 :
 *   OC_norm  = OC / 0.90 × 100    (repère = 0.90)
 *   DR_norm  = max(0, (DR − 1.0) / 0.65 × 100)  (repère_excess = 0.65)
 *
 * 100% sur l'axe Y = exactement au niveau du repère élite (ligne pointillée).
 * markLine de moyenne pour chaque métrique. Couleurs : divergent-pos (OC),
 * divergent-neutral (DR), cohérentes avec CombatYieldBar et la KPI session.
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useOffensiveConversionP80, useProvidesDamageTaken } from '@/lib/damage/effectiveHp'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

/** Repères barre (frontière élite). OC = title-aware (hook useOffensiveConversionP80 :
 *  0.90 Infinite / 1.264 Halo 5) ; DR = miroir const Go (h5 sans damage_taken → DR N/A). */
const DR_P80 = 1.65
const DR_P80_EXCESS = DR_P80 - 1.0 // 0.65

interface OcdrPoint {
  label: string
  ocNorm: number  // OC / P80 × 100
  drNorm: number  // (DR − 1) / P80_excess × 100, plancher 0
  ocRaw: number
  drRaw: number
}

interface OcdrOpts {
  ocLabel: string
  drLabel: string
  p80Label: string
  meanLabel: string
  /** false (titre sans damage_taken, ex. Halo 5) → barres + moyenne DR retirées. */
  showDr?: boolean
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionOcdrBarsOption(
  series: ChartSeries<OcdrPoint>[],
  opts: OcdrOpts,
): EChartsCoreOption {
  const showDr = opts.showDr ?? true
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const ocColor  = resolveToken('divergent-pos')
  const drColor  = resolveToken('divergent-neutral')
  const refColor = resolveToken('divergent-neutral')
  const round1 = (n: number) => Math.round(n * 10) / 10

  const ocMean  = round1(points.reduce((s, p) => s + p.ocNorm, 0) / points.length)
  const drMean  = round1(points.reduce((s, p) => s + p.drNorm, 0) / points.length)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  const markLineBase = {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed' as const, width: 1.5 },
    label: { show: true, position: 'end' as const, fontWeight: 'bold' as const, fontSize: 11, textBorderColor: CHART_BG, textBorderWidth: 3 },
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 64, left: 48, right: 72 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const idx = (arr[0] as { dataIndex: number }).dataIndex
        const p = points[idx]
        if (!p) return ''
        const lines = [
          `<strong>${escapeHtml(p.label.replace('\n', ' · '))}</strong>`,
          `${opts.ocLabel} : <b>${p.ocRaw.toFixed(2)}</b> (${p.ocNorm.toFixed(0)}% P80)`,
        ]
        if (showDr) lines.push(`${opts.drLabel} : <b>${p.drRaw.toFixed(2)}</b> (${p.drNorm.toFixed(0)}% P80)`)
        return lines.join('<br/>')
      },
    },
    legend: {
      data: showDr ? [opts.ocLabel, opts.drLabel] : [opts.ocLabel],
      textStyle: { color: tc.axisLabel },
      bottom: 0,
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: {
      ...axis,
      type: 'value',
      // Référence P80 = 100% ; on affiche les valeurs en %
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), formatter: (v: number) => `${v}%` },
    },
    series: [
      {
        name: opts.ocLabel,
        type: 'bar' as const,
        data: points.map((p) => ({ value: round1(p.ocNorm), itemStyle: { color: ocColor } })),
        barMaxWidth: 18,
        markLine: {
          ...markLineBase,
          lineStyle: { ...markLineBase.lineStyle, color: ocColor },
          label: { ...markLineBase.label, color: ocColor, formatter: `${opts.meanLabel} ${ocMean}%` },
          data: [{ yAxis: ocMean }],
        },
      },
      // Barres DR — retirées si la Résistance n'est pas calculable (h5) : sinon
      // toutes les barres seraient à 0 (DR=0 → drNorm planché à 0), un tracé faux.
      ...(showDr
        ? [{
            name: opts.drLabel,
            type: 'bar' as const,
            data: points.map((p) => ({ value: round1(p.drNorm), itemStyle: { color: drColor } })),
            barMaxWidth: 18,
            markLine: {
              ...markLineBase,
              lineStyle: { ...markLineBase.lineStyle, color: drColor },
              label: { ...markLineBase.label, color: drColor, formatter: `${opts.meanLabel} ${drMean}%` },
              data: [{ yAxis: drMean }],
            },
          }]
        : []),
      // Ligne de référence P80 à 100%
      {
        type: 'bar' as const,
        data: [],
        markLine: {
          silent: true,
          symbol: 'none' as const,
          lineStyle: { type: 'solid' as const, color: refColor, width: 1, opacity: 0.4 },
          label: { show: true, position: 'end' as const, formatter: opts.p80Label, color: refColor, fontSize: 10, textBorderColor: CHART_BG, textBorderWidth: 2 },
          data: [{ yAxis: 100 }],
        },
      },
    ],
  }
}

interface Props {
  title: ReactNode
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionOcdrBars({ title, matches, height = 280 }: Props) {
  const t = useSessionT()
  const ocP80 = useOffensiveConversionP80() // 0.90 Infinite / 1.264 h5 (titre courant)
  // false (Halo 5) → DR non calculable : barres DR retirées (cf. buildSessionOcdrBarsOption).
  const providesDamageTaken = useProvidesDamageTaken()

  const series = useMemo<ChartSeries<OcdrPoint>[]>(() => {
    const sorted = [...matches]
      .filter((m) => m.offensive_conversion != null || m.defensive_resistance != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    return [
      {
        key: 'ocdr',
        datapoints: sorted.map((m, i) => {
          const oc = m.offensive_conversion ?? 0
          const dr = m.defensive_resistance ?? 0
          return {
            label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
            ocNorm: (oc / ocP80) * 100,
            drNorm: Math.max(0, ((dr - 1.0) / DR_P80_EXCESS) * 100),
            ocRaw: oc,
            drRaw: dr,
          }
        }),
      },
    ]
  }, [matches, ocP80])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionOcdrBarsOption(s, {
          ocLabel: t('session.detail.ocdr_axis_oc'),
          drLabel: t('session.detail.ocdr_axis_dr'),
          p80Label: 'P80',
          meanLabel: t('session.detail.chart_perf_mean'),
          showDr: providesDamageTaken,
        })
      }
    />
  )
}
