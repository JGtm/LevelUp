/**
 * Tests — hillSequence : la suite des collines jouées, lue dans `zoneStates[].spans`, et la prochaine seulement quand
 * la suite jouée montre déjà un cycle complet répété.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { hillActivationsOf, hillSequenceAt, type HillActivation } from './hillSequence'

interface Span {
  t0: number
  t1: number
  active?: boolean
}

/** Une zone avec ses rangs de lettre ; `active` vrai par défaut (une colline), faux pour une zone de base. */
function zone(zoneRef: number, letterRank: number | undefined, spans: Span[], active = true) {
  return {
    zoneRef,
    letterRank,
    spans: spans.map((s) => ({ owner: null, active, ...s })),
    gauge: [],
    gaugeRamps: [],
  }
}

function docWith(zoneStates: unknown[]) {
  return testReplayDoc({ frameIntervalMs: 100, zoneStates } as never)
}

/** Trois collines qui tournent A, B, C, A, B, C, de 100 images chacune, une par intervalle (zone par zone). */
function cycleDeTroisCollines() {
  const lettre = (i: number) => i % 3
  const parZone: Span[][] = [[], [], []]
  for (let i = 0; i < 6; i++) parZone[lettre(i)].push({ t0: i * 100, t1: i * 100 + 99 })
  return docWith(parZone.map((spans, ref) => zone(ref, ref, spans)))
}

const labels = (a: readonly HillActivation[]) => a.map((x) => x.label)

describe('hillActivationsOf', () => {
  it('rend les activations de toutes les collines dans l’ordre du temps, avec la lettre du calque', () => {
    expect(labels(hillActivationsOf(cycleDeTroisCollines()))).toEqual(['A', 'B', 'C', 'A', 'B', 'C'])
  })

  it('fusionne les intervalles qui se suivent d’une même colline en UNE activation, et sépare un trou', () => {
    const doc = docWith([
      zone(0, 0, [
        { t0: 0, t1: 49 },
        { t0: 50, t1: 99 },
        { t0: 300, t1: 349 },
      ]),
    ])
    const a = hillActivationsOf(doc)
    expect(a.map((x) => [x.t0, x.t1])).toEqual([[0, 99], [300, 349]])
  })

  it('ignore une zone dont aucun intervalle n’est actif (mode à zones, pas à collines)', () => {
    const doc = docWith([zone(0, 0, [{ t0: 0, t1: 500 }], false)])
    expect(hillActivationsOf(doc)).toEqual([])
  })

  it('sans lettre publiée, la colline se nomme par son rang parmi les collines du match, jamais une lettre inventée', () => {
    const doc = docWith([zone(7, undefined, [{ t0: 0, t1: 9 }]), zone(4, undefined, [{ t0: 20, t1: 29 }])])
    expect(labels(hillActivationsOf(doc))).toEqual(['2', '1'])
  })
})

describe('hillSequenceAt', () => {
  const doc = cycleDeTroisCollines()
  const all = hillActivationsOf(doc)

  it('ne montre que les collines commencées à l’image lue, la dernière étant la courante', () => {
    const r = hillSequenceAt(all, 120)
    expect(labels(r?.played ?? [])).toEqual(['A', 'B'])
    expect(r?.currentIndex).toBe(1)
  })

  it('entre deux collines (aucune active à l’image), aucune n’est la courante', () => {
    const espace = hillActivationsOf(docWith([zone(0, 0, [{ t0: 0, t1: 49 }, { t0: 200, t1: 249 }])]))
    const r = hillSequenceAt(espace, 120)
    expect(labels(r?.played ?? [])).toEqual(['A'])
    expect(r?.currentIndex).toBe(-1)
  })

  it('avant la première colline : rien', () => {
    expect(hillSequenceAt(hillActivationsOf(docWith([zone(0, 0, [{ t0: 50, t1: 99 }])])), 10)).toBeNull()
  })

  it('la prochaine n’est pas connue tant que la suite jouée ne montre pas un cycle complet répété', () => {
    expect(hillSequenceAt(all, 150)?.next).toBeNull() // A, B
    expect(hillSequenceAt(all, 250)?.next).toBeNull() // A, B, C : un cycle, pas répété
    expect(hillSequenceAt(all, 350)?.next).toBeNull() // A, B, C, A : répété à moitié
    expect(hillSequenceAt(all, 450)?.next).toBeNull() // A, B, C, A, B : dernier cycle incomplet
  })

  it('la prochaine est celle du cycle une fois le cycle répété en entier, tant qu’une autre colline suit au document', () => {
    const plusLong = [...all, ...all.map((a) => ({ ...a, t0: a.t0 + 600, t1: a.t1 + 600 }))]
    const r = hillSequenceAt(plusLong, 550) // A B C A B C joués, une suite existe
    expect(labels(r?.played ?? [])).toEqual(['A', 'B', 'C', 'A', 'B', 'C'])
    expect(r?.next).toBe('A')
  })

  it('sur la dernière colline du document, il n’y a pas de prochaine, même si le cycle la prédirait', () => {
    expect(hillSequenceAt(all, 550)?.next).toBeNull()
  })

  it('une colline qui revient seule (A, A) donne A pour prochaine tant qu’une autre activation suit', () => {
    const suite = hillActivationsOf(docWith([zone(0, 0, [{ t0: 0, t1: 9 }, { t0: 20, t1: 29 }, { t0: 40, t1: 49 }])]))
    expect(hillSequenceAt(suite, 25)?.next).toBe('A')
    expect(hillSequenceAt(suite, 45)?.next).toBeNull()
  })

  it('sans activation, aucune lecture', () => {
    expect(hillSequenceAt([], 100)).toBeNull()
  })
})
