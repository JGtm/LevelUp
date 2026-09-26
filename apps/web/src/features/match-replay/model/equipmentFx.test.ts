/**
 * equipmentFx.test.ts — la lecture des épisodes d'équipement actif, aux bornes près.
 *
 * LA RÈGLE VERROUILLÉE : l'effet dure EXACTEMENT l'épisode mesuré — bornes t0 et t1
 * INCLUSES, rien avant, rien après, et jamais l'effet d'une famille voisine ni d'un
 * autre slot.
 */
import { describe, expect, it } from 'vitest'

import { activeEquipmentAt } from './equipmentFx'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

function docWith(episodes: ReplayDocumentReady['equipmentEpisodes']): ReplayDocumentReady {
  return { equipmentEpisodes: episodes } as ReplayDocumentReady
}

describe('activeEquipmentAt', () => {
  const doc = docWith([
    { slot: 512, fam: 'camo', t0: 10, t1: 20, endRead: true },
    { slot: 512, fam: 'overshield', t0: 18, t1: 30 },
    { slot: 700, fam: 'camo', t0: 5, t1: 8 },
  ])

  it("l'effet dure exactement l'épisode : bornes incluses, rien autour", () => {
    expect(activeEquipmentAt(doc, 512, 9).camo).toBe(false)
    expect(activeEquipmentAt(doc, 512, 10).camo).toBe(true)
    expect(activeEquipmentAt(doc, 512, 20).camo).toBe(true)
    expect(activeEquipmentAt(doc, 512, 21).camo).toBe(false)
  })

  it('les deux familles se lisent indépendamment, y compris en recouvrement', () => {
    const at19 = activeEquipmentAt(doc, 512, 19)
    expect(at19).toEqual({ camo: true, overshield: true })
    const at25 = activeEquipmentAt(doc, 512, 25)
    expect(at25).toEqual({ camo: false, overshield: true })
  })

  it("un slot ne lit jamais les épisodes d'un autre", () => {
    expect(activeEquipmentAt(doc, 700, 19)).toEqual({ camo: false, overshield: false })
    expect(activeEquipmentAt(doc, 700, 6)).toEqual({ camo: true, overshield: false })
  })

  it('une famille inconnue du contrat ne déclenche RIEN — jamais l’effet d’une voisine', () => {
    const exotic = docWith([{ slot: 512, fam: 'grapple', t0: 0, t1: 100 }])
    expect(activeEquipmentAt(exotic, 512, 50)).toEqual({ camo: false, overshield: false })
  })

  it('sans épisode, rien : le document normalisé garantit un tableau, jamais null', () => {
    expect(activeEquipmentAt(docWith([]), 512, 10)).toEqual({ camo: false, overshield: false })
  })
})
