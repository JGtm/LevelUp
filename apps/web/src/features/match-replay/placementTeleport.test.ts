/**
 * Les PASSAGES PAR UNE FAILLE — ce que la règle retient, et surtout ce qu'elle écarte.
 *
 * Les deux verrous ont chacun leur contre-exemple mesuré dans le corpus, et ils sont ici :
 * un porteur du translocateur qui franchit une PORTE de la carte (verrou A pris seul en
 * retient 6 au lieu de 4), et un saut instantané fait par quelqu'un qui n'a pas l'équipement
 * (verrou B pris seul laisserait passer tout ce que le canal `abilities` sait exclure).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { lastTeleportAge, riftTeleports, type RiftTeleport } from './placementTeleport'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

type Abilities = ReplayDocumentReady['abilities']

/** Une lecture de capacité. Le rang 11 est le translocateur quantique (cf. abilityLabels). */
function porte(slot: number, r = 11, t = 0): Abilities[number] {
  return { slot, r, t, src: 'i48' }
}

/** Les slots des cas ci-dessous portent tous le translocateur, sauf mention contraire. */
const PORTEURS: Abilities = [1, 2, 3, 4, 9].map((s) => porte(s))

/** Une vie qui saute de `from` à `to` entre deux frames consécutives. */
function vieQuiSaute(
  slot: number,
  from: { x: number; y: number },
  to: { x: number; y: number },
  frame = 10,
): ReplayTrackReady {
  return {
    slot,
    team: 0,
    startFrame: 0,
    endFrame: frame + 10,
    points: [
      { t: frame - 1, x: from.x, y: from.y },
      { t: frame, x: to.x, y: to.y },
      { t: frame + 1, x: to.x + 0.4, y: to.y },
    ],
  }
}

function faille(over: Partial<ReplayEquipmentPlacement> = {}): ReplayEquipmentPlacement {
  return { family: 'translocator_beacon', id: 'f1', owner: 3, t0: 0, t1: 100, x: 40, y: 0, ...over }
}

describe('riftTeleports — verrou A : le porteur', () => {
  it('retient un déplacement instantané fait par qui TIENT le translocateur', () => {
    const t = riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 20, y: 0 })], PORTEURS)
    expect(t).toHaveLength(1)
    expect(t[0]).toMatchObject({ slot: 3, frame: 10, from: { x: 0, y: 0 }, to: { x: 20, y: 0 } })
  })

  it('écarte le même saut fait par qui tient AUTRE CHOSE', () => {
    const grappin = [porte(3, 4)]
    expect(riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 20, y: 0 })], grappin)).toEqual([])
  })

  it('écarte le saut quand aucune lecture de capacité n existe pour ce slot', () => {
    expect(riftTeleports([], [vieQuiSaute(7, { x: 0, y: 0 }, { x: 20, y: 0 })], PORTEURS)).toEqual([])
  })

  it('une lecture À VENIR ne prouve rien : porter plus tard n est pas porter au saut', () => {
    const plusTard: Abilities = [porte(3, 11, 40)]
    expect(riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 20, y: 0 }, 10)], plusTard)).toEqual([])
  })

  it('la lecture retenue est la DERNIÈRE avant le saut, pas la première du film', () => {
    // Il l a eu, il l a lâché : au saut il tient le grappin.
    const perdu: Abilities = [porte(3, 11, 0), porte(3, 4, 5)]
    expect(riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 20, y: 0 }, 10)], perdu)).toEqual([])
  })
})

describe('riftTeleports — verrou B : la porte de la carte', () => {
  it('ÉCARTE un vecteur partagé par plusieurs joueurs, dans les deux sens', () => {
    const passages = [
      vieQuiSaute(1, { x: 0, y: 0 }, { x: 38, y: 12 }, 10),
      vieQuiSaute(2, { x: 5, y: 5 }, { x: 43, y: 17 }, 20),
      // le retour : vecteur opposé, donc la MÊME porte.
      vieQuiSaute(3, { x: 43, y: 17 }, { x: 5, y: 5 }, 30),
    ]
    expect(riftTeleports([], passages, PORTEURS)).toEqual([])
  })

  it('c est le cas mesuré du slot 742 : un porteur qui franchit une porte reste écarté', () => {
    const passages = [
      vieQuiSaute(1, { x: 0, y: 0 }, { x: 46, y: -5 }, 10),
      vieQuiSaute(2, { x: 0, y: 0 }, { x: 46, y: -5 }, 20),
    ]
    // Les deux portent le translocateur ; seul le verrou B les sépare du vrai passage.
    expect(riftTeleports([], passages, PORTEURS)).toEqual([])
  })

  it('un même joueur qui repasse au même endroit ne suffit PAS à faire une porte', () => {
    const seul: ReplayTrackReady = {
      slot: 4, team: 0, startFrame: 0, endFrame: 60,
      points: [
        { t: 10, x: 0, y: 0 }, { t: 11, x: 38, y: 12 },
        { t: 30, x: 0, y: 0 }, { t: 31, x: 38, y: 12 },
      ],
    }
    expect(riftTeleports([], [seul], PORTEURS)).toHaveLength(2)
  })

  it('la dispersion d une porte reste une porte — deux entrées ne sont jamais au centimètre', () => {
    const passages = [
      vieQuiSaute(1, { x: 0, y: 0 }, { x: 38.0, y: 12.0 }, 10),
      vieQuiSaute(2, { x: 0, y: 0 }, { x: 39.4, y: 13.1 }, 20),
    ]
    expect(riftTeleports([], passages, PORTEURS)).toEqual([])
  })
})

