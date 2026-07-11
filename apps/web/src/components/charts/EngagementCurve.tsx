/**
 * EngagementCurve — Mock 10 (intra-match) et Mock 11 (session/periode).
 *
 * Reference visuelle : .ai/mockups/engagement/engagement_visualizations.html
 * Plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §6.6.2 (Match View) + §6.6.3 (Timeseries)
 *
 * Trois courbes superposees :
 *   - Equipe alliee (gris fonce, fine)        — pace per_player
 *   - Attendu (gris medium, pointille)         — coef × team
 *   - Joueur (couleur saturee, epaisse 4px)   — pace observed
 *
 * Exigences §8.6 du doc reflexion (obligatoires) :
 *   - Auto-zoom Y dynamique (range affiche dans label)
 *   - Hierarchie visuelle marquee (joueur 4px sature, attendu 1.5px pointille,
 *     equipe 1.5px effacee)
 *
 * Rejetes explicitement : gap shading, pastille FormScore in-chart.
 *
 * Reutilise pour Match View intra-match (1 point = 10s, smooth=true) et pour
 * Session/Periode (1 point = 1 match, smooth=false avec markers visibles).
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase } from './_utils'

export interface EngagementPoint {
  /** Timestamp en ms (Match View) ou index match (Session) */
  x: number
  /** Pace equipe alliee per_player */
  paceTeam: number
  /** Pace attendu = coef × team */
  paceAttendu: number
  /** Pace joueur observe */
  paceJoueur: number
  /** Optionnel : pace lobby per_player (Mock 15 v2 / Match View context) */
  paceLobby?: number
  /** Mort post-creux > 30s (annotation triangle rouge) */
  isPassiveDeath?: boolean
  /** Periode post-mort active (bande rouge translucide) */
  postDeath?: boolean
}

export interface EngagementCurveProps {
  /** Titre affiche dans la ChartCard. */
  title?: string
  /** Subtitle / sous-titre optionnel. */
  subtitle?: string
  /** Donnees ordonnees chronologiquement. */
  points: EngagementPoint[]
  /** Format des labels X axis. */
  xFormatter?: (x: number) => string
  /** Mode de granularite : 'intra' (smooth) ou 'session' (lineaire avec markers). */
  granularity?: 'intra' | 'session'
  /** Etat externe (loading/error/empty). Cf ChartCard. */
  state?: 'loading' | 'error' | 'empty' | 'ready'
  /** Hauteur en px. Default 280 (chart-tall). */
  height?: number
  /**
   * Masque la serie "Attendu" quand l'attendu n'a pas de base fiable —
   * c'est-a-dire quand expected_basis === 'cold_start' (aucun historique
   * exploitable → coef 1.0 par defaut, l'attendu ne veut rien dire). Indexe
   * sur expected_basis (modele lobby-anchored v2), plus sur confidence (qui
   * qualifie l'historique du PERCENTILE, pas l'attendu).
   */
  hideAttendu?: boolean
  /**
   * Libellés localisés des courbes. Fournis par le parent (qui a la locale)
   * via le manifest engagement.trace.*. Défaut FR si absent.
   *   - team     : « Équipe réelle » (moyenne par joueur, joueur INCLUS)
   *   - expected : « Joueur attendu » (coef × pace lobby — réponse habituelle)
   *   - player   : « Joueur réel » (rythme observé du joueur)
   *   - lobby    : « Lobby » (pace lobby per-player, l'ancre — TOOLTIP seulement,
   *     pas dessiné, cf. D4 : 3 courbes max)
   */
  seriesLabels?: EngagementSeriesLabels
  /**
   * Message affiché quand state === 'error'. On ne montre JAMAIS d'erreur brute
   * (« Error ») : un échec de chargement engagement est rendu comme un état vide
   * neutre, conforme aux autres blocs Timeseries. Défaut FR si absent ; le parent
   * (qui a la locale) fournit la version localisée via le manifest.
   */
  errorMessage?: string
}

export interface EngagementSeriesLabels {
  team: string
  expected: string
  player: string
  /** Libellé de la ligne « Lobby » du tooltip (l'ancre, non dessinée). */
  lobby: string
}

const DEFAULT_SERIES_LABELS: EngagementSeriesLabels = {
  team: 'Équipe réelle',
  expected: 'Joueur attendu',
  player: 'Joueur réel',
  lobby: 'Lobby',
}

/**
 * EngagementCurve — composant generique reutilisable pour Match View
 * (intra-match) et Session (1 point = 1 match).
 */
export function EngagementCurve(props: EngagementCurveProps) {
  const {
    title,
    subtitle,
    points,
    xFormatter,
    granularity = 'intra',
    state = 'ready',
    height = 280,
    hideAttendu = false,
    seriesLabels = DEFAULT_SERIES_LABELS,
    errorMessage,
  } = props

  const buildOption = useCallback(
    (series: ChartSeries<EngagementPoint>[]): EChartsCoreOption => {
      const pts = series[0]?.datapoints ?? points
      return buildEngagementOption(pts, granularity, xFormatter, hideAttendu, seriesLabels)
    },
    [points, granularity, xFormatter, hideAttendu, seriesLabels],
  )

  // Series typee pour ChartCard. On la laisse vide quand state='empty'/'error'
  // OU quand `points` est vide pour que ChartCard rende son emptyMessage au
  // lieu d'un canvas blanc sans contexte.
  const isError = state === 'error'
  const series: ChartSeries<EngagementPoint>[] =
    state === 'empty' || isError || points.length === 0
      ? []
      : [{ key: 'engagement', datapoints: points }]

  // Sur erreur : état vide NEUTRE avec message humain (jamais « Error » brut),
  // conforme aux autres blocs Timeseries. On ne passe PAS `error` à ChartCard
  // (qui rendrait son habillage d'erreur) — on dégrade en empty.
  const emptyMessage = isError
    ? (errorMessage ?? 'Engagement momentanément indisponible')
    : (subtitle ?? "Aucune donnée d'engagement pour ce match")

  return (
    <ChartCard
      title={title}
      series={series}
      loading={state === 'loading'}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
    />
  )
}

