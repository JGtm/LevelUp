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
import { Suspense, lazy, useCallback, useMemo, useRef } from 'react'
import type { EChartsCoreOption, EChartsType } from 'echarts/core'

import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { resolveToken } from '@/lib/accessibility'
// Tokens de dominance : source unique partagée avec la colonne Dominance de
// l'Explorateur — même vocabulaire visuel, aucune palette nouvelle.
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { CHART_BG, escapeHtml, getEChartsThemeColors, getTooltipBase, outcomeColor } from './_utils'
import {
  type DominanceValue,
  type OutcomePoint,
  type Run,
  matchIndexAtX,
  startOf,
  toRuns,
} from './outcomeSequence'

/** Largeur minimale par match (px) sous laquelle le losange n'est plus lisible. */
const DOMINANCE_MIN_WIDTH_PX = 6
/** Demi-diagonale du losange (px). */
const DOMINANCE_HALF = 3.5

// Ré-export des types du modèle pur : les 5 consommateurs importent toujours
// `OutcomeValue`/`OutcomePoint` depuis ce module (frontière stable). La logique
// pure (`matchIndexAtX`) vit dans `outcomeSequence.ts` — importée directement là
// par les tests, jamais ré-exportée ici (react-refresh : pas d'export de valeur
// non-composant depuis un fichier de composant).
export type { OutcomeValue, OutcomePoint, Run, DominanceValue } from './outcomeSequence'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface OutcomeSequenceLabels {
  win: string
  loss: string
  tie: string
  dnf: string
}

export interface OutcomeSequenceTapeProps {
  matches: OutcomePoint[]
  labels: OutcomeSequenceLabels
  loading?: boolean
  height?: number
  /**
   * onMatchClick — si fourni, chaque match de la bande devient cliquable (curseur
   * `pointer`) et un clic remonte le `matchId` résolu. SANS cette prop, le rendu
   * et le comportement sont STRICTEMENT identiques (5 autres consommateurs).
   */
  onMatchClick?: (matchId: string) => void
  /**
   * Libellés des drapeaux de dominance (1..5) pour le tooltip. Absents → le
   * marqueur reste dessiné mais le tooltip n'ajoute rien : aucun consommateur
   * n'est obligé de fournir la traduction.
   */
  dominanceLabels?: Partial<Record<DominanceValue, string>>
}