describe('riftTeleports — la vitesse, la faille, et l ordre', () => {
  it('ignore un déplacement que la course explique', () => {
    // 3 m en une frame de 100 ms : 30 m/s, au-dessus de la course mais sous le seuil.
    expect(riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 3, y: 0 })], PORTEURS)).toEqual([])
  })

  it('ignore un saut étalé sur une coupure de piste — un trou n est pas un passage', () => {
    const trou: ReplayTrackReady = {
      slot: 3, team: 0, startFrame: 0, endFrame: 60,
      points: [{ t: 10, x: 0, y: 0 }, { t: 50, x: 40, y: 0 }],
    }
    expect(riftTeleports([], [trou], PORTEURS)).toEqual([])
  })

  it('corrobore par la faille du MÊME joueur quand elle est active à l arrivée', () => {
    const saut = [vieQuiSaute(3, { x: 0, y: 0 }, { x: 38, y: 2 })]
    expect(riftTeleports([faille({ x: 40, y: 0 })], saut, PORTEURS)[0].viaRift).toBe(true)
    // faille d un AUTRE joueur, au même endroit : on ne se téléporte pas chez le voisin.
    expect(riftTeleports([faille({ owner: 9, x: 40, y: 0 })], saut, PORTEURS)[0].viaRift).toBe(false)
    // faille éteinte à cet instant, puis faille trop loin.
    expect(riftTeleports([faille({ x: 40, y: 0, t0: 50, t1: 90 })], saut, PORTEURS)[0].viaRift).toBe(false)
    expect(riftTeleports([faille({ x: 80, y: 0 })], saut, PORTEURS)[0].viaRift).toBe(false)
  })

  it('l absence de corroboration ne supprime PAS le passage', () => {
    // Tout le corpus local est dans ce cas : aucune pose déployée, donc aucune faille à
    // l arrivée. L exiger ramènerait la règle à zéro — ce qu elle valait avant.
    const t = riftTeleports([], [vieQuiSaute(3, { x: 0, y: 0 }, { x: 38, y: 2 })], PORTEURS)
    expect(t).toHaveLength(1)
    expect(t[0].viaRift).toBe(false)
  })

  it('rend les passages dans l ordre des frames d arrivée', () => {
    const passages = [
      vieQuiSaute(1, { x: 0, y: 0 }, { x: 20, y: 0 }, 50),
      vieQuiSaute(2, { x: 0, y: 100 }, { x: 0, y: 80 }, 20),
    ]
    expect(riftTeleports([], passages, PORTEURS).map((t) => t.frame)).toEqual([20, 50])
  })
})

describe('lastTeleportAge — l âge du passage le plus récent d un slot', () => {
  const passage = (slot: number, frame: number): RiftTeleport => ({
    slot,
    frame,
    from: { x: 0, y: 0 },
    to: { x: 20, y: 0 },
    viaRift: false,
  })

  it('aucun passage advenu, ou passage d un autre slot : -1', () => {
    expect(lastTeleportAge([], 3, 50)).toBe(-1)
    expect(lastTeleportAge([passage(7, 10)], 3, 50)).toBe(-1)
  })

  it('un passage À VENIR ne compte pas — l éclat date un événement advenu', () => {
    expect(lastTeleportAge([passage(3, 60)], 3, 50)).toBe(-1)
  })

  it('deux passages advenus : l âge du plus récent', () => {
    expect(lastTeleportAge([passage(3, 10), passage(3, 40)], 3, 50)).toBe(10)
  })
})
