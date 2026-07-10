/**
 * SessionEngagementChart — engagement par match en barres (option retenue par l'utilisateur).
 *
 * `engagement_score` est un RÉSIDU DE RYTHME : (rythme d'événements du joueur − rythme
 * attendu), en ÉVÉNEMENTS PAR MINUTE. + = sur-engagement vs attendu, − = sous-engagement.
 * Centré sur 0 → barres colorées par signe (divergent-pos/neg) + markLine de moyenne.
 * Axe X "#N + carte" aligné sur "Score de performance" ; axe Y étiqueté pour la lisibilité.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

const round2 = (n: number) => Math.round(n * 100) / 100

interface EngagementPoint {
  label: string
  value: number
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  entry: SessionCompareEntry | null
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionEngagementOption(
  series: ChartSeries<EngagementPoint>[],
  opts: { meanLabel: string; axisName: string; yDomain?: [number, number] },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const mean = round2(points.reduce((acc, p) => acc + p.value, 0) / points.length)
  const meanColor = resolveToken(mean >= 0 ? 'divergent-pos' : 'divergent-neg')
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  return {
    backgroundColor: CHART_BG,
    // Grille alignée sur "Score de performance" (SessionPerfTrend) pour un rendu identique.
    grid: { top: 24, bottom: 64, left: 48, right: 72 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      // L'unité (évén./min vs attendu) vit dans le tooltip plutôt qu'en nom d'axe vertical,
      // pour garder le même style épuré que "Score de performance".
      formatter: (params: Array<{ name: string; value: number; marker: string }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = params[0]
        return `${escapeHtml(p.name.replace('\n', ' · '))}<br/>${p.marker} ${opts.axisName}: <b>${p.value}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // Domaine Y figé en mode comparaison (échelle partagée A/B) ; sinon auto-scale.
    yAxis: { ...axis, type: 'value', ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}) },
    series: [
      {
        type: 'bar',
        data: points.map((p) => ({
          value: round2(p.value),
          itemStyle: { color: resolveToken(p.value >= 0 ? 'divergent-pos' : 'divergent-neg') },
        })),
        barMaxWidth: 28,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: meanColor, type: 'dashed', width: 2 },
          label: {
            show: true,
            position: 'end',
            formatter: `${opts.meanLabel} ${mean}`,
            color: meanColor,
            fontWeight: 'bold',
            textBorderColor: CHART_BG,
            textBorderWidth: 3,
          },
          data: [{ yAxis: mean }],
        },
      },
    ],
  }
}

export function SessionEngagementChart({ title, matches, entry, height = 260, yDomain }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<EngagementPoint>[]>(() => {
    const ms = [...(entry?.match_series ?? [])].sort((a, b) => a.index - b.index)
    if (ms.length === 0) return []
    // Les matchs (pour la carte) sont zippés par ordre chronologique avec la série.
    const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
    const points: EngagementPoint[] = []
    ms.forEach((p, i) => {
      if (p.engagement_score == null) return
      const m = sorted[i]
      points.push({
        label: sessionMatchAxisLabel(i, m?.map_name, m?.pair_name),
        value: p.engagement_score,
      })
    })
    return points.length ? [{ key: 'engagement', datapoints: points }] : []
  }, [matches, entry])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionEngagementOption(s, {
          meanLabel: t('session.detail.chart_perf_mean'),
          axisName: t('session.detail.engagement_axis'),
          yDomain,
        })
      }
    />
  )
}
