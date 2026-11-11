/**
 * MatchScoreDeltaChart — L'ÉCART DE SCORE, décodé du film du match.
 *
 * CE QU'IL AJOUTE À « SCORE DANS LE TEMPS » : qui mène, et de combien, lus en une seule
 * grandeur signée autour de zéro — l'équipe du joueur au-dessus, l'équipe d'en face en dessous.
 * Les deux graphes sont des angles de lecture complémentaires du MÊME calque de score ; ils se
 * lisent l'un sous l'autre sur le même axe des temps.
 *
 * MÊME SOURCE, MÊME CLÉ DE CACHE, MÊME GATE QUE LA COURBE (`useMatchReplay`, gaté par
 * `header.replay_available`) : aucun appel de plus. La carte reste TOUJOURS À L'ÉCRAN : sans film,
 * le bloc vide canonique dit que le graphique demande le rejeu ; avec un film mais sans écart
 * traçable (score non daté, camp du joueur non établi), l'état vide en dit la cause — jamais une
 * courbe devinée.
 *
 * ELLE NE LIT PAS `score_timeline_kind`. Ce champ efface la COURBE en Assassin (le mode marque
 * au frag, « Frags cumulés » le dit déjà) ; l'écart, lui, vit du score du rejeu dans tous les
 * modes (plan 8.1.2, D-9).
 *
 * La projection vit dans `_scoreDelta.ts`, l'option ECharts dans `_divergentStepOption.ts` : pur
 * et testé.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { tipNumber } from '@/components/charts/tipNumber'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { matchClock } from '@/lib/replay/matchClock'
import { useMatchReplay } from '@/lib/replay/queries'
import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { formatClock, teamIdsOf } from './_scoreCurve'
import { buildDivergentStepOption, type DivergentStepPoint } from './_divergentStepOption'
import { buildScoreDelta, scoreDeltaAt, type ScoreDelta } from './_scoreDelta'
import type { MatchViewText } from './i18n'
import { allyOfTeamId, resolveXuidMeta } from './xuidMeta'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — la présence de l'artefact, le même gate que la courbe. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /** `header.t0_ms` — l'ancre de l'horloge du gameplay (cf. `lib/replay/matchClock`). */
  t0Ms?: number
  t: MatchViewText
}

export function MatchScoreDeltaChart({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  meXUID,
  t0Ms,
  t,
}: Props) {
  const { data, isLoading } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const xuidMeta = useMemo(() => resolveXuidMeta(board, meXUID), [board, meXUID])
  // La garde d'horloge du rejeu s'applique ICI AUSSI : un calque daté depuis une origine non
  // résolue placerait chaque palier au mauvais instant (cf. lib/replay/scoreTimeline).
  const timeline = useMemo(() => (data ? scoreTimelineOf(data) : undefined), [data])
  const clock = useMemo(() => matchClock(data, { t0_ms: t0Ms }), [data, t0Ms])
  const modeSupported = data?.coverage?.score?.modeSupported !== false
  const delta = useMemo(
    () =>
      modeSupported
        ? buildScoreDelta({
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
          })
        : null,
    [modeSupported, timeline, clock, board, xuidMeta, t],
  )

  const buildOption = useCallback(
    (s: ChartSeries<DivergentStepPoint>[]): EChartsCoreOption => {
      if (s.length === 0 || !delta) return { backgroundColor: CHART_BG }
      return buildDivergentStepOption(
        {
          points: s[0].datapoints,
          endMs: delta.endMs,
          // Les noms de la légende sont ceux de « Score dans le temps » ; les aires, leurs couleurs.
          positiveName: delta.allyLabel,
          negativeName: delta.opponentLabel ?? t.divergentBestOpponent,
          negativeNeutral: delta.multiTeam,
          tooltipHtml: (ms) => tooltipHtml(delta, ms, t),
        },
        getEChartsThemeColors(),
      )
    },
    [delta, t],
  )

  // La cause de l'absence : pas de rejeu, le film ne date pas le score, ou le camp du joueur est inconnu.
  const unavailable = !replayAvailable
    ? t.divergentNeedsReplay
    : timeline && clock && modeSupported
      ? t.scoreDeltaUnavailableCamps
      : t.scoreDeltaUnavailableFilm

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {t.scoreDeltaTitle}
          <InfoTooltip
            content={
              <p>
                {t.scoreDeltaInfo}
                {delta?.multiTeam ? ` ${t.divergentMultiTeamNote}` : ''}
              </p>
            }
          />
        </span>
      }
      series={
        delta
          ? [
              {
                key: 'match_view.score_delta',
                datapoints: delta.points.map((p) => ({ ms: p.ms, value: p.delta })),
              },
            ]
          : []
      }
      loading={isLoading}
      height={260}
      buildOption={buildOption}
      emptyMessage={unavailable}
    />
  )
}

/**
 * tooltipHtml écrit l'instant en M:SS, le score des deux camps et l'écart signé, lus sur le
 * point TENU à cet instant. Les noms d'équipe viennent de la donnée : échappés, comme partout
 * où une infobulle ECharts interpole autre chose qu'une constante.
 */
function tooltipHtml(delta: ScoreDelta, ms: number, t: MatchViewText): string {
  const p = scoreDeltaAt(delta, ms)
  return [
    `<b>${formatClock(Math.max(0, ms))}</b>`,
    `${escapeHtml(delta.allyLabel)} : <b>${tipNumber(p.ally)}</b>`,
    `${escapeHtml(delta.opponentLabel ?? t.divergentBestOpponent)} : <b>${tipNumber(p.opponent)}</b>`,
    `${escapeHtml(t.divergentGapLabel)} : <b>${tipNumber(p.delta, 0, true)}</b>`,
  ].join('<br/>')
}
