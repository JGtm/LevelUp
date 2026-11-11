/**
 * MatchViewTabChronology.divergent.test.tsx — L'ORDRE ET LE CÂBLAGE DES DEUX GRAPHES DIVERGENTS.
 *
 * Ce que l'onglet doit montrer, de haut en bas : Frags cumulés, la lecture du score (points
 * marqués OU score dans le temps), Écart de score, Supériorité numérique, Dominance, Cadence.
 * Les graphes sont remplacés par des stubs repérables : c'est leur PLACE et leurs PROPS qu'on
 * éprouve, pas leur dessin (chacun a son propre test).
 */
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import { SCORE_TIMELINE_EVENTS, SCORE_TIMELINE_HIDDEN } from './scoreTimelineKind'

const props = vi.hoisted(() => ({
  delta: [] as Record<string, unknown>[],
  numbers: [] as Record<string, unknown>[],
}))

const stub = (id: string) => () => <div data-testid={id} />
vi.mock('./MatchImpactBadgesBar', () => ({ MatchImpactBadgesBar: stub('badges') }))
vi.mock('./MatchKDCumulChart', () => ({ MatchKDCumulChart: stub('kd-cumul') }))
vi.mock('./MatchScoreCurveChart', () => ({ MatchScoreCurveChart: stub('score-curve') }))
vi.mock('./MatchScoreEventsChart', () => ({ MatchScoreEventsChart: stub('score-events') }))
vi.mock('./MatchTugOfWarChart', () => ({ MatchTugOfWarChart: stub('dominance') }))
vi.mock('./MatchCadenceChart', () => ({ MatchCadenceChart: stub('cadence') }))
vi.mock('@/features/engagement/EngagementMatchSection', () => ({
  EngagementMatchSection: stub('engagement'),
}))
vi.mock('./MatchScoreDeltaChart', () => ({
  MatchScoreDeltaChart: (p: Record<string, unknown>) => {
    props.delta.push(p)
    return <div data-testid="score-delta" />
  },
}))
vi.mock('./MatchNumbersAdvantageChart', () => ({
  MatchNumbersAdvantageChart: (p: Record<string, unknown>) => {
    props.numbers.push(p)
    return <div data-testid="numbers-advantage" />
  },
}))

const { MatchViewTabChronology } = await import('./MatchViewTabChronology')

const t = MATCH_VIEW_TEXT.fr
const SCOREBOARD = [
  { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
] as unknown as MatchScoreboardRow[]

function afficher(scoreTimelineKind?: string, replayAvailable = true) {
  props.delta.length = 0
  props.numbers.length = 0
  return renderWithProviders(
    <MatchViewTabChronology
      playerSlug="joueur"
      matchId="m1"
      replayAvailable={replayAvailable}
      impactBadges={[]}
      highlightEvents={[]}
      scoreboard={SCOREBOARD}
      meXUID="me"
      objectiveEvents={undefined}
      tugOfWar={[]}
      cadence={null}
      scoreTimelineKind={scoreTimelineKind}
      t0Ms={30_000}
      t={t}
    />,
  )
}

/** Les identifiants de stub, dans l'ordre du document. */
function ordre(view: ReturnType<typeof afficher>): string[] {
  return [...view.container.querySelectorAll('[data-testid]')].map(
    (n) => n.getAttribute('data-testid') ?? '',
  )
}

describe('onglet Chronologie — l’ordre des graphes', () => {
  it('score dans le temps, puis Écart de score, puis Supériorité numérique, puis Dominance et Cadence', () => {
    const ids = ordre(afficher())
    const suite = ['kd-cumul', 'score-curve', 'score-delta', 'numbers-advantage', 'dominance', 'cadence']
    expect(ids.filter((i) => suite.includes(i))).toEqual(suite)
  })

  it('en mode à peu de points, les points marqués précèdent de même les deux graphes', () => {
    const ids = ordre(afficher(SCORE_TIMELINE_EVENTS))
    const suite = ['kd-cumul', 'score-events', 'score-delta', 'numbers-advantage', 'dominance', 'cadence']
    expect(ids.filter((i) => suite.includes(i))).toEqual(suite)
  })

  it('en Assassin (`hidden`) la courbe s’efface mais l’écart et la supériorité restent (D-9)', () => {
    const ids = ordre(afficher(SCORE_TIMELINE_HIDDEN))
    expect(ids).toContain('score-delta')
    expect(ids).toContain('numbers-advantage')
  })
})

describe('onglet Chronologie — le câblage des deux graphes', () => {
  it('chacun reçoit le film du match, le joueur de la page et le coup d’envoi', () => {
    afficher()
    for (const p of [props.delta.at(-1), props.numbers.at(-1)]) {
      expect(p).toMatchObject({
        playerSlug: 'joueur',
        matchId: 'm1',
        replayAvailable: true,
        meXUID: 'me',
        t0Ms: 30_000,
      })
      expect(p?.scoreboard).toBe(SCOREBOARD)
    }
  })

  it('l’écart ne reçoit PAS le type de lecture du score : il ne dépend pas de `hidden`', () => {
    afficher(SCORE_TIMELINE_HIDDEN)
    expect(props.delta.at(-1)).not.toHaveProperty('scoreTimelineKind')
  })
})
