/**
 * MatchNumbersAdvantageChart — LA SUPÉRIORITÉ NUMÉRIQUE, lue sur les vies du film du match.
 *
 * CE QU'IL DIT : à chaque seconde du match, combien de joueurs en vie de plus (ou de moins)
 * l'équipe du joueur compte face à l'équipe d'en face, BOTS COMPRIS. Il se lit sous « Écart de
 * score » : le rapport de force qui précède, ou explique, un changement de score.
 *
 * LA LOGIQUE N'EST PAS ICI. `lib/replay/aliveBalanceLogic` compte les joueurs en vie (la vie est
 * celle de `isAliveAt`, l'équipe est celle du film, `ReplayPlayer.team`) ; ce composant ne fait
 * que la mémoïser — UNE fois par document et par camp, jamais à chaque rendu — et la tracer
 * avec l'option divergente commune (`_divergentStepOption`).
 *
 * MÊME GATE QUE LES AUTRES GRAPHES DU FILM (`useMatchReplay`, `header.replay_available`) :
 * sans film, la carte reste à l'écran avec le bloc vide canonique (le graphique demande le rejeu).
 * Avec un film mais sans camp établi pour le joueur, l'état vide dit pourquoi — jamais un
 * effectif deviné.
 */
import type { ChartOption } from '@/components/charts/chartOption'
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { tipNumber } from '@/components/charts/tipNumber'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { aliveBalanceAt, buildAliveBalance, type AliveBalance } from '@/lib/replay/aliveBalanceLogic'
import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import { matchClock } from '@/lib/replay/matchClock'
import { useMatchReplay } from '@/lib/replay/queries'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { formatClock } from './_scoreCurve'
import { teamLegend } from './teamLegend'
import { buildDivergentStepOption, type DivergentStepPoint } from './_divergentStepOption'
import type { MatchViewText } from './i18n'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — la présence de l'artefact, le même gate que les autres graphes du film. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /** `header.t0_ms` — l'ancre de l'horloge du gameplay (cf. `lib/replay/matchClock`). */
  t0Ms?: number
  t: MatchViewText
}

export function MatchNumbersAdvantageChart({
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
  const players = useMemo(() => (data ? buildPlayers(data, board) : []), [data, board])
  // L'équipe du joueur est celle du FILM (ADR 0034 D-9) : la feuille ne décide pas du camp.
  const allyTeam = useMemo(() => buildFilmAllegiance(players, meXUID).allyTeam, [players, meXUID])
  const clock = useMemo(() => matchClock(data, { t0_ms: t0Ms }), [data, t0Ms])
  const balance = useMemo(
    () => buildAliveBalance({ players, allyTeam, clock }),
    [players, allyTeam, clock],
  )
  const legend = useMemo(() => teamLegend(board, meXUID, t), [board, meXUID, t])

  const buildOption = useCallback(
    (s: ChartSeries<DivergentStepPoint>[]): ChartOption => {
      if (s.length === 0 || !balance) return { backgroundColor: CHART_BG }
      return buildDivergentStepOption(
        {
          points: s[0].datapoints,
          endMs: balance.endMs,
          // Les noms de la légende sont ceux de la feuille (comme « Score dans le temps ») ; sans
          // camp situé, le texte générique. Les aires prennent les couleurs d'équipe.
          positiveName: legend.ally ?? t.numbersAdvantageLegendAlly,
          negativeName: legend.opponent ?? t.numbersAdvantageLegendEnemy,
          negativeNeutral: balance.multiTeam,
          tooltipHtml: (ms) => tooltipHtml(balance, ms, t),
        },
        getEChartsThemeColors(),
      )
    },
    [balance, t, legend],
  )

  // La cause de l'absence : pas de rejeu, le film ne date pas les vies, ou le camp du joueur est inconnu.
  const unavailable = !replayAvailable
    ? t.divergentNeedsReplay
    : clock && players.length > 0
      ? t.numbersAdvantageUnavailableCamps
      : t.numbersAdvantageUnavailableFilm

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {t.numbersAdvantageTitle}
          <InfoTooltip
            content={
              <p>
                {t.numbersAdvantageInfo}
                {balance?.multiTeam ? ` ${t.divergentMultiTeamNote}` : ''}
              </p>
            }
          />
        </span>
      }
      series={
        balance
          ? [
              {
                key: 'match_view.numbers_advantage',
                datapoints: balance.points.map((p) => ({ ms: p.ms, value: p.ally - p.opponent })),
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
 * tooltipHtml écrit l'instant du relevé en M:SS (la seconde entière), les joueurs en vie de chaque côté et l'écart signé, lus
 * sur le point d'échantillonnage tenu à cet instant (un point par seconde). Le nom du camp d'en
 * face est celui de la donnée quand il n'y en a qu'un seul ; sinon la meilleure équipe adverse.
 */
function tooltipHtml(balance: AliveBalance, ms: number, t: MatchViewText): string {
  const p = aliveBalanceAt(balance, ms)
  const against = balance.multiTeam ? t.divergentBestOpponent : t.combatEnemyLabel
  return [
    `<b>${formatClock(p.ms)}</b>`,
    `${escapeHtml(t.combatTeamLabel)} : <b>${escapeHtml(t.numbersAdvantageAliveFmt(p.ally))}</b>`,
    `${escapeHtml(against)} : <b>${escapeHtml(t.numbersAdvantageAliveFmt(p.opponent))}</b>`,
    `${escapeHtml(t.divergentGapLabel)} : <b>${tipNumber(p.ally - p.opponent, 0, true)}</b>`,
  ].join('<br/>')
}
