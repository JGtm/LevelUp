/**
 * MatchScoreCurveChart — LE SCORE DANS LE TEMPS, décodé du film du match.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. « Dominance » montre qui tue qui, « Frags cumulés »
 * qui marque des points ; aucun des deux ne montre LE SCORE — celui du mode, celui qui
 * décide du match. Un 3-0 en capture de drapeau et un 43-50 en Slayer se lisaient jusqu'ici
 * par déduction. Le calque de score de l'artefact de rejeu (schéma 12) le publie enfin, et
 * c'est cette carte qui le pose.
 *
 * ELLE N'APPARAÎT QUE SI L'ARTEFACT EXISTE, et par le MÊME chemin que le lien « rejeu » :
 * même endpoint, même clé de cache (`match-replay`), même présence servie par l'API
 * (`header.replay_available`). Sans artefact — la quasi-totalité des matchs aujourd'hui —
 * la carte ne rend RIEN : pas de cadre vide, pas de « bientôt disponible ». Décision D2 du
 * plan d'exploitation du registre : aucune table nouvelle, la donnée vit dans l'artefact.
 *
 * L'UNITÉ EST CELLE DU JEU, ET RIEN N'EST RECALCULÉ ICI. `coverage.score.oracle` vaut
 * `displayed` : les valeurs publiées SONT le score affiché à l'écran, confronté à l'oracle
 * lors de la cuisson. La carte les trace telles quelles. Quand le mode ne porte pas de
 * compteur (`modeSupported: false`) elle s'efface ; quand la lecture du film s'est arrêtée
 * avant la fin (`truncated`), elle le DIT — une courbe tronquée qui a l'air complète est
 * pire qu'une courbe absente.
 *
 * La projection (paliers, bornes, retournements) vit dans `_scoreCurve.ts`, pur et testé.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  hoverRevealSymbol,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import { useMatchReplay } from '@/features/match-replay/queries'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { allyOfTeamId, scoreTimelineOf } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { buildScoreCurve, formatClock, teamIdsOf, type ScoreCurve, type ScoreCurveSeries } from './_scoreCurve'
import type { MatchViewText } from './i18n'
import { resolveXuidMeta } from './xuidMeta'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — la présence de l'artefact, le même gate que le lien rejeu. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

export function MatchScoreCurveChart({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  meXUID,
  t,
}: Props) {
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const xuidMeta = useMemo(() => resolveXuidMeta(board, meXUID), [board, meXUID])
  // La garde d'horloge du rejeu s'applique ICI AUSSI : un calque daté depuis une origine
  // non résolue placerait chaque but au mauvais instant (cf. lib/replay/scoreTimeline).
  const timeline = useMemo(() => (data ? scoreTimelineOf(data) : undefined), [data])
  const curve = useMemo(
    () =>
      buildScoreCurve({
        timeline,
        frameIntervalMs: data?.frameIntervalMs,
        frameCount: data?.frameCount ?? 0,
        teamIds: teamIdsOf(
          board.map((r) => parseTeamSideID(r.team_side)),
          timeline,
        ),
        allyOf: (teamId) => allyOfTeamId(board, xuidMeta, teamId),
        labelOf: (teamId) =>
          resolveTeamLabel(
            board.filter((r) => r.team_side === `t${teamId}`),
            `t${teamId}`,
            t,
          ),
      }),
    [timeline, data?.frameIntervalMs, data?.frameCount, board, xuidMeta, t],
  )

  const buildOption = useCallback(
    (s: ChartSeries<ScoreCurveSeries>[]): EChartsCoreOption => {
      if (s.length === 0 || !curve) return { backgroundColor: CHART_BG }
      return scoreCurveOption(curve, getEChartsThemeColors(), t)
    },
    [curve, t],
  )

  // Pas d'artefact, pas de calque, ou mode sans compteur : RIEN. Pas de cadre vide.
  if (!curve || data?.coverage?.score?.modeSupported === false) return null

  return (
    <ChartCard
      title={t.scoreCurveTitle}
      series={[{ key: 'match_view.score_curve', datapoints: curve.series }]}
      height={260}
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    >
      <p className="px-3 pb-2 text-[11px] text-muted-foreground">
        {t.scoreCurveSource}
        {data?.coverage?.score?.truncated ? ` ${t.scoreCurveTruncated}` : ''}
      </p>
    </ChartCard>
  )
}

/** La couleur d'une équipe : ses tokens allié/adverse, encre neutre quand le camp est inconnu. */
function colorOf(serie: ScoreCurveSeries, tc: EChartsThemeColors): string {
  if (serie.ally === null) return tc.axisLabel
  return resolveToken(serie.ally ? 'team-ally' : 'team-enemy')
}

