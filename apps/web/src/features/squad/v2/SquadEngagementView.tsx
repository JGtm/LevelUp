/**
 * SquadEngagementView — Mock 15 v2 dans la Squad Page principale.
 *
 * Reference visuelle : .ai/mockups/engagement/engagement_visualizations.html (Mock 15 v2)
 * Plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §6.6.1 (Squad page) + §8.7 doc reflexion
 *
 * Format :
 *   - 3 courbes team-level toujours visibles (lobby pointille pale, attendu
 *     pointille marque, equipe pleine epaisse)
 *   - Boutons joueurs en bas de la card pour overlay 1 joueur a la fois
 *   - Click bouton -> ajoute la courbe pace_observed du joueur en couleur saturee
 *   - Click bouton actif -> deselectionne
 *
 * Auto-zoom Y dynamique selon presence de l'overlay (cf §8.6 doc).
 *
 * Source de donnees : payload backend SquadEngagementSession.
 */
import { useCallback, useMemo, useState } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'

export interface SquadEngagementSession {
  /** Labels matchs (ex. M1, M2, ... ou date). */
  labels: string[]
  /** Noms de carte par match (parallèle à labels). */
  mapNames: string[]
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
  /** Token couleur semantique pour la courbe + bouton (fallback si colorHex absent). */
  colorToken: SemanticToken
  /** Couleur hex directe — prioritaire sur colorToken (ex. couleur joueur depuis playerColors). */
  colorHex?: string
}

export interface SquadEngagementSeriesLabels {
  lobby: string
  /** Attendu de l'escouade (coef squad × lobby). */
  expected: string
  /** Rythme réel de l'escouade (per-player, joueur inclus). */
  observed: string
}

const DEFAULT_SQUAD_LABELS: SquadEngagementSeriesLabels = {
  lobby: 'Partie',
  expected: 'Escouade attendue',
  observed: 'Escouade réelle',
}

export interface SquadEngagementViewProps {
  /** Donnees de la session ou periode. */
  session: SquadEngagementSession
  /** Etat externe. */
  state?: 'loading' | 'error' | 'empty' | 'ready'
  /** Hauteur du chart. Default 280. */
  height?: number
  /** Libellés localisés des 3 courbes (défaut FR). Fournis par le parent. */
  seriesLabels?: SquadEngagementSeriesLabels
}

/**
 * SquadEngagementView — Mock 15 v2 :
 *   3 courbes team-level + boutons squad pour overlay 1 joueur a la fois.
 */
export function SquadEngagementView(props: SquadEngagementViewProps) {
  const { session, state = 'ready', height = 280, seriesLabels = DEFAULT_SQUAD_LABELS } = props
  const [selectedXUID, setSelectedXUID] = useState<string | null>(null)

  const overlayPlayer = useMemo(
    () => session.players.find((p) => p.xuid === selectedXUID) ?? null,
    [session.players, selectedXUID],
  )

  const buildOption = useCallback(
    (): EChartsCoreOption => buildSquadEngagementOption(session, overlayPlayer, seriesLabels),
    [session, overlayPlayer, seriesLabels],
  )

  // Series virtuelle non vide (sinon ChartCard affiche emptyMessage).
  const dummySeries: ChartSeries<unknown>[] = useMemo(
    () => (session.labels.length > 0 ? [{ key: 'engagement', datapoints: session.labels }] : []),
    [session.labels],
  )

  return (
    <ChartCard
      title="Engagement"
      series={dummySeries}
      loading={state === 'loading'}
      error={state === 'error' ? new Error('error') : undefined}
      height={height}
      buildOption={buildOption}
    >
      {session.players.length > 0 && (
        <div className="flex flex-wrap gap-1 px-3 pb-3">
          {session.players.map((p) => {
            const isActive = p.xuid === selectedXUID
            // color-allow: hex résolu depuis semantic tokens via colorHex ou resolveToken
            const accentHex = p.colorHex ?? resolveToken(p.colorToken)
            return (
              <button
                key={p.xuid}
                type="button"
                onClick={() => setSelectedXUID(isActive ? null : p.xuid)}
                aria-pressed={isActive}
                className={[
                  'inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors',
                  isActive
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-input bg-background hover:bg-muted',
                ].join(' ')}
              >
                <span
                  aria-hidden
                  style={{ background: accentHex, width: 8, height: 8, display: 'inline-block', flexShrink: 0 }}
                />
                {p.gamertag}
              </button>
            )
          })}
        </div>
      )}
    </ChartCard>
  )
}

// ---------------------------------------------------------------------------
// Builder ECharts
// ---------------------------------------------------------------------------

function truncateMapName(s: string, max = 14): string {
  if (s.length <= max) return s
  const sepIdx = Math.min(
    ...[' ', '-'].map((c) => { const i = s.indexOf(c); return i > 0 ? i : Infinity }),
  )
  if (sepIdx <= max) return `${s.slice(0, sepIdx)}…`
  return `${s.slice(0, max - 1)}…`
}

function buildSquadEngagementOption(
  session: SquadEngagementSession,
  overlay: SquadPlayerEngagement | null,
  labels: SquadEngagementSeriesLabels = DEFAULT_SQUAD_LABELS,
): EChartsCoreOption {
  if (session.labels.length === 0) {
    return {} as EChartsCoreOption
  }

  // Étiquettes X sur 2 lignes : "#N\nMapName" comme les charts timeseries.
  const xLabels = session.labels.map((label, i) => {
    const num = label.replace(/^M/, '#')
    const mapName = session.mapNames[i]
    return mapName ? `${num}\n${truncateMapName(mapName)}` : num
  })

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
  const overlayColor = overlay ? (overlay.colorHex ?? resolveToken(overlay.colorToken)) : undefined

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
      name: labels.lobby,
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
      name: labels.expected,
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
      name: labels.observed,
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

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    // grid.bottom = espace pour x-axis 2 lignes + légende en bas du container
    grid: { left: 50, right: 24, top: 8, bottom: 64 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      valueFormatter: (v: unknown) =>
        typeof v === 'number' ? v.toFixed(2) : String(v),
    },
    // legendBase a déjà bottom: 0 — pas d'override top
    legend: { ...getLegendBase(tc) },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: { ...axis.axisLabel, interval: 0 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: yMin,
      max: yMax,
      name: `events / min (auto-zoom ${yMin}..${yMax})`,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series,
  }
}
