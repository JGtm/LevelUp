import { describe, it, expect } from 'vitest'

import { firstBloodMaxSec, toFirstBloodSeries } from './firstBlood'

describe('toFirstBloodSeries', () => {
  it('retourne un tableau vide sur payload absent', () => {
    expect(toFirstBloodSeries(null)).toEqual([])
    expect(toFirstBloodSeries(undefined)).toEqual([])
    expect(toFirstBloodSeries([])).toEqual([])
  })

  it('projette snake_case → props du chart, null conservé', () => {
    const got = toFirstBloodSeries([
      {
        player: 'Madina',
        matches: [
          { match_id: 'm1', first_kill_sec: 12.3, first_death_sec: 45 },
          { match_id: 'm2', first_kill_sec: null, first_death_sec: null },
        ],
      },
    ])
    expect(got).toEqual([
      {
        player: 'Madina',
        matches: [
          { matchId: 'm1', firstKillSec: 12.3, firstDeathSec: 45 },
          { matchId: 'm2', firstKillSec: null, firstDeathSec: null },
        ],
      },
    ])
  })

  it('tolère une liste de matchs nulle (contrat OpenAPI nullable)', () => {
    expect(toFirstBloodSeries([{ player: 'Solo', matches: null }])).toEqual([
      { player: 'Solo', matches: [] },
    ])
  })
})

describe('firstBloodMaxSec', () => {
  it('plancher à 300 s sans donnée', () => {
    expect(firstBloodMaxSec([])).toBe(300)
    expect(firstBloodMaxSec([{ player: 'A', matches: [] }])).toBe(300)
  })

  it('reste au plancher tant que les événements tiennent sous 5 min', () => {
    const series = [
      { player: 'A', matches: [{ matchId: 'm1', firstKillSec: 20, firstDeathSec: 60 }] },
    ]
    expect(firstBloodMaxSec(series)).toBe(300)
  })

  it('étend à la minute supérieure au-delà du plancher', () => {
    const series = [
      { player: 'A', matches: [{ matchId: 'm1', firstKillSec: 310, firstDeathSec: 425 }] },
    ]
    // p99 sur 2 valeurs = 425 → arrondi 480.
    expect(firstBloodMaxSec(series)).toBe(480)
  })

  it("ignore la valeur aberrante isolée (p99, pas max)", () => {
    const matches = Array.from({ length: 100 }, (_, i) => ({
      matchId: `m${i}`,
      firstKillSec: 30,
      firstDeathSec: i === 99 ? 3000 : 90,
    }))
    // 200 valeurs, p99 = 90 → sous le plancher → 300 (et pas 3000).
    expect(firstBloodMaxSec([{ player: 'A', matches }])).toBe(300)
  })
})
