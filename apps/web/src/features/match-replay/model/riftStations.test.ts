/**
 * LA FAILLE SE DÉPLACE, ET ELLE S'ÉTEINT — les deux affirmations que ce lot fait, et les deux
 * qu'il refuse de faire (rien avant le premier échange, rien à demeure).
 *
 * Chaque cas ci-dessous a son fait mesuré : le va-et-vient à 0,09 m près (R1 §4.4), la fin par
 * épuisement ou par expiration jusqu'à 16,5 s après le dernier échange, la mort du porteur qui
 * clôt sans `spent`, et le `from` sous saut de compteur qui ne ferme rien (R1 §1, JGtm).
 */
import { describe, expect, it } from 'vitest'

import { riftStations, riftStationsAt } from './riftStations'
import type { ReplayDocumentReady } from './replayNormalize'
import { testReplayDoc } from '../test/testDoc'

type Translocation = ReplayDocumentReady['translocations'][number]
type Change = ReplayDocumentReady['equipmentChanges'][number]

/** Une téléportation SITUÉE : le joueur quitte `from` et atteint `to`. */
function saut(slot: number, t: number, from: [number, number], to: [number, number]): Translocation {
  return { slot, t, fx: from[0], fy: from[1], fz: 0, tx: to[0], ty: to[1], tz: 0 }
}

function spent(slot: number, t: number, from: number, gap?: number): Change {
  return { slot, t, kind: 'spent', r: -1, from, ...(gap === undefined ? {} : { gap }) }
}

/** Une vie qui couvre [start, end] — la borne ultime de toute faille du slot. */
function vie(slot: number, start: number, end: number) {
  return { slot, team: 0, startFrame: start, endFrame: end, points: [{ t: start, x: 0, y: 0 }] }
}

const LIBELLES = { '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' } }

function doc(over: {
  translocations?: Translocation[]
  equipmentChanges?: Change[]
  tracks?: ReturnType<typeof vie>[]
}) {
  return testReplayDoc({
    abilityLabels: LIBELLES,
    tracks: over.tracks ?? [vie(3, 0, 400)],
    translocations: over.translocations ?? [],
    equipmentChanges: over.equipmentChanges ?? [],
  })
}

describe('riftStations — rien avant le premier échange', () => {
  it('un film sans translocation ne pose AUCUNE faille', () => {
    expect(riftStations(doc({}))).toEqual([])
  })

  it('la première station commence à l ÉCHANGE, jamais avant', () => {
    const s = riftStations(doc({ translocations: [saut(3, 100, [12, 5], [40, 5])] }))
    expect(s).toHaveLength(1)
    expect(s[0].t0).toBe(100)
    expect(riftStationsAt(s, 99)).toEqual([])
    expect(riftStationsAt(s, 100)).toHaveLength(1)
  })

  it('une translocation SANS positions ne situe rien — le geste reste daté ailleurs', () => {
    expect(riftStations(doc({ translocations: [{ slot: 3, t: 100 }] }))).toEqual([])
  })
})

describe('riftStations — le VA-ET-VIENT : la faille est au point de DÉPART, et elle se déplace', () => {
  it('elle se pose là où le joueur PART, pas là où il arrive', () => {
    const s = riftStations(doc({ translocations: [saut(3, 100, [12, 5], [40, 5])] }))
    expect({ x: s[0].x, y: s[0].y }).toEqual({ x: 12, y: 5 })
  })

  it('chaque échange suivant la DÉPLACE, et clôt la station précédente à l image d avant', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5]), saut(3, 160, [40, 5], [12, 5])],
        tracks: [vie(3, 0, 400)],
      }),
    )
    expect(s).toHaveLength(2)
    expect([s[0].t0, s[0].t1]).toEqual([100, 159])
    expect({ x: s[0].x, y: s[0].y }).toEqual({ x: 12, y: 5 })
    expect(s[1].t0).toBe(160)
    expect({ x: s[1].x, y: s[1].y }).toEqual({ x: 40, y: 5 })
    // À une image donnée, UNE SEULE station de cette vie est active : la faille est unique.
    expect(riftStationsAt(s, 159)).toHaveLength(1)
    expect(riftStationsAt(s, 160)).toHaveLength(1)
    expect(riftStationsAt(s, 160)[0].x).toBe(40)
  })

  it('deux vies portent chacune la leur, sans se mélanger', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [1, 1], [40, 5]), saut(9, 120, [70, 2], [10, 9])],
        tracks: [vie(3, 0, 400), vie(9, 0, 400)],
      }),
    )
    expect(s.map((x) => x.slot)).toEqual([3, 9])
  })
})

