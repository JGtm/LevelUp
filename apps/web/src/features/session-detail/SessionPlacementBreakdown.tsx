/**
 * SessionPlacementBreakdown — répartition des placements du joueur sur la session.
 *
 * X = tous les placements possibles (1..N), Y = nombre de matchs finis à ce placement.
 * Placement = `placement` (rang API, "Rang" du scoreboard). La borne N = taille de
 * lobby la plus FRÉQUENTE sur la session (compte des présents à la fin, bots inclus,
 * fourni par le backend), pour rester cohérent malgré le churn (départs/arrivées) et
 * les valeurs aberrantes d'un match isolé. Fallback : max(placement observé) si les
 * tailles de lobby ne sont pas disponibles. Bars colorés du meilleur (vert) au pire (rouge).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { useSessionT } from './_shared'
import { log } from './_logger'

interface PlacementPoint {
  placement: number
  count: number
}

/** Couleur du bâton selon la qualité du placement : top tiers vert, milieu neutre, bas rouge. */
function placementToken(index: number, total: number): SemanticToken {
  if (total <= 1) return 'divergent-pos'
  const r = index / (total - 1)
  if (r < 0.34) return 'divergent-pos'
  if (r < 0.67) return 'divergent-neutral'
  return 'divergent-neg'
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionPlacementOption(
  series: ChartSeries<PlacementPoint>[],
  opts: { countLabel: string; yMax?: number },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const n = points.length
  // Même logique d'intervalle que SessionNetScoreArea (axe X "#N") : tout afficher
  // sous 30 graduations, espacer au-delà. Garde les deux axes "#N" cohérents.
  const interval = n > 30 ? Math.floor(n / 12) : 0

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 32, left: 40, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const p = arr[0] as { name: string; value: number }
        return `${p.name}: <b>${p.value}</b> ${opts.countLabel}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => `#${p.placement}`),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // max figé en mode comparaison (compte partagé A/B) → hauteurs de barres comparables.
    yAxis: { ...axis, type: 'value', minInterval: 1, ...(opts.yMax != null ? { max: opts.yMax } : {}) },
    series: [
      {
        type: 'bar',
        data: points.map((p, i) => ({ value: p.count, itemStyle: { color: resolveToken(placementToken(i, n)) } })),
        barMaxWidth: 48,
        label: {
          show: true,
          position: 'top',
          color: tc.text,
          formatter: (p: { value: number }) => (p.value > 0 ? String(p.value) : ''),
        },
      },
    ],
  }
}

/** Valeur la plus fréquente d'une liste (tie-break : la plus grande). Exporté pour test. */
// eslint-disable-next-line react-refresh/only-export-components
export function modalValue(values: number[]): number | null {
  if (values.length === 0) return null
  const counts = new Map<number, number>()
  for (const v of values) counts.set(v, (counts.get(v) ?? 0) + 1)
  let best: number | null = null
  let bestCount = -1
  for (const [v, c] of counts) {
    if (c > bestCount || (c === bestCount && best != null && v > best)) {
      best = v
      bestCount = c
    }
  }
  return best
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Max du compte (axe Y) partagé A/B en mode comparaison (sinon auto-scale). */
  yMax?: number
  /** Nb de placements (axe X #1..N) imposé pour aligner A/B (sinon calcul local). */
  axisMaxOverride?: number
}

export function SessionPlacementBreakdown({ title, matches, height = 260, yMax, axisMaxOverride }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<PlacementPoint>[]>(() => {
    const placements = matches.map((m) => m.placement).filter((p): p is number => p != null && p > 0)
    if (placements.length === 0) return []
    const lobbySizes = matches.map((m) => m.lobby_size).filter((n): n is number => n != null && n > 0)
    const modalLobby = modalValue(lobbySizes) ?? 0
    const maxObserved = Math.max(...placements)
    // Axe = override partagé A/B (comparaison) sinon taille de lobby modale, jamais en-deçà
    // du pire placement réellement atteint.
    const axisMax = axisMaxOverride ?? Math.max(modalLobby, maxObserved)
    if (axisMax <= 0) return []
    const counts = new Array<number>(axisMax).fill(0)
    for (const p of placements) {
      if (p >= 1 && p <= axisMax) counts[p - 1] += 1
    }
    return [
      {
        key: 'placement',
        datapoints: counts.map((count, i) => ({ placement: i + 1, count })),
      },
    ]
  }, [matches, axisMaxOverride])

  // Observabilité : distingue "aucun placement" (rang non peuplé) de "axe = max
  // observé" (taille de lobby present_at_completion indisponible → fallback).
  const sLabel = matches[0]?.session_label ?? ''
  const hasPlacement = matches.some((m) => m.placement != null && m.placement > 0)
  const hasLobby = matches.some((m) => m.lobby_size != null && m.lobby_size > 0)
  if (matches.length > 0 && !hasPlacement) {
    log.warn(`placement_missing:${sLabel}`, 'Breakdown placements vide : aucun rang (placement) sur la session')
  } else if (hasPlacement && !hasLobby) {
    log.warn(
      `lobby_size_fallback:${sLabel}`,
      'Breakdown placements : taille de lobby (present_at_completion) indisponible → axe = max(rang observé)',
    )
  }

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionPlacementOption(s, { countLabel: t('session.detail.mode_breakdown_count'), yMax })}
    />
  )
}