/**
 * scoreCurveOption — l'option ECharts, extraite du composant pour rester lisible.
 *
 * UN SEUL AXE DE VALEURS (règle dataviz) : les deux camps se comparent sur la même échelle,
 * sans quoi « qui mène » deviendrait une illusion d'optique. L'axe des temps est en valeurs
 * (millisecondes) et non en catégories : les paliers ne tombent pas sur une grille régulière,
 * et les espacer également mentirait sur le rythme du match.
 */
function scoreCurveOption(
  curve: ScoreCurve,
  tc: EChartsThemeColors,
  t: MatchViewText,
): EChartsCoreOption {
  const couleurs = new Map(curve.series.map((s) => [s.teamId, colorOf(s, tc)]))
  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ bottom: 44, left: 40, right: 16, top: 12 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line', lineStyle: { color: tc.splitLine } },
      formatter: tooltipFormatter(curve, t),
    },
    legend: { ...getLegendBase(tc), data: curve.series.map((s) => s.label) },
    xAxis: {
      ...getAxisBase(tc),
      type: 'value',
      min: 0,
      max: curve.durationMs,
      axisLabel: { color: tc.axisLabel, fontSize: 9, formatter: (v: number) => formatClock(v) },
      splitLine: { show: false },
    },
    yAxis: {
      ...getAxisBase(tc),
      type: 'value',
      min: 0,
      minInterval: 1,
      axisLabel: { color: tc.axisLabel, fontSize: 9 },
      splitLine: { show: true, lineStyle: { color: tc.splitLine, opacity: 0.35 } },
    },
    series: curve.series.map((s, i) => {
      const color = couleurs.get(s.teamId) ?? tc.axisLabel
      return {
        type: 'line',
        name: s.label,
        // ESCALIER : le film ne transmet que les changements, la valeur ATTEND entre deux
        // paliers. `end` fait sauter la courbe AU point, jamais avant (cf. _scoreCurve).
        step: 'end',
        data: s.points,
        lineStyle: { color, width: 2 },
        ...hoverRevealSymbol(color),
        // Les RETOURNEMENTS ne se posent qu'une fois, sur la première série : répétés sur
        // chaque courbe, ils doubleraient chaque trait.
        markLine:
          i === 0 && curve.leads.length > 0
            ? {
                silent: true,
                symbol: ['none', 'none'],
                label: { show: false },
                data: curve.leads.map((lead) => ({
                  xAxis: lead.ms,
                  lineStyle: {
                    color: couleurs.get(lead.teamId) ?? tc.axisLabel,
                    width: 1,
                    type: 'dashed',
                    opacity: 0.55,
                  },
                })),
              }
            : undefined,
        z: 3,
      }
    }),
  }
}

/**
 * tooltipFormatter écrit l'instant en mm:ss, chaque camp avec sa valeur, et signale le
 * RETOURNEMENT quand le survol tombe dessus.
 *
 * Les noms d'équipe viennent de la donnée (`team_name` du backend) : ils sont échappés,
 * comme partout où un tooltip ECharts interpole autre chose qu'une constante.
 */
function tooltipFormatter(curve: ScoreCurve, t: MatchViewText) {
  return (params: unknown): string => {
    const arr = (Array.isArray(params) ? params : [params]) as Array<{
      value?: [number, number]
      seriesName?: string
    }>
    const ms = arr[0]?.value?.[0] ?? 0
    const lignes = arr.map(
      (p) => `${escapeHtml(p.seriesName ?? '')} : <b>${p.value?.[1] ?? 0}</b>`,
    )
    const lead = curve.leads.find((l) => Math.abs(l.ms - ms) < 1)
    if (lead) {
      const meneur = curve.series.find((s) => s.teamId === lead.teamId)?.label ?? ''
      lignes.push(`<i>${escapeHtml(t.scoreCurveLead)} — ${escapeHtml(meneur)}</i>`)
    }
    return [`<b>${formatClock(ms)}</b>`, ...lignes].join('<br/>')
  }
}