describe('riftStations — la fin est MESURÉE, jamais à demeure', () => {
  it('l ÉPUISEMENT par l usage final ferme à l image du `spent`', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(3, 101, 11)],
      }),
    )
    expect(s[0].t1).toBe(101)
  })

  it('l EXPIRATION tardive ferme aussi — 16,5 s après le dernier échange', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(3, 265, 11)],
      }),
    )
    expect(s[0].t1).toBe(265)
  })

  it('une émission RÉCUPÉRÉE ferme exactement comme une stricte', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [{ ...spent(3, 140, 11), recovered: true }],
      }),
    )
    expect(s[0].t1).toBe(140)
  })

  it('la MORT DU PORTEUR clôt sans `spent` — et la faille ne dure JAMAIS jusqu à la fin du rejeu', () => {
    const s = riftStations(
      doc({ translocations: [saut(3, 100, [12, 5], [40, 5])], tracks: [vie(3, 0, 180)] }),
    )
    expect(s[0].t1).toBe(180)
    expect(riftStationsAt(s, 181)).toEqual([])
  })

  it('un `spent` d un AUTRE équipement, ou d une autre vie, ne ferme rien', () => {
    const autreRang = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(3, 120, 4)],
        tracks: [vie(3, 0, 300)],
      }),
    )
    expect(autreRang[0].t1).toBe(300)
    const autreSlot = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(9, 120, 11)],
        tracks: [vie(3, 0, 300), vie(9, 0, 300)],
      }),
    )
    expect(autreSlot[0].t1).toBe(300)
  })

  it('un `spent` ENTRE deux échanges ferme la station qu il suit, PAS la suivante (K1-a)', () => {
    // Le joueur épuise son translocateur (t=150), en ramasse un autre, s'en ressert (t=200).
    // La station ouverte à t=100 doit s'éteindre à 150 — la borner sur le seul échange suivant
    // la laissait vivre jusqu'à 199, soit 5 s APRÈS la fin mesurée de l'équipement.
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [1, 1], [40, 5]), saut(3, 200, [40, 5], [1, 1])],
        equipmentChanges: [spent(3, 150, 11)],
        tracks: [vie(3, 0, 300)],
      }),
    )
    expect(s[0].t1).toBe(150)
    expect(riftStationsAt(s, 151)).toEqual([])
    // Le `spent` est ANTÉRIEUR au second échange : il a consommé un équipement que la seconde
    // faille ne connaît pas, et ne la ferme donc pas.
    expect(s[1].t1).toBe(300)
  })
})

describe('riftStations — le SLOT est réattribué : une faille ne survit pas à son porteur (K1-b, K2)', () => {
  /** Deux VIES sur le MÊME slot — l'invariant du dossier, et le seul cas qui teste `lifeEnd`. */
  const deuxVies = () => [vie(3, 0, 180), vie(3, 200, 400)]

  it('la station s arrête à la mort de SA vie, pas à celle de la dernière piste du slot', () => {
    const s = riftStations(doc({ translocations: [saut(3, 100, [12, 5], [40, 5])], tracks: deuxVies() }))
    // 180 = fin de la vie qui COUVRE l'échange. « La dernière piste de ce slot » rendrait 400 :
    // la faille traverserait la mort de son porteur et brillerait pendant la vie du suivant.
    expect(s[0].t1).toBe(180)
    expect(riftStationsAt(s, 181)).toEqual([])
    expect(riftStationsAt(s, 300)).toEqual([])
  })

  it('un échange dans CHAQUE vie donne deux failles, chacune bornée par la sienne', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [1, 1], [40, 5]), saut(3, 300, [70, 2], [10, 9])],
        tracks: deuxVies(),
      }),
    )
    expect(s.map((x) => x.t1)).toEqual([180, 400])
    // Entre les deux vies, plus personne ne porte de translocateur : la carte est muette.
    expect(riftStationsAt(s, 220)).toEqual([])
  })

  it('un `spent` de la vie SUIVANTE ne ferme pas la faille de la précédente', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(3, 250, 11)],
        tracks: deuxVies(),
      }),
    )
    expect(s[0].t1).toBe(180)
  })
})

describe('riftStations — P2.4 : un `from` sous saut de compteur ne ferme pas la faille', () => {
  it('la station retombe sur la MORT DU PORTEUR plutôt que sur une fermeture affirmée à tort', () => {
    const s = riftStations(
      doc({
        translocations: [saut(3, 100, [12, 5], [40, 5])],
        equipmentChanges: [spent(3, 120, 11, 1)],
        tracks: [vie(3, 0, 300)],
      }),
    )
    expect(s[0].t1).toBe(300)
  })
})
