/**
 * SquadEngagementView — Mock 15 v2 dans la Squad Page principale.
 *
 * Reference visuelle : .ai/mockups/engagement/engagement_visualizations.html (Mock 15 v2)
 * Plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §6.6.1 (Squad page) + §8.7 doc reflexion
 *
 * Format :
 *   - 3 courbes team-level toujours visibles (lobby pointille pale, attendu
 *     pointille marque, equipe pleine epaisse)
 *   - Chips sous le chart pour overlay 1 joueur a la fois (PlayerChips)
 *   - Click chip -> ajoute la courbe pace_observed du joueur en couleur saturee
 *   - Click chip active -> deselectionne
 *
 * Auto-zoom Y dynamique selon presence de l'overlay (cf §8.6 doc).
 *
 * Source de donnees : payload backend SquadEngagementSession
 * (a wirer en Phase 4.b — pour MVP la prop est typee mais la query peut etre
 * stub).
 */
import { useCallback, useMemo, useState } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, axisBase, legendBase, tooltipBase } from '@/components/charts/_utils'
import { PlayerChips, type PlayerChipItem } from '@/components/PlayerChips'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'

export interface SquadEngagementSession {
  /** Labels matchs (ex. M1, M2, ... ou date). */
  labels: string[]
  /** Pace lobby per_player par match. */
  lobbyPerPlayer: number[]
  /** Pace attendu equipe (mean coef squad × lobby). */
  teamExpected: number[]
  /** Pace equipe observee per_player. */
  teamObserved: number[]
  /** Joueurs du squad avec leur pace observe par match. */
  players: SquadPlayerEngagement[]
}

export interface SquadPlayerEngagement {
  xuid: string
  gamertag: string
  /** Pace observe par match (longueur = labels.length). */
  paceObserved: number[]
  /** Token couleur semantique pour la courbe + chip. */
  colorToken: SemanticToken
}

export interface SquadEngagementViewProps {
  /** Donnees de la session ou periode. */
  session: SquadEngagementSession
  /** Etat externe. */
  state?: 'loading' | 'error' | 'empty' | 'ready'
  /** Hauteur du chart. Default 280. */
  height?: number
}

/**
 * SquadEngagementView — Mock 15 v2 :
 *   3 courbes team-level + chips squad pour overlay 1 joueur a la fois.
 */
export function SquadEngagementView(props: SquadEngagementViewProps) {
  const { session, state = 'ready', height = 280 } = props
  const [selectedXUID, setSelectedXUID] = useState<string | null>(null)

  const chips: PlayerChipItem[] = useMemo(
    () =>
      session.players.map((p) => ({
        id: p.xuid,
        label: p.gamertag,
        colorToken: p.colorToken,
      })),
    [session.players],
  )

  const overlayPlayer = useMemo(
    () => session.players.find((p) => p.xuid === selectedXUID) ?? null,
    [session.players, selectedXUID],
  )

  const buildOption = useCallback(
    (): EChartsCoreOption => buildSquadEngagementOption(session, overlayPlayer),
    [session, overlayPlayer],
  )

  // Series virtuelle non vide (sinon ChartCard affiche emptyMessage).
  const dummySeries: ChartSeries<unknown>[] = useMemo(
    () => (session.labels.length > 0 ? [{ key: 'engagement', datapoints: session.labels }] : []),
    [session.labels],
  )

  return (
    <div>
      <ChartCard
        title="Engagement equipe"
        series={dummySeries}
        loading={state === 'loading'}
        error={state === 'error' ? new Error('error') : undefined}
        height={height}
        buildOption={buildOption}
      />
      <div style={{ marginTop: '8px', marginLeft: '50px' }}>
        <PlayerChips
          players={chips}
          selectedId={selectedXUID}
          onChange={setSelectedXUID}
          groupLabel="Afficher joueur :"
          ariaLabel="Selecteur de joueur a afficher"
        />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Builder ECharts
// ---------------------------------------------------------------------------

function buildSquadEngagementOption(
  session: SquadEngagementSession,
  overlay: SquadPlayerEngagement | null,
): EChartsCoreOption {
  if (session.labels.length === 0) {
    return {} as EChartsCoreOption
  }

  // Auto-zoom Y inclut l'overlay si present.
  const allY = [
    ...session.lobbyPerPlayer,
    ...session.teamExpected,
    ...session.teamObserved,
  ]
  if (overlay) {
    allY.push(...overlay.paceObserved)
  }
  const yMin = Math.floor(Math.min(...allY) - 1)
  const yMax = Math.ceil(Math.max(...allY) + 1)

  const lobbyColor = resolveToken('chart-series-1') // pale, en arriere-plan
  const expectedColor = resolveToken('chart-series-2') // medium, dashed marque
  const teamColor = resolveToken('chart-series-3') // gris fonce sature
  const overlayColor = overlay ? resolveToken(overlay.colorToken) : undefined

  type SeriesItem = {
    name: string
    type: 'line'
    data: number[]
    smooth: boolean
    symbol: string
    symbolSize: number
    lineStyle: { color: string; width: number; type?: 'dashed' | 'solid' }
    itemStyle: { color: string }
    z: number
  }
  const series: SeriesItem[] = [
    {
      name: 'Lobby',
      type: 'line',
      data: session.lobbyPerPlayer,
      smooth: false,
      symbol: 'circle',
      symbolSize: 4,
      lineStyle: { color: lobbyColor, width: 1.5, type: 'dashed' },
      itemStyle: { color: lobbyColor },
      z: 1,
    },
    {
      name: 'Attendu equipe',
      type: 'line',
      data: session.teamExpected,
      smooth: false,
      symbol: 'circle',
      symbolSize: 5,
      lineStyle: { color: expectedColor, width: 2, type: 'dashed' },
      itemStyle: { color: expectedColor },
      z: 2,
    },
    {
      name: 'Equipe observee',
      type: 'line',
      data: session.teamObserved,
      smooth: false,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { color: teamColor, width: 3 },
      itemStyle: { color: teamColor },
      z: 3,
    },
  ]

  if (overlay && overlayColor) {
    series.push({
      name: overlay.gamertag,
      type: 'line',
      data: overlay.paceObserved,
      smooth: false,
      symbol: 'circle',
      symbolSize: 9,
      lineStyle: { color: overlayColor, width: 4 },
      itemStyle: { color: overlayColor },
      z: 5,
    })
  }

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 24, top: 18, bottom: 38 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: { ...legendBase, top: 0, bottom: 'auto' },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: session.labels,
      axisLabel: { ...axisBase.axisLabel },
    },
    yAxis: {
      ...axisBase,
      type: 'value',
      min: yMin,
      max: yMax,
      name: `events / min (auto-zoom ${yMin}..${yMax})`,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: 'rgba(255,255,255,0.45)', fontSize: 10 },
    },
    series,
  }
}
