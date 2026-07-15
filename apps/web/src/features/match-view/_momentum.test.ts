// @vitest-environment node
/**
 * Tests de `computeMomentumBins` (logique pure de l'histogramme momentum,
 * carte « Dominance » match_view.10). Couvre les cas a–g du plan
 * PLAN_MATCHVIEW_MOMENTUM Phase 2.
 */
import { describe, it, expect } from 'vitest'
import { computeMomentumBins } from './_momentum'
import type { MatchHighlightEvent, MatchTugOfWarBin } from '@/lib/api/types'

/** Bin de `[start, end)` secondes ; les counts backend sont ignorés (recompute front). */
function bin(start: number, end: number): MatchTugOfWarBin {
  return { bin_start: start, bin_end: end, team_kills: 0, enemy_kills: 0, net_kills: 0 }
}

function killEvent(actorXuid: string | null, timeMs: number | null): MatchHighlightEvent {
  return {
    event_type: 'kill',
    actor_xuid: actorXuid,
    event_time_ms: timeMs,
    target_xuid: null,
    weapon_id: null,
  }
}

// ally1/ally2 alliés, enemy1 ennemi.
const META = new Map<string, { ally: boolean }>([
  ['ally1', { ally: true }],
  ['ally2', { ally: true }],
  ['enemy1', { ally: false }],
])

describe('computeMomentumBins', () => {
  it('(a) nominal deux équipes — deltas et cumuls corrects', () => {
    const bins = [bin(0, 30), bin(30, 60), bin(60, 90)]
    const events = [
      killEvent('ally1', 5_000), // bin0
      killEvent('ally2', 10_000), // bin0
      killEvent('enemy1', 15_000), // bin0
      killEvent('enemy1', 40_000), // bin1
      killEvent('ally1', 70_000), // bin2
    ]
    const { momentum, kills } = computeMomentumBins(bins, events, META)

    expect(kills).toHaveLength(5)
    expect(momentum.map((b) => b.delta)).toEqual([1, -1, 1])
    expect(momentum.map((b) => b.teamKills)).toEqual([2, 0, 1])
    expect(momentum.map((b) => b.enemyKills)).toEqual([1, 1, 0])
    expect(momentum.map((b) => b.cumTeam)).toEqual([2, 2, 3])
    expect(momentum.map((b) => b.cumEnemy)).toEqual([1, 2, 2])
  })

  it('(b) premier bin non nul → trend "up"', () => {
    const bins = [bin(0, 30), bin(30, 60)]
    const events = [killEvent('ally1', 5_000)] // bin0
    const { momentum } = computeMomentumBins(bins, events, META)

    expect(momentum[0].delta).toBe(1)
    expect(momentum[0].trend).toBe('up')
  })

  it('(c) bin à delta 0 entre deux bins signés → pas de barre, cumuls conservés', () => {
    const bins = [bin(0, 30), bin(30, 60), bin(60, 90)]
    const events = [
      killEvent('ally1', 5_000), // bin0 → +1
      killEvent('ally1', 40_000), // bin1 → allié
      killEvent('enemy1', 45_000), // bin1 → ennemi (delta 0)
      killEvent('enemy1', 70_000), // bin2 → -1
    ]
    const { momentum } = computeMomentumBins(bins, events, META)

    expect(momentum.map((b) => b.delta)).toEqual([1, 0, -1])
    // delta 0 → pas de barre : trend neutralisé à 'down'.
    expect(momentum[1].trend).toBe('down')
    // Cumuls conservés à travers le bin neutre.
    expect(momentum[1].cumTeam).toBe(2)
    expect(momentum[1].cumEnemy).toBe(1)
    // Bin suivant : bascule ennemie relative au bin neutre (0) → 'up'.
    expect(momentum[2].trend).toBe('up')
  })

  it('(d) event au-delà du dernier bin → clampé dans le dernier bin (parité backend)', () => {
    const bins = [bin(0, 30), bin(30, 60)]
    const events = [killEvent('ally1', 80_000)] // 80 s au-delà du dernier bin [30,60)
    const { momentum, kills } = computeMomentumBins(bins, events, META)

    // tug_of_war.go clampe l'event dans le dernier bin : le front fait de même.
    expect(kills).toHaveLength(1)
    expect(kills[0].binIdx).toBe(1)
    expect(momentum[1].delta).toBe(1)
    expect(momentum[1].teamKills).toBe(1)
  })

  it('(d2) event avant le premier bin → ignoré (parité backend TimeMS < 0)', () => {
    const bins = [bin(0, 30)]
    const events = [killEvent('ally1', -5_000)] // -5 s, avant le premier bin
    const { momentum, kills } = computeMomentumBins(bins, events, META)

    expect(kills).toHaveLength(0)
    expect(momentum[0].delta).toBe(0)
    expect(momentum[0].teamKills).toBe(0)
  })

  it('(e) event sans actor_xuid ou sans event_time_ms → ignoré', () => {
    const bins = [bin(0, 30)]
    const events = [
      killEvent(null, 5_000), // actor_xuid manquant
      killEvent('ally1', null), // event_time_ms manquant
      { ...killEvent('ally1', 5_000), event_type: 'medal' }, // pas un kill
      killEvent('ghost', 5_000), // acteur hors scoreboard
    ]
    const { momentum, kills } = computeMomentumBins(bins, events, META)

    expect(kills).toHaveLength(0)
    expect(momentum[0].delta).toBe(0)
  })

  it('(f) kills d’une seule équipe → deltas de même signe, trends corrects', () => {
    const bins = [bin(0, 30), bin(30, 60), bin(60, 90), bin(90, 120)]
    const events = [
      killEvent('ally1', 5_000), // bin0 → +1
      killEvent('ally1', 35_000), // bin1
      killEvent('ally2', 40_000), // bin1 → +2
      killEvent('ally1', 65_000), // bin2 → +1
      killEvent('ally1', 95_000), // bin3
      killEvent('ally2', 100_000), // bin3
      killEvent('ally1', 105_000), // bin3 → +3
    ]
    const { momentum } = computeMomentumBins(bins, events, META)

    expect(momentum.map((b) => b.delta)).toEqual([1, 2, 1, 3])
    expect(momentum.map((b) => b.enemyKills)).toEqual([0, 0, 0, 0])
    // +1(prev0)up, +2(prev1)up, +1(prev2)down, +3(prev1)up
    expect(momentum.map((b) => b.trend)).toEqual(['up', 'up', 'down', 'up'])
  })

  it('(g) côté négatif : renforcement (−2→−5) = "up", essoufflement (−5→−1) = "down"', () => {
    const bins = [bin(0, 30), bin(30, 60), bin(60, 90)]
    const events = [
      killEvent('enemy1', 5_000),
      killEvent('enemy1', 10_000), // bin0 → -2
      killEvent('enemy1', 35_000),
      killEvent('enemy1', 40_000),
      killEvent('enemy1', 45_000),
      killEvent('enemy1', 50_000),
      killEvent('enemy1', 55_000), // bin1 → -5
      killEvent('enemy1', 65_000), // bin2 → -1
    ]
    const { momentum } = computeMomentumBins(bins, events, META)

    expect(momentum.map((b) => b.delta)).toEqual([-2, -5, -1])
    // -2(prev0)up, -5(prev-2)up (momentum ennemi se renforce), -1(prev-5)down
    expect(momentum.map((b) => b.trend)).toEqual(['up', 'up', 'down'])
  })
})
