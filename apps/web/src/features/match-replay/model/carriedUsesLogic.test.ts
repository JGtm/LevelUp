/**
 * Tests — carriedUsesLogic. Relevés réels figés de `0d9a9af9` (Forest), cuisson au schéma 95 du
 * 2026-10-14 : Joueur900b (slot 513) prend un répulseur (rang 6) à l'image 638 avec trois usages, les
 * consomme aux images 714, 811 et 921 ; l'objet disparaît à 930. Il portait avant un mur (rang 2).
 */
import { describe, expect, it } from 'vitest'

import { RESPAWN_WITNESSES } from '../test/spawnedObjectWitnesses'
import { testReplayDoc } from '../test/testDoc'
import { carriedUsesAt } from './carriedUsesLogic'

const PORTES_0D9A9AF9 = [
  { slot: 513, r: 2, t0: 191, t1: 248, energy: [{ t: 191, e: 0.66, n: 2 }] },
  {
    slot: 513,
    r: 6,
    t0: 638,
    t1: 930,
    energy: [
      { t: 638, e: 0.6, n: 3 },
      { t: 714, e: 0.4, n: 2 },
      { t: 811, e: 0.2, n: 1 },
      { t: 921, e: 0, n: 0 },
    ],
    uses: [714, 811, 921],
  },
]

describe('carriedUsesAt — les usages restants de l équipement porté', () => {
  const doc = testReplayDoc({ frameCount: 3000, carriedEquipment: PORTES_0D9A9AF9 })
  it('suit la lecture la plus récente de l objet porté', () => {
    expect(carriedUsesAt(doc, 513, 700, 6)).toEqual({ kind: 'count', charges: 3, age: 62 })
    expect(carriedUsesAt(doc, 513, 714, 6)).toEqual({ kind: 'count', charges: 2, age: 0 })
    expect(carriedUsesAt(doc, 513, 925, 6)).toEqual({ kind: 'count', charges: 0, age: 4 })
  })
  it('rien hors du port, ni pour un autre rang ou un autre porteur', () => {
    expect(carriedUsesAt(doc, 513, 930, 6)).toBeNull()
    expect(carriedUsesAt(doc, 513, 700, 2)).toBeNull()
    expect(carriedUsesAt(doc, 519, 700, 6)).toBeNull()
  })
})

describe('carriedUsesAt - un objet de reapparition (releve reel bc4bde17, slot 518, cree avant la frame 0)', () => {
  const doc = RESPAWN_WITNESSES.sensor()
  it('compte les usages restants du capteur de depart, de la prise a la frame 0 jusqu a ses quatre usages', () => {
    expect(carriedUsesAt(doc, 518, 0, 22)).toEqual({ kind: 'count', charges: 4, age: 0 })
    expect(carriedUsesAt(doc, 518, 100, 22)).toEqual({ kind: 'count', charges: 3, age: 24 })
    expect(carriedUsesAt(doc, 518, 600, 22)).toEqual({ kind: 'count', charges: 1, age: 48 })
  })
  it('joint par la famille quand le rang affiche n est pas celui de l objet, jamais par un autre porteur', () => {
    // Le rang 99 n a aucune famille dans le document : rien ne se joint.
    expect(carriedUsesAt(doc, 518, 100, 99)).toBeNull()
    expect(carriedUsesAt(doc, 519, 100, 22)).toBeNull()
    // Un objet dont le rang est inconnu (-1) se joint a la capacite de sa famille.
    const sansRang = { ...doc, carriedEquipment: doc.carriedEquipment.map((c) => ({ ...c, r: -1 })) }
    expect(carriedUsesAt(sansRang, 518, 100, 22)).toEqual({ kind: 'count', charges: 3, age: 24 })
  })
})
