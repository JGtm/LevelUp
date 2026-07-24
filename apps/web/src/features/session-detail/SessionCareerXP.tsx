/**
 * SessionCareerXP — « XP de carrière (estimée) » sur les matchs d'une session
 * (V72-13). MIROIR du chart Timeseries (TimeseriesCareerXP,
 * features/timeseries/TimeseriesFormCharts.tsx) : ligne = XP de carrière
 * cumulée sur la session (trajectoire) ; barres = XP estimée gagnée par match
 * (axe secondaire). Cumul calculé par le helper PARTAGÉ `buildCareerXpSeries`
 * (lib/charts/careerXpSeries, type d'entrée structurel minimal) — AUCUNE
 * dépendance croisée vers la feature timeseries.
 *
 * Auto-gate DATA-DRIVEN (comme Timeseries) : le backend ne renseigne
 * career_xp_estimated que pour les titres à capability
 * analytics.career_xp_estimate (Halo Infinite) ; sans donnée sur la session,
 * le composant se masque silencieusement (pas de comparaison de slug côté
 * front, règle multi-titre CLAUDE.md).
 *
 * NON câblé dans SessionChartStack.tsx (fichier hors périmètre de cet agent,
 * V72-13) : à insérer par le superviseur juste après le bloc `{frags}`
 * (dernier bloc actuel de la pile), à l'identique du pattern `ocdr` (titre +
 * InfoTooltip construits au call-site, cf. SessionChartStack.tsx L166-176) :
 *
 *   const careerXp = (
 *     <SessionCareerXP
 *       title={
 *         <span className="flex items-center gap-1.5">
 *           {t('session.detail.career_xp_title')}
 *           <InfoTooltip content={t('session.detail.career_xp_tooltip')} />
 *         </span>
 *       }
 *       matches={matches}
 *     />
 *   )
 *   ...
 *   {frags}
 *   {careerXp}
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getGridBase,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { buildCareerXpSeries } from '@/lib/charts/careerXpSeries'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

/** Point par match : étiquette d'axe + XP par match + XP cumulée. */
interface CareerXpPoint {
  label: string
  perMatch: number | null
  cumulative: number | null
}

interface CareerXpOpts {
  cumulativeLabel: string
  perMatchLabel: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionCareerXpOption(
  series: ChartSeries<CareerXpPoint>[],
  opts: CareerXpOpts,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const colLine = resolveToken('chart-series-3')
  const colBar = resolveToken('chart-series-6')

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ right: 64, left: 64 }),
    tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
    legend: { ...getLegendBase(tc), bottom: 0 },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval: 0, fontSize: 9 },
    },
    yAxis: [
      {
        ...axis,
        type: 'value',
        name: opts.cumulativeLabel,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      },
      {
        ...axis,
        type: 'value',
        name: opts.perMatchLabel,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        type: 'bar',
        name: opts.perMatchLabel,
        yAxisIndex: 1,
        data: points.map((p) => p.perMatch),
        itemStyle: { color: colBar, opacity: 0.55 },
        barMaxWidth: 10,
      },
      {
        type: 'line',
        name: opts.cumulativeLabel,
        yAxisIndex: 0,
        data: points.map((p) => p.cumulative),
        showSymbol: false,
        smooth: false,
        connectNulls: true,
        areaStyle: { color: colLine, opacity: 0.15 },
        lineStyle: { color: colLine, width: 2 },
        z: 5,
      },
    ],
  }
}

interface Props {
  title: ReactNode
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionCareerXP({ title, matches, height = 280 }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<CareerXpPoint>[]>(() => {
    const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
    const { hasData, perMatch, cumulative } = buildCareerXpSeries(sorted)
    if (!hasData) return []
    return [
      {
        key: 'career_xp',
        datapoints: sorted.map((m, i) => ({
          label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
          perMatch: perMatch[i] ?? null,
          cumulative: cumulative[i] ?? null,
        })),
      },
    ]
  }, [matches])

  // Sans estimation sur la session (titre sans capability / que du Firefight) :
  // masquage silencieux, comme Timeseries (pas de carte vide).
  if (series.length === 0) return null

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionCareerXpOption(s, {
          cumulativeLabel: t('session.detail.career_xp_cumulative'),
          perMatchLabel: t('session.detail.career_xp_per_match'),
        })
      }
    />
  )
}
