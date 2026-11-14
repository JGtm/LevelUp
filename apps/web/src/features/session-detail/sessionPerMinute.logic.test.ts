/**
 * Tests sessionPerMinute.logic — cadences par minute d'une session et bornes d'axe communes à
 * deux sessions comparées.
 */
import { describe, expect, it } from 'vitest'

import type { SessionDetailMatchRow } from '@/lib/api/types'

import { perMinuteDomains, sessionPerMinuteRates } from './sessionPerMinute.logic'

function match(o: Partial<SessionDetailMatchRow>): SessionDetailMatchRow {
  return {
    match_id: 'm',
    start_time: '2026-01-01T00:00:00Z',
    outcome: 2,
    playlist_name: 'P',
    pair_name: 'Slayer',
    is_ranked: false,
    kills: 0,
    deaths: 0,
    assists: 0,
    ...o,
  } as SessionDetailMatchRow
}

describe('sessionPerMinuteRates', () => {
  it('somme des comptes / somme des minutes, dégâts compris', () => {
    const r = sessionPerMinuteRates(
      [
        match({ kills: 10, deaths: 5, assists: 2, duration_seconds: 600, damage_dealt: 3000, damage_taken: 1200 }),
        match({ kills: 3, deaths: 1, assists: 1, duration_seconds: 180, damage_dealt: 900, damage_taken: 300 }),
      ],
      true,
    )
    expect(r).not.toBeNull()
    expect(r!.frags).toBeCloseTo(13 / 13, 10)
    expect(r!.deaths).toBeCloseTo(6 / 13, 10)
    expect(r!.assists).toBeCloseTo(3 / 13, 10)
    expect(r!.damageDealt).toBeCloseTo(3900 / 13, 10)
    expect(r!.damageTaken).toBeCloseTo(1500 / 13, 10)
  })

  it('un match sans durée est exclu des deux termes', () => {
    const r = sessionPerMinuteRates(
      [
        match({ kills: 10, duration_seconds: 600, damage_dealt: 1000 }),
        match({ kills: 99, duration_seconds: 0, damage_dealt: 999999 }),
        match({ kills: 99, damage_dealt: 999999 }),
      ],
      true,
    )
    expect(r!.frags).toBeCloseTo(1, 10)
    expect(r!.damageDealt).toBeCloseTo(100, 10)
  })

  it('un dégât inconnu exclut le match de CE dégât seulement', () => {
    const r = sessionPerMinuteRates(
      [
        match({ kills: 10, duration_seconds: 600, damage_dealt: 1000, damage_taken: 400 }),
        match({ kills: 10, duration_seconds: 600 }), // dégâts inconnus : compte pour les frags, pas pour les dégâts
      ],
      true,
    )
    expect(r!.frags).toBeCloseTo(1, 10)
    expect(r!.damageDealt).toBeCloseTo(100, 10)
    expect(r!.damageTaken).toBeCloseTo(40, 10)
  })

  it('aucun dégât connu : absent (null), pas zéro', () => {
    const r = sessionPerMinuteRates([match({ kills: 10, duration_seconds: 600 })], true)
    expect(r!.damageDealt).toBeNull()
    expect(r!.damageTaken).toBeNull()
  })

  it('titre sans dégâts subis : absents même avec une valeur nulle rangée', () => {
    const r = sessionPerMinuteRates([match({ kills: 10, duration_seconds: 600, damage_dealt: 1000, damage_taken: 0 })], false)
    expect(r!.damageDealt).toBeCloseTo(100, 10)
    expect(r!.damageTaken).toBeNull()
  })

  it('aucune durée : pas de cadence', () => {
    expect(sessionPerMinuteRates([match({ kills: 10 })], true)).toBeNull()
    expect(sessionPerMinuteRates([], true)).toBeNull()
  })
})

describe('perMinuteDomains', () => {
  const A = [
    match({ kills: 5, deaths: 2, assists: 3, duration_seconds: 60 }),
    match({ kills: 1, deaths: 4, assists: 1, duration_seconds: 60 }),
  ]
  const B = [match({ kills: 10, deaths: 2, assists: 5, duration_seconds: 120 })]

  it('sans dégâts : axe de gauche [−morts, +max], pas d’axe de droite', () => {
    // A : frags 3 / assistances 2 / morts 3 ; B : frags 5 / assistances 2,5 / morts 1.
    expect(perMinuteDomains([A, B], true)).toEqual({ fdaMinute: [-3, 5] })
  })

  it('avec dégâts : deux axes qui partagent la ligne zéro et contiennent toutes les valeurs', () => {
    const withDamage = [
      match({ kills: 5, deaths: 2, assists: 3, duration_seconds: 60, damage_dealt: 800, damage_taken: 400 }),
    ]
    const d = perMinuteDomains([withDamage, B], true)!
    expect(d.damageMinute).toBeDefined()
    const share = (b: [number, number]) => -b[0] / (b[1] - b[0])
    expect(share(d.fdaMinute)).toBeCloseTo(share(d.damageMinute!), 10)
    expect(d.fdaMinute[1]).toBeGreaterThanOrEqual(5) // frags de A : 5/min
    expect(d.damageMinute![1]).toBeGreaterThanOrEqual(800)
    expect(d.damageMinute![0]).toBeLessThanOrEqual(-400)
  })

  it('aucune session exploitable : undefined (auto-scale)', () => {
    expect(perMinuteDomains([[], []], true)).toBeUndefined()
  })
})
