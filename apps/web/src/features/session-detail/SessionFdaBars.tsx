/**
 * SessionFdaBars — barres Frags / Morts / Assists d'une session, en deux modes :
 *  - mode="game"   : moyenne PAR PARTIE (total / nb de matchs)
 *  - mode="minute" : taux PAR MINUTE (total / minutes jouées) — équivalent "session"
 *    du graphe "Stats par minute" de la page Escouade.
 *
 * 3 barres verticales colorées par stat (Frags→outcome-win, Morts→outcome-loss,
 * Assists→outcome-draw), cohérentes avec le graphe FDA. ChartCard + buildOption custom.
 * (Layout simple à 3 barres plutôt que le layout symétrique multi-joueurs de l'Escouade,
 *  plus lisible pour une seule session.)
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionDetailMatchRow } from '@/lib/api/types'

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  mode: 'game' | 'minute'
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

interface FdaPoint {
  key: 'frags' | 'deaths' | 'assists'
  label: string
  value: number
}

const STAT_TOKEN: Record<FdaPoint['key'], SemanticToken> = {
  frags: 'outcome-win',
  deaths: 'outcome-loss',
  assists: 'outcome-draw',
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionFdaBarsOption(
  series: ChartSeries<FdaPoint>[],
  opts: { decimals: number; yDomain?: [number, number] },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const fmt = (n: number) => Number(n.toFixed(opts.decimals))
  const abs = (v: number) => Math.abs(v).toFixed(opts.decimals)

  // Visuel page Escouade : Frags/Assists au-dessus de l'axe zéro (positifs),
  // Morts en dessous (négatifs). Axe X accentué (= ligne zéro), labels en valeur
  // absolue. Couleurs sémantiques par stat (win/loss/draw) plutôt que complément
  // joueur (une seule session → pas de mapping joueur→couleur à préserver).
  const data = points.map((p) => ({
    value: fmt(p.key === 'deaths' ? -p.value : p.value),
    itemStyle: { color: resolveToken(STAT_TOKEN[p.key]) },
  }))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 44, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const p = arr[0] as { name: string; value: number }
        return `${escapeHtml(p.name)}: <b>${abs(p.value)}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLine: { lineStyle: { color: tc.text, width: 2 } }, // ligne zéro accentuée (foreground)
    },
    yAxis: {
      ...axis,
      type: 'value',
      // Domaine figé en mode comparaison (échelle partagée A/B) ; sinon auto-scale.
      ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), formatter: (v: number) => abs(v) },
    },
    series: [
      {
        type: 'bar',
        data,
        barMaxWidth: 56,
        label: {
          show: true,
          position: 'top',
          color: tc.text,
          formatter: (p: { value: number }) => abs(p.value),
        },
      },
    ],
  }
}

export function SessionFdaBars({ title, matches, mode, height = 260, yDomain }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields

  const series = useMemo<ChartSeries<FdaPoint>[]>(() => {
    const lbl = (k: string): string => fields?.[k]?.label ?? k
    const rows = mode === 'minute' ? matches.filter((m) => (m.duration_seconds ?? 0) > 0) : matches
    if (rows.length === 0) return []
    const totals = rows.reduce(
      (acc, m) => {
        acc.kills += m.kills
        acc.deaths += m.deaths
        acc.assists += m.assists
        acc.minutes += (m.duration_seconds ?? 0) / 60
        return acc
      },
      { kills: 0, deaths: 0, assists: 0, minutes: 0 },
    )
    const denom = mode === 'minute' ? totals.minutes : rows.length
    if (denom <= 0) return []
    return [
      {
        key: 'fda',
        datapoints: [
          { key: 'frags', label: lbl('kills'), value: totals.kills / denom },
          { key: 'deaths', label: lbl('deaths'), value: totals.deaths / denom },
          { key: 'assists', label: lbl('assists'), value: totals.assists / denom },
        ],
      },
    ]
  }, [matches, mode, fields])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionFdaBarsOption(s, { decimals: mode === 'minute' ? 2 : 1, yDomain })}
    />
  )
}
