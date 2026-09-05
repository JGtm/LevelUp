/**
 * MatchScoreEventsChart — LES POINTS MARQUÉS DANS LE TEMPS, décodés du film du match.
 *
 * CE QU'IL FAIT QUE LA COURBE NE FAISAIT PAS. `MatchScoreCurveChart` trace le score en
 * escalier, et c'est la bonne lecture d'un mode qui marque en continu (Oddball au tic,
 * Bastion à la garde). En capture de drapeau, en roi de la colline et en assaut, un match
 * entier tient en TROIS à CINQ paliers : la courbe y est un escalier vide, deux lignes
 * plates qui sautent une fois par tiers de match. Ce sont les INSTANTS de marque qui portent
 * l'information, et c'est ce que ce graphe pose — une barre verticale par point marqué.
 *
 * QUI DÉCIDE LEQUEL DES DEUX S'AFFICHE : la DONNÉE, jamais ce fichier. L'en-tête du match
 * porte `score_timeline_kind` (`hidden` / `events` / `curve`), résolu côté serveur depuis
 * `regulation.toml [score_timeline]` par jeton de mode. Un mode non déclaré retombe sur la
 * courbe — le comportement d'avant le 2026-09-03.
 *
 * AUCUN APPEL DE PLUS QUE LA COURBE : même artefact, même clé de cache (`useMatchReplay`,
 * gaté par `header.replay_available`), même calque (`scoreTimelineOf`, donc la même garde
 * d'horloge — un calque daté depuis une origine non résolue placerait chaque point au
 * mauvais instant). Mêmes portes, aussi : pas d'artefact, pas de calque, mode sans compteur
 * (`coverage.score.modeSupported === false`), aucune marque -> RIEN. Pas de cadre vide.
 *
 * DEUX SÉRIES DE BARRES DÉCALÉES, ET NON DEUX VOIES SUR L'AXE VERTICAL. Le choix se justifie
 * par ce que chaque axe porte : l'axe vertical dit COMBIEN a été marqué d'un coup (un point
 * en drapeau, mais un mode peut en donner plusieurs), et le dépenser en étiquettes d'équipe
 * l'aurait rendu muet. L'identité du camp, elle, se lit à la COULEUR et à la légende — la
 * MÊME encre que la courbe qu'il remplace (`teamSeriesColor` : les tokens `team-ally` /
 * `team-enemy`, donc la palette d'accessibilité réglée par l'utilisateur), jamais une teinte
 * inventée. ECharts décale automatiquement deux séries de barres au même instant.
 *
 * La projection (paliers -> marques datées) vit dans `_scoreEvents.ts`, pur et testé.
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
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import { useMatchReplay } from '@/features/match-replay/queries'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { matchClock } from '@/lib/replay/matchClock'
import { allyOfTeamId, scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { formatClock, teamIdsOf } from './_scoreCurve'
import { buildScoreEvents, type ScoreEvents, type ScoreEventsTeam } from './_scoreEvents'
import type { MatchViewText } from './i18n'
import { teamSeriesColor } from './teamSeriesColor'
import { resolveXuidMeta } from './xuidMeta'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — la présence de l'artefact, le même gate que le lien rejeu. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /**
   * `header.t0_ms` — le countdown d'avant-match, en ms. C'est l'ancre qui met ces barres sur
   * l'horloge du GAMEPLAY, celle de « Frags cumulés » juste au-dessus dans le même onglet
   * (cf. `lib/replay/matchClock`).
   */
  t0Ms?: number
  t: MatchViewText
}