// ---------------------------------------------------------------------------
// Builders ECharts
// ---------------------------------------------------------------------------

function buildEngagementOption(
  points: EngagementPoint[],
  granularity: 'intra' | 'session',
  xFormatter?: (x: number) => string,
  hideAttendu = false,
  labels: EngagementSeriesLabels = DEFAULT_SERIES_LABELS,
): EChartsCoreOption {
  if (points.length === 0) {
    return {} as EChartsCoreOption
  }

  // Auto-zoom Y : couvre toutes les traces avec padding ±1.
  const allY = points.flatMap((p) => [p.paceTeam, p.paceAttendu, p.paceJoueur])
  const yMin = Math.floor(Math.min(...allY) - 1)
  const yMax = Math.ceil(Math.max(...allY) + 1)

  const xData = points.map((p) => (xFormatter ? xFormatter(p.x) : String(p.x)))
  const teamColor = resolveToken('chart-series-1') // gris fonce / atone
  const expectedColor = resolveToken('chart-series-2') // gris medium pointille
  const playerColor = resolveToken('info') // saturated blue

  // Smooth=true en intra (echantillonnage 10s), false en session (matchs discrets).
  const smooth = granularity === 'intra'
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 24, top: 18, bottom: 56 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      // Pace = événements/min, rate continu → arrondi au centième pour la
      // lisibilité (demande UX 2026-05-07 — éviter les 0.6234567).
      // Pattern standard du codebase : formatter qui reçoit le tableau de
      // params axis-trigger (cf. BarStackedChart, OutcomeSequenceTape).
      formatter: (params: unknown) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const items = params as Array<{
          axisValueLabel?: string
          seriesName?: string
          marker?: string
          value?: number | null
          dataIndex?: number
        }>
        const head = escapeHtml(items[0]?.axisValueLabel ?? '')
        const lines = items.map((p) => {
          const v = typeof p.value === 'number' && Number.isFinite(p.value) ? p.value.toFixed(2) : '—'
          return `${p.marker ?? ''} ${escapeHtml(p.seriesName ?? '')}: <b>${v}</b>`
        })
        // Ligne « Lobby » (l'ancre de l'attendu) : non dessinée (D4 : 3 courbes
        // max) mais exposée dans le tooltip. Récupérée par dataIndex sur les
        // points d'origine (paceLobby n'est pas une série ECharts).
        const idx = items[0]?.dataIndex
        const lobby = typeof idx === 'number' ? points[idx]?.paceLobby : undefined
        if (typeof lobby === 'number' && Number.isFinite(lobby)) {
          lines.push(`${escapeHtml(labels.lobby)}: <b>${lobby.toFixed(2)}</b>`)
        }
        return [head, ...lines].join('<br/>')
      },
    },
    // Légende en BAS (comme tous les autres graphes) + pastille élargie
    // (itemWidth 30) pour distinguer trait plein (Joueur/Équipe) vs pointillé
    // (Attendu). getLegendBase fournit déjà bottom: 0.
    legend: { ...getLegendBase(tc), itemWidth: 30 },
    xAxis: {
      ...axis,
      type: 'category',
      data: xData,
      axisLabel: { ...axis.axisLabel, interval: computeXInterval(xData.length) },
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
    series: [
      // Equipe reelle (joueur INCLUS) — fine effacee
      {
        name: labels.team,
        type: 'line',
        data: points.map((p) => p.paceTeam),
        smooth,
        symbol: granularity === 'session' ? 'circle' : 'none',
        symbolSize: 5,
        lineStyle: { color: teamColor, width: 1.5 },
        itemStyle: { color: teamColor },
        z: 1,
      },
      // Joueur attendu — pointille fin (masque au cold-start : coef=1.0 => superposition avec Equipe reelle)
      ...(!hideAttendu
        ? [
            {
              name: labels.expected,
              type: 'line' as const,
              data: points.map((p) => p.paceAttendu),
              smooth,
              symbol: granularity === 'session' ? 'circle' : 'none',
              symbolSize: 5,
              lineStyle: { color: expectedColor, width: 1.5, type: 'dashed' as const },
              itemStyle: { color: expectedColor },
              z: 2,
            },
          ]
        : []),
      // Joueur reel — epais sature (hierarchie visuelle marquee §8.6)
      {
        name: labels.player,
        type: 'line',
        data: points.map((p) => p.paceJoueur),
        smooth,
        symbol: granularity === 'session' ? 'circle' : 'none',
        symbolSize: granularity === 'session' ? 8 : 0,
        lineStyle: { color: playerColor, width: 4 },
        itemStyle: { color: playerColor },
        z: 5,
      },
    ],
  }
}

function computeXInterval(n: number): number {
  if (n <= 12) return 0
  if (n <= 30) return 1
  return Math.ceil(n / 15) - 1
}
