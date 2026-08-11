import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

import { MatchKDCumulChart } from './MatchKDCumulChart'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import { MATCH_VIEW_TEXT } from './i18n'
import type {
  MatchHighlightEvent,
  MatchObjectiveEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  // tokenCssVar : requis depuis que le kill feed est rendu en DOM (MatchKillFeed) et
  // teinte les icônes d'arme par équipe. Sans lui, le mock partiel casse le rendu.
  tokenCssVar: (token: string) => `var(--${token})`,
}))

// Un chart ECharts monté en jsdom crashe le canvas — on stubbe le wrapper.
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option).slice(0, 80)}</div>
  ),
}))

const t = MATCH_VIEW_TEXT.fr

function sbRow(partial: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return {
    xuid: 'x', gamertag: 'GT', team_side: null, is_me: false, rank: null, score: null,
    kills: null, deaths: null, assists: null, shots_fired: null, shots_hit: null,
    accuracy: null, damage_dealt: null, damage_taken: null, average_life: null,
    headshot_kills: null, max_killing_spree: null, perfect_kills: null,
    power_weapon_kills: null, melee_kills: null, outcome_label: '', ...partial,
  } as MatchScoreboardRow
}

const SCOREBOARD: MatchScoreboardRow[] = [
  sbRow({ xuid: 'me', gamertag: 'Me', team_side: '0', is_me: true }),
  sbRow({ xuid: 'ally1', gamertag: 'AllyOne', team_side: '0' }),
  sbRow({ xuid: 'enemy1', gamertag: 'EnemyOne', team_side: '1' }),
]

const KILLS: MatchHighlightEvent[] = [
  { event_time_ms: 5000, event_type: 'kill', actor_xuid: 'me', target_xuid: 'enemy1', weapon_id: null },
  { event_time_ms: 12000, event_type: 'kill', actor_xuid: 'enemy1', target_xuid: 'ally1', weapon_id: null },
  { event_time_ms: 20000, event_type: 'kill', actor_xuid: 'ally1', target_xuid: 'enemy1', weapon_id: null },
]

const CAPTURES: MatchObjectiveEvent[] = [
  { matchId: 'm1', seq: 0, timeMs: 8000, objectiveType: 'flag', eventType: 'capture', teamId: 0, value: 1, source: 'film', confidence: 'high', players: [{ xuid: 'me', role: 'scorer' }] },
  { matchId: 'm1', seq: 1, timeMs: 18000, objectiveType: 'flag', eventType: 'capture', teamId: 1, value: 1, source: 'film', confidence: 'high', players: [{ xuid: 'enemy1', role: 'scorer' }] },
]

const BINS: MatchTugOfWarBin[] = [
  { bin_start: 0, bin_end: 10, team_kills: 1, enemy_kills: 0, net_kills: 1 },
  { bin_start: 10, bin_end: 20, team_kills: 1, enemy_kills: 1, net_kills: 0 },
  { bin_start: 20, bin_end: 30, team_kills: 1, enemy_kills: 0, net_kills: 1 },
]

describe('CTF capture overlay — charts combat', () => {
  it('MatchKDCumulChart rend sans crash avec objectiveEvents fournis', async () => {
    render(
      <MatchKDCumulChart
        events={KILLS}
        badges={[]}
        scoreboard={SCOREBOARD}
        meXUID="me"
        objectiveEvents={CAPTURES}
        t={t}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('echarts-stub')).toBeInTheDocument()
    })
  })

  it('MatchKDCumulChart rend sans crash quand objectiveEvents est null', async () => {
    render(
      <MatchKDCumulChart
        events={KILLS}
        badges={[]}
        scoreboard={SCOREBOARD}
        meXUID="me"
        objectiveEvents={null}
        t={t}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('echarts-stub')).toBeInTheDocument()
    })
  })

  it('MatchTugOfWarChart rend sans crash avec objectiveEvents fournis', async () => {
    render(
      <MatchTugOfWarChart
        bins={BINS}
        events={KILLS}
        scoreboard={SCOREBOARD}
        meXUID="me"
        objectiveEvents={CAPTURES}
        t={t}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('echarts-stub')).toBeInTheDocument()
    })
  })

  it('MatchTugOfWarChart rend sans crash quand objectiveEvents est null', async () => {
    render(
      <MatchTugOfWarChart
        bins={BINS}
        events={KILLS}
        scoreboard={SCOREBOARD}
        meXUID="me"
        objectiveEvents={null}
        t={t}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('echarts-stub')).toBeInTheDocument()
    })
  })
})