export function MatchScoreEventsChart({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  meXUID,
  t0Ms,
  t,
}: Props) {
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const xuidMeta = useMemo(() => resolveXuidMeta(board, meXUID), [board, meXUID])
  const timeline = useMemo(() => (data ? scoreTimelineOf(data) : undefined), [data])
  // L'HORLOGE DU MATCH, ÉTABLIE UNE FOIS : même axe que la courbe et que « Frags cumulés ».
  const clock = useMemo(() => matchClock(data, { t0_ms: t0Ms }), [data, t0Ms])
  const events = useMemo(
    () =>
      buildScoreEvents({
        timeline,
        clock,
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
    [timeline, clock, board, xuidMeta, t],
  )

  const buildOption = useCallback(
    (s: ChartSeries<ScoreEventsTeam>[]): EChartsCoreOption => {
      if (s.length === 0 || !events) return { backgroundColor: CHART_BG }
      // Les tokens sont résolus ICI, dans le builder : leur valeur calculée change avec la
      // palette d'accessibilité, et c'est ce rebuild qui la rafraîchit (cf. ChartCard,
      // deps paletteVersion).
      return scoreEventsOption(events, getEChartsThemeColors(), t)
    },
    [events, t],
  )

  // Pas d'artefact, pas de calque, aucune marque, ou mode sans compteur : RIEN.
  if (!events || data?.coverage?.score?.modeSupported === false) return null

  return (
    <ChartCard
      title={t.scoreEventsTitle}
      series={[{ key: 'match_view.score_events', datapoints: events.teams }]}
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

/**
 * scoreEventsOption — l'option ECharts, extraite du composant pour rester lisible.
 *
 * L'AXE DES TEMPS EST EN VALEURS (millisecondes DEPUIS LE COUP D'ENVOI, cf. `_scoreEvents`)
 * et non en catégories : les marques ne tombent pas sur une grille régulière, et les espacer
 * également mentirait sur le rythme du match — le même choix que la courbe. Il court de 0 à
 * la fin du film, pas du premier au dernier point : une capture à 1:10 et une à 9:50 doivent
 * se lire aux deux bouts.
 *
 * `barWidth` EST EN PIXELS, ET C'EST OBLIGATOIRE ici : sur un axe de valeurs, la largeur
 * automatique d'une barre se déduit de l'intervalle entre catégories — qui n'existe pas.
 * Sans elle, trois marques donneraient trois pavés larges d'un tiers de graphe.
 */
function scoreEventsOption(
  events: ScoreEvents,
  tc: EChartsThemeColors,
  t: MatchViewText,
): EChartsCoreOption {
  const couleurs = new Map(
    events.teams.map((team) => [team.teamId, teamSeriesColor(team.ally, tc)] as const),
  )
  return {
    backgroundColor: CHART_BG,
    // `bottom` laisse la place à la légende posée sous le graphe.
    grid: getGridBase({ bottom: 52, left: 40, right: 16, top: 12 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: tooltipFormatter(t),
    },
    // LÉGENDE EN BAS, CENTRÉE (demande utilisateur du 2026-09-03) : c'est elle qui porte
    // l'identité des camps, puisque l'axe vertical porte les points marqués.
    legend: { ...getLegendBase(tc), left: 'center', data: events.teams.map((s) => s.label) },
    xAxis: {
      ...getAxisBase(tc),
      type: 'value',
      min: 0,
      max: events.endMs,
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
    series: events.teams.map((team) => {
      const color = couleurs.get(team.teamId) ?? tc.axisLabel
      return {
        type: 'bar',
        name: team.label,
        barWidth: 6,
        barMinHeight: 2,
        itemStyle: { color },
        // `[ms, points marqués, score cumulé]` : la troisième valeur ne se dessine pas,
        // elle sert l'infobulle — le cumul vient du FILM, jamais d'une somme refaite ici.
        data: team.events.map((e) => [e.ms, e.points, e.total]),
        z: 3,
      }
    }),
  }
}

/**
 * tooltipFormatter écrit l'instant en mm:ss, le camp, les points pris et le score cumulé
 * juste après.
 *
 * Les noms d'équipe viennent de la donnée (`team_name` du backend) : ils sont échappés,
 * comme partout où un tooltip ECharts interpole autre chose qu'une constante.
 */
function tooltipFormatter(t: MatchViewText) {
  return (params: unknown): string => {
    const p = (Array.isArray(params) ? params[0] : params) as {
      value?: [number, number, number]
      seriesName?: string
    }
    const [ms, points, total] = p?.value ?? [0, 0, 0]
    return [
      `<b>${formatClock(ms)}</b>`,
      `${escapeHtml(p?.seriesName ?? '')} : <b>${escapeHtml(t.scoreEventsScoredFmt(points))}</b>`,
      escapeHtml(t.scoreEventsTotalFmt(total)),
    ].join('<br/>')
  }
}
