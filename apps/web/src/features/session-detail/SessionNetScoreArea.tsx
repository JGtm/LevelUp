/**
 * SessionNetScoreArea — évolution du NET SCORE CUMULÉ (Frags − Morts) match après match.
 *
 * Aire + courbe DIVERGENTES : colorées par le SIGNE du solde cumulé via un DÉGRADÉ vert
 * (`divergent-pos`) au-dessus de 0 / rouge (`divergent-neg`) en dessous, à bascule EXACTE
 * sur la ligne 0. L'aire est ancrée à 0 (`areaStyle.origin`) ; la markLine matérialise le 0.
 *
 * PAS de visualMap : il rendait historiquement la courbe invisible (et une série scatter sur
 * axe catégoriel ne rendait pas non plus → graphe vide). À la place, le dégradé divergent
 * ancré à 0 provient du helper canonique `divergentZeroGradient` (CLAUDE.md n°6).
 *
 * Chaque match porte un POINT (le SYMBOLE de la ligne) coloré par son OUTCOME
 * (`outcome-win/loss/draw/dnf`) via `itemStyle` par point → l'issue se lit match par match,
 * indépendamment du signe du cumul. Une seule série (ligne + symboles), aucun scatter séparé.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  outcomeColor,
} from '@/components/charts/_utils'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { outcomeCodeToValue } from '@/lib/outcome'
import { sessionMatchAxisLabel, useSessionT } from './_shared'

type OutcomeKey = 'win' | 'loss' | 'tie' | 'dnf'

interface NetPoint {
  label: string
  cumulative: number
  /** Clé outcome canonique ou null — pilote la couleur du point. */
  outcomeKey: OutcomeKey | null
  /** Libellé outcome localisé (field mappings backend) pour le tooltip. */
  outcomeLabel: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionNetScoreOption(
  series: ChartSeries<NetPoint>[],
  opts: { seriesLabel: string; yDomain?: [number, number] },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  // Dégradé DIVERGENT vert/rouge à bascule EXACTE sur 0, SANS visualMap, ancré à 0
  // (helper canonique partagé — CLAUDE.md n°6). Le MÊME dégradé colore ligne ET aire.
  const values = points.map((p) => p.cumulative)
  const divergentColor = divergentZeroGradient(values)

  // Un point par match : `value` = solde cumulé (scalaire, aligné par index sur l'axe
  // catégoriel) ; `itemStyle.color` = OUTCOME → le SYMBOLE révèle l'issue, indépendamment
  // du signe. Pas de série scatter séparée (qui ne rendait pas sur l'axe catégoriel).
  const data = points.map((p) => ({
    value: p.cumulative,
    itemStyle: { color: outcomeColor(p.outcomeKey ?? undefined) },
  }))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const idx = params[0]?.dataIndex ?? 0
        const p = points[idx]
        if (!p) return ''
        const vStr = p.cumulative >= 0 ? `+${p.cumulative}` : `${p.cumulative}`
        const swatch = `<span style="display:inline-block;width:9px;height:9px;border-radius:9px;background:${outcomeColor(
          p.outcomeKey ?? undefined,
        )};margin-right:5px;vertical-align:middle"></span>`
        const outcomeRow = p.outcomeLabel ? `<br/>${swatch}${escapeHtml(p.outcomeLabel)}` : ''
        return `${escapeHtml(p.label.replace('\n', ' · '))}<br/>${opts.seriesLabel}: <b>${vStr}</b>${outcomeRow}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // Domaine Y figé en mode comparaison (échelle partagée A/B) ; sinon auto-scale. Le
    // dégradé reste ancré à 0 (calculé depuis la boîte de l'aire, indépendant de l'axe).
    yAxis: { ...axis, type: 'value', ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}) },
    series: [
      {
        name: opts.seriesLabel,
        type: 'line',
        data,
        symbol: 'circle',
        symbolSize: 7,
        showSymbol: true,
        // Ligne colorée par le SIGNE (dégradé divergent, bascule pile sur 0).
        lineStyle: { width: 2, color: divergentColor },
        // Aire ancrée à 0 + même dégradé → vert au-dessus, rouge en dessous.
        areaStyle: { color: divergentColor, opacity: 0.18, origin: 0 },
        // Anneau (couleur de surface) pour détacher le point outcome de l'aire colorée.
        itemStyle: { borderColor: tc.tooltipBg, borderWidth: 1.5 },
        emphasis: { scale: 1.6 },
        // Ligne de référence à 0 (séparation solde positif / négatif).
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          label: { show: false },
          data: [{ yAxis: 0 }],
        },
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

export function SessionNetScoreArea({ title, matches, height = 280, yDomain }: Props) {
  const t = useSessionT()
  const { data: fieldMappings } = useFieldMappings()

  const series = useMemo<ChartSeries<NetPoint>[]>(() => {
    const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    const outcomeLabel = (key: OutcomeKey | null): string =>
      key ? (fieldMappings?.outcomes?.[key]?.label ?? key) : ''
    let running = 0
    const datapoints = sorted.map((m, i) => {
      // Garde-fou : un kills/deaths manquant ne doit pas propager un NaN dans le cumul.
      running += (m.kills ?? 0) - (m.deaths ?? 0)
      const outcomeKey = outcomeCodeToValue(m.outcome)
      return {
        label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
        cumulative: running,
        outcomeKey,
        outcomeLabel: outcomeLabel(outcomeKey),
      }
    })
    return [{ key: 'net', datapoints }]
  }, [matches, fieldMappings])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionNetScoreOption(s, { seriesLabel: t('session.detail.net_score_series'), yDomain })}
    />
  )
}
