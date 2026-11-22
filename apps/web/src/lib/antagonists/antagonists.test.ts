import { describe, expect, it } from 'vitest'

import { buildAntagonists, type AntagonistKill } from './antagonists'

const SUBJECT = 'S'

/** Élimination : `k(tueur, victime, instant)`. */
function k(killer: string, victim: string, t: number): AntagonistKill {
  return { killer, victim, t }
}

/** Sujet éliminé par `foe` à l'instant `t` (alimente le rôle némésis). */
function diedTo(foe: string, t: number): AntagonistKill {
  return k(foe, SUBJECT, t)
}

/** Sujet qui élimine `foe` à l'instant `t` (alimente le rôle souffre-douleur). */
function killed(foe: string, t: number): AntagonistKill {
  return k(SUBJECT, foe, t)
}

/** Camps : tout autre joueur dont la clé commence par `e` est un adversaire. */
const campEnnemi = (other: string): boolean => other.startsWith('e')

describe('buildAntagonists : changements de titre', () => {
  it('donne le titre au premier adversaire qui atteint un compte positif', () => {
    const a = buildAntagonists([diedTo('e1', 1000)], SUBJECT, campEnnemi)
    expect(a.changes).toEqual([
      { t: 1000, role: 'nemesis', kind: 'prise', holder: 'e1', previousHolder: null, count: 1, previousCount: 0 },
    ])
  })

  it('change de main sur un dépassement strict seulement', () => {
    const a = buildAntagonists(
      [diedTo('e1', 1000), diedTo('e2', 2000), diedTo('e2', 3000)],
      SUBJECT,
      campEnnemi,
    )
    expect(a.changes.map((c) => [c.t, c.kind, c.holder, c.previousHolder, c.count, c.previousCount])).toEqual([
      [1000, 'prise', 'e1', null, 1, 0],
      [2000, 'egalite', 'e2', 'e1', 1, 1],
      [3000, 'prise', 'e2', 'e1', 2, 1],
    ])
  })

  it("garde le titre au tenant sur une égalité et la signale par kind 'egalite'", () => {
    const a = buildAntagonists(
      [diedTo('e1', 1000), diedTo('e1', 2000), diedTo('e2', 3000), diedTo('e2', 4000)],
      SUBJECT,
      campEnnemi,
    )
    expect(a.changes.map((c) => c.kind)).toEqual(['prise', 'egalite'])
    expect(a.changes[1]).toEqual({
      t: 4000,
      role: 'nemesis',
      kind: 'egalite',
      holder: 'e2',
      previousHolder: 'e1',
      count: 2,
      previousCount: 2,
    })
    expect(a.at(4000).nemesis).toEqual({ holder: 'e1', count: 2, tiedWith: ['e2'] })
  })

  it('laisse un même adversaire tenir les deux titres', () => {
    const a = buildAntagonists([diedTo('e1', 1000), killed('e1', 2000)], SUBJECT, campEnnemi)
    expect(a.changes.map((c) => [c.role, c.holder])).toEqual([
      ['nemesis', 'e1'],
      ['victim', 'e1'],
    ])
    expect(a.at(2000)).toEqual({
      nemesis: { holder: 'e1', count: 1, tiedWith: [] },
      victim: { holder: 'e1', count: 1, tiedWith: [] },
    })
  })

  it("ignore un tueur qui n'est pas un adversaire (trahison)", () => {
    const a = buildAntagonists([diedTo('a1', 1000), diedTo('a1', 2000)], SUBJECT, campEnnemi)
    expect(a.changes).toEqual([])
    expect(a.at(5000)).toEqual({ nemesis: null, victim: null })
  })

  it("ignore une victime qui n'est pas un adversaire", () => {
    const a = buildAntagonists([killed('a1', 1000)], SUBJECT, campEnnemi)
    expect(a.changes).toEqual([])
    expect(a.at(5000)).toEqual({ nemesis: null, victim: null })
  })

  it('compte tous les autres joueurs en mêlée générale', () => {
    const a = buildAntagonists(
      [diedTo('x', 1000), killed('y', 2000), diedTo('x', 3000)],
      SUBJECT,
      () => true,
    )
    expect(a.at(3000)).toEqual({
      nemesis: { holder: 'x', count: 2, tiedWith: [] },
      victim: { holder: 'y', count: 1, tiedWith: [] },
    })
  })

  it("ignore les éliminations où le sujet n'est ni tueur ni victime", () => {
    const a = buildAntagonists([k('e1', 'e2', 1000), k('e2', 'e1', 2000)], SUBJECT, () => true)
    expect(a.changes).toEqual([])
  })

  it("ignore l'auto-élimination du sujet", () => {
    const a = buildAntagonists([k(SUBJECT, SUBJECT, 1000)], SUBJECT, () => true)
    expect(a.changes).toEqual([])
    expect(a.at(2000)).toEqual({ nemesis: null, victim: null })
  })

  it('écarte les doublons (même tueur, même victime, même instant)', () => {
    const a = buildAntagonists(
      [diedTo('e1', 1000), diedTo('e1', 1000), diedTo('e2', 2000), diedTo('e2', 2000)],
      SUBJECT,
      campEnnemi,
    )
    expect(a.at(2000).nemesis).toEqual({ holder: 'e1', count: 1, tiedWith: ['e2'] })
    expect(a.changes.map((c) => c.kind)).toEqual(['prise', 'egalite'])
  })

  it('garde deux éliminations identiques à des instants différents', () => {
    const a = buildAntagonists([diedTo('e1', 1000), diedTo('e1', 1001)], SUBJECT, campEnnemi)
    expect(a.at(1001).nemesis).toEqual({ holder: 'e1', count: 2, tiedWith: [] })
  })

  it("donne le même résultat quel que soit l'ordre d'entrée", () => {
    const kills = [
      diedTo('e1', 1000),
      diedTo('e2', 1000),
      killed('e3', 1500),
      diedTo('e2', 2000),
      diedTo('e1', 2500),
      diedTo('e1', 2500),
      killed('e4', 3000),
      killed('e4', 3500),
    ]
    const ref = buildAntagonists(kills, SUBJECT, campEnnemi)
    const shuffled = [kills[5], kills[2], kills[7], kills[0], kills[4], kills[1], kills[6], kills[3]]
    const other = buildAntagonists(shuffled, SUBJECT, campEnnemi)
    expect(other.changes).toEqual(ref.changes)
    for (const t of [0, 1000, 1500, 2000, 2500, 3000, 3500, 9999]) {
      expect(other.at(t)).toEqual(ref.at(t))
    }
  })

  it("ne modifie pas la liste d'entrée", () => {
    const kills = [diedTo('e1', 2000), diedTo('e2', 1000)]
    const copie = kills.map((x) => ({ ...x }))
    buildAntagonists(kills, SUBJECT, campEnnemi)
    expect(kills).toEqual(copie)
  })
})

