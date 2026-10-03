/**
 * _riposte.test.ts — la projection du bloc « Riposte ».
 *
 * Les trois règles de `_riposte.ts` sont celles qui se cassent en silence : l'échelle
 * commune, l'ordre des camps (le mien d'abord) et le tri des joueurs. Elles sont ici.
 */
import { describe, expect, it } from 'vitest'

import type { MatchRiposteBlock } from '@/lib/api/types'
import { buildRiposteModel, riposteFraction, splitCouples } from './_riposte'

const block: MatchRiposteBlock = {
  fenetre_ms: 5000,
  measured_deaths: 6,
  deaths: [
    {
      victim_xuid: 'me',
      victim_gamertag: 'JGtm',
      victim_team_id: 1,
      killer_xuid: 'vex',
      time_ms: 1000,
      avenged: true,
      avenger_xuid: 'kaya',
      avenger_gamertag: 'Kaya',
      delai_ms: 3100,
      vengeable: true,
    },
    {
      victim_xuid: 'kaya',
      victim_gamertag: 'Kaya',
      victim_team_id: 1,
      killer_xuid: 'vex',
      time_ms: 4000,
      avenged: true,
      avenger_xuid: 'me',
      avenger_gamertag: 'JGtm',
      delai_ms: 900,
      vengeable: true,
    },
    {
      victim_xuid: 'vex',
      victim_gamertag: 'Vex',
      victim_team_id: 0,
      killer_xuid: 'me',
      time_ms: 5000,
      avenged: false,
      vengeable: true,
    },
  ],
  players: [
    { xuid: 'vex', gamertag: 'Vex', team_id: 0, deaths_avenged: 0, ripostes: 0 },
    { xuid: 'me', gamertag: 'JGtm', team_id: 1, deaths_avenged: 1, ripostes: 1 },
    { xuid: 'kaya', gamertag: 'Kaya', team_id: 1, deaths_avenged: 1, ripostes: 4 },
  ],
}

describe('buildRiposteModel', () => {
  it('range MON camp en premier', () => {
    const model = buildRiposteModel(block, 'me')
    expect(model.camps.map((c) => c.key)).toEqual(['t1', 't0'])
  })

  it('sans xuid connu, les camps suivent le team_id', () => {
    const model = buildRiposteModel(block, null)
    expect(model.camps.map((c) => c.key)).toEqual(['t0', 't1'])
  })

  it('l’échelle est COMMUNE : le plus grand compte du match, tous camps confondus', () => {
    expect(buildRiposteModel(block, 'me').scaleMax).toBe(4)
  })

  it('trie chaque camp sur les ripostes portées, décroissant', () => {
    const mien = buildRiposteModel(block, 'me').camps[0]
    expect(mien.rows.map((r) => r.xuid)).toEqual(['kaya', 'me'])
  })

  it('attache à chaque joueur les couples nommés de ses deux comptes', () => {
    const mien = buildRiposteModel(block, 'me').camps[0]
    const moi = mien.rows.find((r) => r.xuid === 'me')
    expect(moi?.avengedBy).toEqual([{ name: 'Kaya', delaiMs: 3100 }])
    expect(moi?.avengedFor).toEqual([{ name: 'Kaya', delaiMs: 900 }])
  })

  it('ne compte que les morts VENGÉES au total du pied de carte', () => {
    const model = buildRiposteModel(block, 'me')
    expect(model.avengedTotal).toBe(2)
    expect(model.measuredDeaths).toBe(6)
  })

  it('le joueur sans camp forme un camp à part, rangé en dernier', () => {
    const model = buildRiposteModel(
      {
        ...block,
        players: [
          { xuid: 'ffa', gamertag: 'Solo', deaths_avenged: 0, ripostes: 0 },
          ...(block.players ?? []),
        ],
      },
      'me',
    )
    expect(model.camps.map((c) => c.key)).toEqual(['t1', 't0', 'none'])
    expect(model.camps[2].teamSide).toBeNull()
  })

  it('tient un bloc sans aucune ligne (tableaux nuls du contrat)', () => {
    const model = buildRiposteModel(
      { fenetre_ms: 5000, measured_deaths: 0, deaths: null, players: null },
      'me',
    )
    expect(model.camps).toEqual([])
    expect(model.scaleMax).toBe(1)
    expect(model.avengedTotal).toBe(0)
  })
})

describe('splitCouples', () => {
  it('rend tout sous la borne', () => {
    const couples = [{ name: 'A', delaiMs: 1 }]
    expect(splitCouples(couples)).toEqual({ shown: couples, rest: 0 })
  })

  it('coupe à cinq lignes et compte le reste', () => {
    const couples = Array.from({ length: 8 }, (_, i) => ({ name: `P${i}`, delaiMs: i }))
    const { shown, rest } = splitCouples(couples)
    expect(shown).toHaveLength(5)
    expect(rest).toBe(3)
  })
})

describe('riposteFraction', () => {
  it('borne la part entre 0 et 1', () => {
    expect(riposteFraction(2, 4)).toBe(0.5)
    expect(riposteFraction(9, 4)).toBe(1)
    expect(riposteFraction(1, 0)).toBe(0)
  })
})
