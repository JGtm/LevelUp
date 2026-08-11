/**
 * Tests — MatchTugOfWarChart (carte Dominance, W2 revue 2026-07-17).
 *
 * Régression W2 : le passage du tooltip global à `trigger:'axis'` avait tué les
 * tooltips item des séries kill-feed / vagues (ECharts n'honore pas un `trigger`
 * par-série). Le fix repasse le trigger global à `'item'` et sert le résumé de
 * tranche via le tooltip item des barres.
 *
 * On ne rend PAS ECharts en jsdom (canvas non supporté) : echarts-for-react est
 * mocké pour capturer l'option, puis on invoque directement les formatters —
 * le tip per-kill au survol d'un point kill ET le résumé de bin au survol d'une
 * barre (les DEUX comportements exigés).
 */
import { describe, it, expect, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'

import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import { MATCH_VIEW_TEXT } from './i18n'
import type {
  MatchHighlightEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'

// resolveToken → hex réel pour que hexToRgba produise du rgba() (sinon fallback).
vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => (token === 'team-ally' ? '#22c55e' : '#ef4444'),
  tokenCssVar: (token: string) => `var(--${token})`,
}))

// Capture l'option ECharts sans rendre le canvas (règle repo).
const captured = vi.hoisted(() => ({ option: null as unknown }))
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => {
    captured.option = option
    return <div data-testid="echarts-stub" />
  },
}))

type Fn = (p: unknown) => string
interface SeriesLike {
  type?: string
  name?: string
  data?: Array<{ _tip?: string }>
  tooltip?: { trigger?: string; formatter?: Fn }
}
interface OptionLike {
  tooltip?: { trigger?: string }
  series?: SeriesLike[]
}

function sb(xuid: string, gamertag: string, teamSide: string, isMe = false): MatchScoreboardRow {
  return { xuid, gamertag, team_side: teamSide, is_me: isMe, outcome_label: 'Win' } as MatchScoreboardRow
}

function ev(actorXuid: string, tMs: number): MatchHighlightEvent {
  return {
    event_type: 'kill',
    actor_xuid: actorXuid,
    event_time_ms: tMs,
    target_xuid: null,
    weapon_id: null,
  }
}

const BINS: MatchTugOfWarBin[] = [
  { bin_start: 0, bin_end: 60, team_kills: 0, enemy_kills: 0, net_kills: 0 },
  { bin_start: 60, bin_end: 120, team_kills: 0, enemy_kills: 0, net_kills: 0 },
]

const SCOREBOARD: MatchScoreboardRow[] = [
  sb('me', 'MePlayer', '0', true),
  sb('ally2', 'AllyTwo', '0'),
  sb('foe', 'FoePlayer', '1'),
]

// 3 kills alliés dans les 8 s (→ 1 vague ×3, bin 0, delta +3) + 1 kill ennemi (bin 1).
const EVENTS: MatchHighlightEvent[] = [
  ev('me', 1000),
  ev('ally2', 3000),
  ev('me', 5000),
  ev('foe', 65000),
]

async function renderAndCapture(): Promise<OptionLike> {
  captured.option = null
  render(
    <MatchTugOfWarChart
      bins={BINS}
      events={EVENTS}
      scoreboard={SCOREBOARD}
      meXUID="me"
      t={MATCH_VIEW_TEXT.fr}
    />,
  )
  await waitFor(() => expect(captured.option).not.toBeNull())
  return captured.option as OptionLike
}

describe('MatchTugOfWarChart — tooltips (W2)', () => {
  it("le trigger global est 'item' (pas 'axis' — sinon les tips par-série sont morts)", async () => {
    const opt = await renderAndCapture()
    expect(opt.tooltip?.trigger).toBe('item')
  })

  it('résumé de bin au survol d’une barre (delta signé + kills X/Y)', async () => {
    const opt = await renderAndCapture()
    const bars = (opt.series ?? []).filter((s) => s.type === 'bar')
    expect(bars.length).toBe(2)
    const tip = bars[0].tooltip?.formatter?.({ seriesType: 'bar', dataIndex: 0 }) ?? ''
    // combatMomentumDelta FR = 'Écart' ; bin 0 : delta +3 en faveur de « Mon équipe ».
    expect(tip).toContain('Écart')
    expect(tip).toContain('+3')
    expect(tip).toContain('Mon équipe')
    // Ce n'est PAS un tip per-kill.
    expect(tip).not.toContain('0:01')
  })

  it('les kills ne sont plus des symboles ECharts (ils sont rendus en DOM)', async () => {
    const opt = await renderAndCapture()
    // Garde-rail de non-retour : un symbole image ECharts ne se teint pas, donc l'icône
    // d'arme du kill feed ne peut PAS revivre en `scatter`. Les lanes et les vagues, si.
    const scatter = (opt.series ?? []).filter((s) => s.type === 'scatter')
    expect(scatter).toHaveLength(0)
    const lanes = (opt.series ?? []).filter((s) => s.name === 'Lane alliée' || s.name === 'Lane ennemie')
    expect(lanes).toHaveLength(2)
  })

  it('tip de vague au survol d’un segment (détail ×N)', async () => {
    const opt = await renderAndCapture()
    const wave = (opt.series ?? []).find(
      (s) => typeof s.name === 'string' && s.name.startsWith('wave-ally-'),
    )
    expect(wave).toBeTruthy()
    expect(wave?.tooltip?.trigger).toBeUndefined()
    const datum = wave?.data?.[1]
    const tip = wave?.tooltip?.formatter?.({ data: datum }) ?? ''
    expect(tip).toContain('×3')
    expect(tip).toContain('Vague')
  })
})