describe("buildAntagonists : lecture à l'instant", () => {
  const kills = [
    diedTo('e1', 1000),
    diedTo('e2', 2000),
    diedTo('e3', 3000),
    diedTo('e2', 4000),
    diedTo('e1', 5000),
    diedTo('e3', 6000),
    killed('e1', 7000),
  ]
  const a = buildAntagonists(kills, SUBJECT, campEnnemi)

  it('rend null pour les deux rôles avant toute élimination comptée', () => {
    expect(a.at(999)).toEqual({ nemesis: null, victim: null })
    expect(a.at(-5)).toEqual({ nemesis: null, victim: null })
  })

  it("compte l'élimination dont l'instant est exactement celui lu", () => {
    expect(a.at(1000).nemesis).toEqual({ holder: 'e1', count: 1, tiedWith: [] })
    expect(a.at(4000).nemesis).toEqual({ holder: 'e2', count: 2, tiedWith: [] })
  })

  it("garde l'état d'avant entre deux éliminations", () => {
    expect(a.at(3999).nemesis).toEqual({ holder: 'e1', count: 1, tiedWith: ['e2', 'e3'] })
    expect(a.at(1999).nemesis).toEqual({ holder: 'e1', count: 1, tiedWith: [] })
  })

  it('rend les joueurs à égalité justes à un instant intermédiaire', () => {
    expect(a.at(3000).nemesis).toEqual({ holder: 'e1', count: 1, tiedWith: ['e2', 'e3'] })
    expect(a.at(5000).nemesis).toEqual({ holder: 'e2', count: 2, tiedWith: ['e1'] })
    expect(a.at(6000).nemesis).toEqual({ holder: 'e2', count: 2, tiedWith: ['e1', 'e3'] })
    expect(a.at(6500).victim).toBeNull()
    expect(a.at(7000).victim).toEqual({ holder: 'e1', count: 1, tiedWith: [] })
  })

  it('rend au dernier instant le calcul sur toute la liste', () => {
    const final = a.at(Number.MAX_SAFE_INTEGER)
    expect(final).toEqual(a.at(7000))
    expect(final.nemesis).toEqual({ holder: 'e2', count: 2, tiedWith: ['e1', 'e3'] })
  })

  it('rend des tableaux que le lecteur peut modifier sans fausser les lectures suivantes', () => {
    const lu = a.at(6000).nemesis
    lu?.tiedWith.push('zzz')
    expect(a.at(6000).nemesis?.tiedWith).toEqual(['e1', 'e3'])
  })

  it('reste juste sur une longue partie (même résultat que le parcours complet)', () => {
    const longues: AntagonistKill[] = []
    for (let i = 1; i <= 400; i++) longues.push(diedTo(`e${(i * 7) % 5}`, i * 10))
    const idx = buildAntagonists(longues, SUBJECT, campEnnemi)
    for (const t of [5, 10, 333, 1000, 2001, 4000, 9999]) {
      const prefixe = longues.filter((x) => x.t <= t)
      const attendu = buildAntagonists(prefixe, SUBJECT, campEnnemi).at(Number.MAX_SAFE_INTEGER)
      expect(idx.at(t)).toEqual(attendu)
    }
  })
})
