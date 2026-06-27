/**
 * OutcomeSequenceTape — bande séquentielle des outcomes de matchs.
 *
 * Affiche une piste horizontale unique colorée par outcome. Les streaks
 * consécutifs sont groupés par RLE et annotés d'un bracket I-beam :
 *   - wins/ties  → bracket au-dessus de la bande
 *   - losses/dnf → bracket en-dessous (miroir)
 *
 * Compact by design : hauteur ~100px, pas de card/border autour.
 * Aucune couleur hex directe : tout passe par outcomeColor() → resolveToken().
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { CHART_BG, getEChartsThemeColors, getTooltipBase, outcomeColor } from './_utils'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export type OutcomeValue = 'win' | 'loss' | 'tie' | 'dnf'

export interface OutcomePoint {
  outcome: OutcomeValue
  matchId: string
  map?: string
  mode?: string
  /** Libellé pré-formaté pour le tooltip (ex. « 12 mars · Slayer · Aquarius — 14/9 »). */
  label?: string
}

export interface OutcomeSequenceLabels {
  win: string
  loss: string
  tie: string
  dnf: string
}

interface Run {
  outcome: OutcomeValue
  count: number
  matches: OutcomePoint[]
}

function toRuns(arr: OutcomePoint[]): Run[] {
  const runs: Run[] = []
  for (const m of arr) {
    const last = runs[runs.length - 1]
    if (last && last.outcome === m.outcome) {
      last.count += 1
      last.matches.push(m)
    } else {
      runs.push({ outcome: m.outcome, count: 1, matches: [m] })
    }
  }
  return runs
}

function startOf(runs: Run[], i: number): number {
  let p = 0
  for (let k = 0; k < i; k++) p += runs[k].count
  return p
}

export interface OutcomeSequenceTapeProps {
  matches: OutcomePoint[]
  labels: OutcomeSequenceLabels
  loading?: boolean
  height?: number
}

export function OutcomeSequenceTape({
  matches,
  labels,
  loading = false,
  height = 100,
}: OutcomeSequenceTapeProps) {
  const runs = useMemo(() => toRuns(matches), [matches])
  const xMax = runs.reduce((s, r) => s + r.count, 0)
  const themeVersion = useThemeVersion()


  const option = useMemo((): EChartsCoreOption => {
    if (xMax === 0) return {}
    const tc = getEChartsThemeColors()
    return {
      backgroundColor: CHART_BG,
      grid: { top: 32, bottom: 32, left: 8, right: 8 },
      xAxis: { type: 'value', min: 0, max: xMax, show: false },
      yAxis: { type: 'value', min: -1, max: 1, show: false },
      tooltip: {
        trigger: 'item',
        ...getTooltipBase(tc),
        formatter: (p: unknown) => {
          const item = p as { data: { run: Run } }
          const r = item.data.run
          const label = labels[r.outcome]
          const lines = r.matches
            .slice(0, 5)
            .map((m) =>
              m.label ? `· ${m.label}` : `· ${m.map ?? m.matchId}${m.mode ? ` (${m.mode})` : ''}`,
            )
          if (r.matches.length > 5) lines.push(`+${r.matches.length - 5}`)
          return [`<b>${r.count}× ${label}</b>`, ...lines].join('<br/>')
        },
      },
      series: [
        {
          type: 'custom',
          renderItem: (params: unknown, api: unknown) => {
            const p = params as { dataIndex: number }
            const a = api as { coord: (v: number[]) => number[] }
            const i = p.dataIndex
            const r = runs[i]
            const x0 = a.coord([startOf(runs, i), 0])[0]
            const x1 = a.coord([startOf(runs, i) + r.count, 0])[0]
            const yCenter = a.coord([0, 0])[1]

            const STRIP_H = 14
            const BRACKET_GAP = 10
            const TICK_H = 4
            const APP_R = 8  // matches --radius: 0.5rem (card/KPI tile radius)
            const INNER_R = 2
            const yStripTop = yCenter - STRIP_H / 2
            const yStripBot = yCenter + STRIP_H / 2
            const w = Math.max(2, x1 - x0 - 1)
            const stroke = outcomeColor(r.outcome)
            const isTop = r.outcome === 'win' || r.outcome === 'tie'
            const isFirst = i === 0
            const isLast = i === runs.length - 1
            const rectR: [number, number, number, number] = [
              isFirst ? APP_R : INNER_R,
              isLast ? APP_R : INNER_R,
              isLast ? APP_R : INNER_R,
              isFirst ? APP_R : INNER_R,
            ]

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const children: any[] = [
              {
                type: 'rect',
                shape: { x: x0, y: yStripTop, width: w, height: STRIP_H, r: rectR },
                style: { fill: stroke },
              },
            ]

            if (r.count >= 2 && x1 - x0 >= 8) {
              const yLine = isTop
                ? yStripTop - BRACKET_GAP
                : yStripBot + BRACKET_GAP
              const tickEnd = isTop ? yLine + TICK_H : yLine - TICK_H
              const labelY = isTop ? yLine - 4 : yLine + 4
              const xL = x0 + 3
              const xR = x1 - 3
              const xMid = (xL + xR) / 2

              children.push(
                {
                  type: 'line',
                  shape: { x1: xL, y1: yLine, x2: xR, y2: yLine },
                  style: { stroke, lineWidth: 1 },
                },
                {
                  type: 'line',
                  shape: { x1: xL, y1: yLine, x2: xL, y2: tickEnd },
                  style: { stroke, lineWidth: 1 },
                },
                {
                  type: 'line',
                  shape: { x1: xR, y1: yLine, x2: xR, y2: tickEnd },
                  style: { stroke, lineWidth: 1 },
                },
                {
                  type: 'text',
                  style: {
                    text: 'x' + r.count,
                    x: xMid,
                    y: labelY,
                    fill: stroke,
                    fontSize: 11,
                    fontWeight: 700,
                    textAlign: 'center',
                    textBaseline: isTop ? 'bottom' : 'top',
                  },
                },
              )
            }

            return { type: 'group', children }
          },
          encode: { x: 0, y: 1 },
          data: runs.map((r, i) => ({ value: [i, 0], run: r, name: r.outcome })),
        },
      ],
    }
  }, [runs, xMax, labels, themeVersion])

  if (loading || xMax === 0) return null

  return (
    <Suspense fallback={null}>
      <ReactECharts
        option={option}
        style={{ height, width: '100%' }}
        notMerge
        lazyUpdate
        theme={undefined}
      />
    </Suspense>
  )
}
