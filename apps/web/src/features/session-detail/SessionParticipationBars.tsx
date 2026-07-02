/**
 * SessionParticipationBars — profil de participation (6 axes, normalisés 0..100) en
 * barres horizontales. Réutilise les axes `Participation` déjà calculés backend
 * (Combat / Survie / Support / Score / Objectif / Impact).
 *
 * Effet "papillon" en comparaison : l'axe des catégories est placé à DROITE en vue
 * single (`axisSide="right"`, barres vers la gauche) et à GAUCHE dans le drawer
 * (`axisSide="left"`, barres vers la droite). Côte à côte, les deux graphes se
 * reflètent autour des libellés centraux. Échelle fixe 0..100 → symétrie lisible.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SessionCompareEntry } from '@/lib/api/types'

import { useSessionT } from './_shared'
import { log } from './_logger'

interface ParticipationBar {
  label: string
  value: number
}

interface ParticipationOpts {
  axisSide: 'left' | 'right'
  colorToken: SemanticToken
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionParticipationBarsOption(
  series: ChartSeries<ParticipationBar>[],
  opts: ParticipationOpts,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const isRight = opts.axisSide === 'right'
  const barColor = resolveToken(opts.colorToken)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 12, bottom: 12, left: 8, right: 8, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const p = arr[0] as { name: string; value: number }
        return `${escapeHtml(p.name)}: <b>${p.value}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      inverse: isRight, // axe à droite → valeurs croissantes vers la gauche (barres vers la gauche)
      axisLabel: { show: false },
      splitLine: { show: false },
    },
    yAxis: {
      ...axis,
      type: 'category',
      position: opts.axisSide,
      data: points.map((p) => p.label),
      axisTick: { show: false },
    },
    series: [
      {
        type: 'bar',
        data: points.map((p) => p.value),
        barMaxWidth: 18,
        itemStyle: { color: barColor, borderRadius: 2 },
        label: {
          show: true,
          // Étiquette de valeur au bout de la barre (côté extérieur).
          position: isRight ? 'left' : 'right',
          color: tc.text,
          fontSize: 11,
          formatter: (p: { value: number }) => String(p.value),
        },
      },
    ],
  }
}

interface Props {
  title: string
  entry: SessionCompareEntry | null
  axisSide?: 'left' | 'right'
  colorToken?: SemanticToken
  height?: number
}

export function SessionParticipationBars({
  title,
  entry,
  axisSide = 'right',
  colorToken = 'compare-a',
  height = 280,
}: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<ParticipationBar>[]>(() => {
    const axes = entry?.participation
    if (!axes?.length) return []
    const labelOf = (name: string): string => {
      const key = `session.compare.participation_${name}` as Parameters<typeof t>[0]
      const resolved = t(key)
      // Si la clé n'existe pas (axe inconnu), retombe sur le nom brut.
      return resolved === key ? name : resolved
    }
    return [
      {
        key: 'participation',
        datapoints: axes.map((a) => ({ label: labelOf(a.name), value: Math.round(a.value) })),
      },
    ]
  }, [entry, t])

  // Observabilité : profil de participation absent de l'entry (non calculé backend).
  if (entry && !entry.participation?.length) {
    log.warn(
      `participation_empty:${entry.session_label ?? ''}`,
      'Profil de participation vide : aucun axe dans entry.participation (backend)',
    )
  }

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionParticipationBarsOption(s, { axisSide, colorToken })}
    />
  )
}