export function OutcomeSequenceTape({
  matches,
  labels,
  loading = false,
  height = 100,
  onMatchClick,
  dominanceLabels,
}: OutcomeSequenceTapeProps) {
  const runs = useMemo(() => toRuns(matches), [matches])
  const xMax = runs.reduce((s, r) => s + r.count, 0)
  const themeVersion = useThemeVersion()
  const interactive = !!onMatchClick
  const chartRef = useRef<EChartsType | null>(null)

  // handleClick — résout le match via convertFromPixel (offset pixel → valeur X)
  // puis matchIndexAtX. Cas dégradé (conversion indisponible/échouée) : premier
  // match du run cliqué (params.dataIndex).
  const handleClick = useCallback(
    (params: unknown) => {
      if (!onMatchClick) return
      const p = params as {
        dataIndex?: number
        event?: { offsetX?: number; offsetY?: number }
      }
      let picked: OutcomePoint | null = null
      const chart = chartRef.current
      const offsetX = p.event?.offsetX
      const offsetY = p.event?.offsetY
      if (chart && offsetX != null && offsetY != null) {
        try {
          const converted = chart.convertFromPixel({ xAxisIndex: 0 }, [offsetX, offsetY])
          const xValue = Array.isArray(converted) ? converted[0] : converted
          if (typeof xValue === 'number' && Number.isFinite(xValue)) {
            picked = matchIndexAtX(runs, xValue)
          }
        } catch {
          picked = null
        }
      }
      if (!picked && typeof p.dataIndex === 'number') {
        picked = runs[p.dataIndex]?.matches[0] ?? null
      }
      if (picked) onMatchClick(picked.matchId)
    },
    [onMatchClick, runs],
  )


  const option = useMemo((): EChartsCoreOption => {
    if (xMax === 0) return {}
    // themeVersion force le recalcul quand le thème bascule : les couleurs
    // (CHART_BG, outcomeColor, getEChartsThemeColors) sont lues à l'appel via des
    // variables CSS, invisibles pour exhaustive-deps. Référence réelle pour garder
    // la dépendance légitime sans disable.
    void themeVersion
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
          const lines = r.matches.slice(0, 5).map((m) => {
            const base = m.label
              ? `· ${escapeHtml(m.label)}`
              : `· ${escapeHtml(m.map ?? m.matchId)}${m.mode ? ` (${escapeHtml(m.mode)})` : ''}`
            // Suffixe de dominance : seulement si le drapeau ET sa traduction
            // sont fournis (sinon la ligne reste strictement inchangée).
            const dom = m.dominance ? dominanceLabels?.[m.dominance] : undefined
            return dom ? `${base} — ${escapeHtml(dom)}` : base
          })
          if (r.matches.length > 5) lines.push(`+${r.matches.length - 5}`)
          return [`<b>${r.count}× ${escapeHtml(label)}</b>`, ...lines].join('<br/>')
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
            const rect: any = {
              type: 'rect',
              shape: { x: x0, y: yStripTop, width: w, height: STRIP_H, r: rectR },
              style: { fill: stroke },
            }
            // Curseur cliquable uniquement en mode interactif (B3) : sans
            // `onMatchClick`, la propriété reste absente → rendu inchangé.
            if (interactive) rect.cursor = 'pointer'
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const children: any[] = [rect]

            // Marqueurs de dominance (F1) : un losange centré sur la cellule du
            // match, DANS la bande (n'empiète pas sur les brackets de séries).
            // En dessous de DOMINANCE_MIN_WIDTH_PX par match, la bande est trop
            // dense : on n'affiche rien et le tooltip porte l'information.
            const perMatchW = (x1 - x0) / r.count
            if (perMatchW >= DOMINANCE_MIN_WIDTH_PX) {
              for (let k = 0; k < r.matches.length; k++) {
                const flag = r.matches[k].dominance
                if (!flag) continue
                const cx = x0 + perMatchW * (k + 0.5)
                children.push({
                  type: 'polygon',
                  shape: {
                    points: [
                      [cx, yCenter - DOMINANCE_HALF],
                      [cx + DOMINANCE_HALF, yCenter],
                      [cx, yCenter + DOMINANCE_HALF],
                      [cx - DOMINANCE_HALF, yCenter],
                    ],
                  },
                  style: {
                    fill: resolveToken(DOMINANCE_COLOR_TOKENS[flag]),
                    // Liseré couleur de fond de carte (var(--popover)) : effet
                    // « découpe » qui sépare le losange de la couleur d'outcome
                    // dans les deux thèmes (CHART_BG vaut 'transparent' et ne
                    // séparerait rien).
                    stroke: tc.tooltipBg,
                    lineWidth: 1,
                  },
                })
              }
            }

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
  }, [runs, xMax, labels, themeVersion, interactive, dominanceLabels])

  if (loading || xMax === 0) return null

  // Sans `onMatchClick`, aucun `onEvents`/`onChartReady` n'est passé : les 5 autres
  // consommateurs conservent un comportement strictement identique (B1).
  const interactiveProps = interactive
    ? {
        onEvents: { click: handleClick },
        // Param `unknown` (assignable par contravariance au type attendu par
        // echarts-for-react) puis cast : les deux paquets echarts exposent deux
        // déclarations nominales distinctes d'`EChartsType`.
        onChartReady: (chart: unknown) => {
          chartRef.current = chart as EChartsType
        },
      }
    : {}

  return (
    <Suspense fallback={null}>
      <ReactECharts
        option={option}
        style={{ height, width: '100%' }}
        notMerge
        lazyUpdate
        theme={undefined}
        {...interactiveProps}
      />
    </Suspense>
  )
}
