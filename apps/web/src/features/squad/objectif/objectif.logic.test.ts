import { describe, expect, it } from 'vitest'

import {
  buildEveningsView,
  buildObjectiveBalance,
  buildObjectiveSheets,
  buildSessionFil,
  familyMix,
  median,
} from './objectif.logic'
import {
  block0709,
  block2209,
  history0709,
  history0709Evenings,
  history2209Evenings,
} from './objectif.fixtures'

const r1 = (v: number | null | undefined) => (v == null ? null : Math.round(v * 1000) / 10)

describe('buildObjectiveBalance — rapport de force par famille (07/09)', () => {
  const fams = buildObjectiveBalance(block0709())

  it('un cadre par famille, dans l’ordre de la soirée, avec son nombre de matchs', () => {
    expect(fams.map((f) => [f.family, f.matches])).toEqual([
      ['zones_strongholds', 4],
      ['ctf', 3],
    ])
  })

  it('les actions rangées sous Prendre / Défendre / Tenir, notre camp contre l’adversaire', () => {
    const bases = fams[0]
    expect(bases.roles.map((r) => r.role)).toEqual(['take', 'defend', 'hold'])
    const caps = bases.roles[0].lines[0]
    expect([caps.key, caps.us, caps.them]).toEqual(['zone_captures', 71, 79])
    const hold = bases.roles[2].lines[0]
    expect(hold.duration).toBe(true)
    expect(Math.round(hold.us * 10) / 10).toBe(1067.8)
    expect(Math.round((hold.us + hold.them) * 10) / 10).toBe(1993.9)
  })

  it('la colonne facultative (prises nettes) reste une ligne du rapport de force', () => {
    const take = fams[1].roles[0].lines.map((l) => [l.key, l.us, l.us + l.them])
    expect(take).toEqual([
      ['flag_captures', 1, 10],
      ['flag_capture_assists', 1, 8],
      ['flag_steals', 15, 31],
      ['flag_returners_killed', 0, 6],
      ['flag_grabs_net', 29, 70],
    ])
  })
})

describe('buildSessionFil — rapport de force au fil de la session (07/09)', () => {
  const fil = buildSessionFil(block0709(), history0709())

  it('chaque match pèse pareil : le cumul final vaut 39,0 / 36,8 / 43,8 % (L3.10)', () => {
    const last = fil[fil.length - 1]
    expect([r1(last.roles.take.cumulative), r1(last.roles.defend.cumulative), r1(last.roles.hold.cumulative)]).toEqual([
      39.0, 36.8, 43.8,
    ])
  })

  it('les parts par match et leur volume (la prise nette facultative n’entre dans aucun rôle)', () => {
    expect(fil.map((m) => m.matchId)).toEqual(['b1', 'b2', 'b3', 'b4', 'd1', 'd2', 'd3'])
    expect([fil[4].roles.take.us, fil[4].roles.take.lobby]).toEqual([7, 20])
    expect(r1(fil[0].roles.take.share)).toBe(51.8)
  })

  it('résultat, score et dominance joints depuis l’historique par match_id (L3.2)', () => {
    expect([fil[2].outcome, fil[2].score, fil[2].map]).toEqual(['win', '200–183', 'Illusion'])
    expect(fil[1].dominance).toBe(2)
    expect(fil[0].dominance).toBeUndefined()
  })

  it('un match sans ligne d’historique garde sa case, sans résultat ni erreur (L3.2)', () => {
    const d2 = fil.find((m) => m.matchId === 'd2')
    expect(d2).toBeDefined()
    expect([d2?.outcome, d2?.score, d2?.dominance]).toEqual([null, null, undefined])
    expect(d2?.map).toBe('Domicile')
  })

  it('un rôle que le lobby n’a pas joué (0 sur 0) ne pèse pas dans le cumul', () => {
    const b = block0709()
    const m0 = b.matches![0]
    for (const p of m0.objective!.players!) p.values = { ...p.values, zone_secures: 0, zone_defensive_kills: 0 }
    const f = buildSessionFil(b, [])
    expect(f[0].roles.defend.share).toBeNull()
    expect(f[0].roles.defend.cumulative).toBeNull()
    expect(r1(f[1].roles.defend.cumulative)).toBe(23.1)
  })
})

describe('buildObjectiveSheets — répartition de l’objectif dans l’escouade (22/09)', () => {
  const sheets = buildObjectiveSheets(block2209())
  const line = (key: string) => sheets.families[0].lines.find((l) => l.key === key)!

  it('une fiche par joueur de l’escouade plus le reste du camp', () => {
    expect(sheets.owners.map((o) => o.xuid)).toEqual(['xj', 'xc', 'xm', null])
  })

  it('JGtm : 4 drapeaux capturés, 3 volés (L3.10)', () => {
    expect(line('flag_captures').values).toEqual([4, 0, 0, 0])
    expect(line('flag_steals').values).toEqual([3, 4, 5, 2])
  })

  it('mêmes lignes dans le même ordre ; l’échelle d’une ligne est son maximum sur les fiches', () => {
    expect(sheets.families[0].lines.map((l) => l.key)).toHaveLength(9)
    expect(line('flag_steals').max).toBe(5)
    expect(line('time_as_flag_carrier_seconds').values.map((v) => Math.round(v * 10) / 10)).toEqual([64.9, 19.7, 44.8, 8.1])
  })

  it('pied de fiche : Prendre / Défendre hors colonnes facultatives, Tenir en secondes', () => {
    expect(sheets.roleTotals[0].map((v) => Math.round(v * 10) / 10)).toEqual([7, 10, 64.9])
    expect(sheets.roleTotals[3].map((v) => Math.round(v * 10) / 10)).toEqual([3, 6, 8.1])
  })

  it('rôle dominant = le rôle où le joueur pèse le plus dans notre camp', () => {
    // JGtm : 7/22 prendre, 10/33 défendre, 64,9/137,5 tenir → Tenir.
    expect(sheets.dominant).toEqual(['hold', 'take', 'defend', 'defend'])
  })
})

describe('buildEveningsView — rapport de force, soirée après soirée', () => {
  it('07/09 : dix soirées précédentes puis ce soir, médianes des précédentes', () => {
    const v = buildEveningsView(history0709Evenings())
    expect(v.kind).toBe('chart')
    if (v.kind !== 'chart') return
    expect(v.points).toHaveLength(11)
    expect(v.points[10].current).toBe(true)
    expect([v.points[10].shares.take, v.points[10].shares.defend, v.points[10].shares.hold].map((x) => Math.round(x! * 10) / 10)).toEqual([
      39.0, 36.8, 43.8,
    ])
    expect(v.medians.take).toBeCloseTo(50.0, 5)
  })

  it('22/09 : sous le minimum de trois, la note et les comptes de la composition', () => {
    expect(buildEveningsView(history2209Evenings())).toEqual({ kind: 'belowMinimum', matches: 2, below: 24, withObjective: 49 })
  })

  it('première soirée à objectif : aucune soirée précédente', () => {
    const h = { ...history0709Evenings(), previous: [] }
    expect(buildEveningsView(h).kind).toBe('noHistory')
  })
})

describe('utilitaires', () => {
  it('median', () => {
    expect(median([])).toBeNull()
    expect(median([3, 1, 2])).toBe(2)
    expect(median([4, 1, 2, 3])).toBe(2.5)
  })
  it('familyMix', () => {
    expect(familyMix([{ family: 'zones_strongholds', matches: 4 }, { family: 'ctf', matches: 3 }], { zones_strongholds: 'B', ctf: 'D' })).toBe(
      '4 B · 3 D',
    )
  })
})
