/**
 * Tests — carrierPosition (« le glyphe d'un porteur EMBARQUÉ se dessine sur le véhicule »,
 * décision produit du 2026-09-05).
 *
 * CE QU'ILS PROTÈGENT, ET COMMENT ILS SONT CONSTRUITS. Le défaut corrigé n'est pas « pas de
 * position » mais « une position PLAUSIBLE ET FAUSSE » : le bipède embarqué ne réplique plus, et
 * l'interpolation linéaire de `positionAt` traverse le décor en ligne droite. Un test qui se
 * contenterait d'affirmer « une position sort » passerait AVANT comme APRÈS le correctif. Chaque
 * cas ci-dessous oppose donc les DEUX positions du même instant : la fixture est bâtie pour
 * qu'elles soient franchement distinctes (le bipède file en X, le véhicule monte en Y), et
 * l'assertion nomme celle qui doit gagner ET celle qui doit perdre.
 *
 * LE REPLI EST TESTÉ AUTANT QUE LA RÈGLE : jamais embarqué, épisode anonyme, épisode posé sur du
 * décor, monture sans aucune position, et document ANTÉRIEUR AUX VÉHICULES (schéma <= 38) — dans
 * ces cinq cas la position doit être EXACTEMENT celle d'avant ce lot, sans quoi le correctif
 * serait une régression sur tout l'historique déjà cuit.
 */
import { describe, expect, it } from 'vitest'

import { buildCarrierPosAt, buildEmbarkedPosAt, positionOfCarrierAt } from './carrierPosition'
import { buildPlayerPosAt } from './livesPosition'
import { testReplayDoc } from './test/testDoc'

/**
 * Le joueur A : deux échantillons seulement, aux deux bouts — c'est EXACTEMENT la forme d'une
 * trajectoire trouée par un embarquement, et `positionAt` y interpole en ligne droite le long
 * de l'axe X (frame 50 -> x = 50).
 */
const TRACKS = [
  {
    slot: 1,
    team: 0,
    xuid: 'A',
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 100, y: 0 },
    ],
    startFrame: 0,
    endFrame: 100,
  },
  {
    slot: 2,
    team: 0,
    xuid: 'B',
    points: [
      { t: 0, x: 5, y: 5 },
      { t: 100, x: 5, y: 5 },
    ],
    startFrame: 0,
    endFrame: 100,
  },
]

/** Le véhicule, lui, MONTE en Y : aucune image ne peut confondre les deux trajectoires. */
const SAMPLES = [
  { t: 0, x: 0, y: 0 },
  { t: 100, x: 0, y: 100 },
]

function vehicle(over: Record<string, unknown> = {}) {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 100,
    t1max: 100,
    end: 'unknown',
    family: 'warthog',
    samples: SAMPLES,
    rides: [{ t0: 10, t1: 80, slot: 1, seat: 0, src: 'event', xuid: 'A' }],
    ...over,
  }
}

/** Un document avec véhicules ; sans argument, le cas nominal (A conduit de 10 à 80). */
function docWithVehicles(vehicles: unknown[] = [vehicle()]) {
  return testReplayDoc({ tracks: TRACKS as never, vehicles: vehicles as never })
}

/** Le document d'AVANT les véhicules (schéma <= 38) : la clé n'existe pas au transport. */
function docSansVehicules() {
  return testReplayDoc({ tracks: TRACKS as never })
}

describe('positionOfCarrierAt — la règle, isolée', () => {
  it('embarqué : la position du véhicule GAGNE sur celle du bipède', () => {
    const embarked = () => ({ x: 1, y: 1 })
    const biped = () => ({ x: 9, y: 9 })
    expect(positionOfCarrierAt(embarked, biped, 'A', 50)).toEqual({ x: 1, y: 1 })
  })

  it('non embarqué : le bipède répond, et son « rien » reste un rien', () => {
    const biped = () => ({ x: 9, y: 9 })
    expect(positionOfCarrierAt(() => null, biped, 'A', 50)).toEqual({ x: 9, y: 9 })
    expect(positionOfCarrierAt(() => null, () => null, 'A', 50)).toBeNull()
  })
})

describe('buildCarrierPosAt — porteur embarqué', () => {
  it("à une image DE L'ÉPISODE, rend le VÉHICULE et non l'interpolation du bipède", () => {
    const doc = docWithVehicles()
    const bipede = buildPlayerPosAt(doc)('A', 50)
    const porteur = buildCarrierPosAt(doc)('A', 50)
    // La preuve tient dans l'écart : l'ancien chemin traversait le décor en (50, 0).
    expect(bipede).toEqual({ x: 50, y: 0 })
    expect(porteur).toEqual({ x: 0, y: 50 })
    expect(porteur).not.toEqual(bipede)
  })

  it('la MÊME interpolation que le sprite du véhicule, à toutes les images de l’épisode', () => {
    const posOf = buildCarrierPosAt(docWithVehicles())
    expect(posOf('A', 10)).toEqual({ x: 0, y: 10 })
    expect(posOf('A', 80)).toEqual({ x: 0, y: 80 })
  })

  it('À LA DESCENTE, le glyphe revient EXACTEMENT sur le bipède', () => {
    const doc = docWithVehicles()
    const bipede = buildPlayerPosAt(doc)
    const porteur = buildCarrierPosAt(doc)
    // 80 = dernière image de l'épisode (borne incluse, même convention que le prédicat du pion).
    expect(porteur('A', 80)).not.toEqual(bipede('A', 80))
    // 81 = première image à pied : plus aucune trace du véhicule.
    expect(porteur('A', 81)).toEqual(bipede('A', 81))
    expect(porteur('A', 81)).toEqual({ x: 81, y: 0 })
  })

  it("AVANT l'embarquement, le bipède répond déjà", () => {
    const doc = docWithVehicles()
    expect(buildCarrierPosAt(doc)('A', 5)).toEqual(buildPlayerPosAt(doc)('A', 5))
  })

  it('un joueur qui CHANGE DE MONTURE suit chaque véhicule à son tour', () => {
    const second = vehicle({
      slot: 701,
      samples: [
        { t: 0, x: 100, y: 0 },
        { t: 100, x: 100, y: 100 },
      ],
      rides: [{ t0: 85, t1: 95, slot: 1, seat: 0, src: 'event', xuid: 'A' }],
    })
    const posOf = buildCarrierPosAt(docWithVehicles([vehicle(), second]))
    expect(posOf('A', 50)).toEqual({ x: 0, y: 50 })
    expect(posOf('A', 90)).toEqual({ x: 100, y: 90 })
  })
})

describe('buildCarrierPosAt — les replis, tous identiques au comportement d’avant ce lot', () => {
  it('un joueur JAMAIS embarqué est inchangé, image par image', () => {
    const doc = docWithVehicles()
    const bipede = buildPlayerPosAt(doc)
    const porteur = buildCarrierPosAt(doc)
    for (const frame of [0, 10, 50, 80, 100]) {
      expect(porteur('B', frame)).toEqual(bipede('B', frame))
    }
  })

  it('document ANTÉRIEUR AUX VÉHICULES (schéma <= 38) : aucune différence, aucune image', () => {
    const doc = docSansVehicules()
    const bipede = buildPlayerPosAt(doc)
    const porteur = buildCarrierPosAt(doc)
    for (const frame of [0, 10, 50, 80, 100]) {
      expect(porteur('A', frame)).toEqual(bipede('A', frame))
    }
  })

  it('un épisode SANS XUID ne déplace aucun glyphe (aucun second pont d’identité)', () => {
    const anonyme = vehicle({ rides: [{ t0: 10, t1: 80, slot: 1, seat: 0, src: 'event' }] })
    const doc = docWithVehicles([anonyme])
    expect(buildCarrierPosAt(doc)('A', 50)).toEqual(buildPlayerPosAt(doc)('A', 50))
  })

  it('un épisode posé sur du DÉCOR n’embarque personne (bug du prop Falcon)', () => {
    const doc = docWithVehicles([vehicle({ family: 'falcon' })])
    expect(buildCarrierPosAt(doc)('A', 50)).toEqual(buildPlayerPosAt(doc)('A', 50))
  })

  it('une monture SANS aucune position à cette image rend la main au bipède', () => {
    const doc = docWithVehicles([vehicle({ samples: [], spawn: undefined })])
    expect(buildCarrierPosAt(doc)('A', 50)).toEqual({ x: 50, y: 0 })
  })

  it('un joueur INCONNU du document reste sans position', () => {
    expect(buildCarrierPosAt(docWithVehicles())('Z', 50)).toBeNull()
  })
})

describe('buildEmbarkedPosAt — l’index seul', () => {
  it('rend null hors épisode, et la position du véhicule dedans', () => {
    const doc = docWithVehicles()
    const embarked = buildEmbarkedPosAt(doc.vehicles)
    expect(embarked('A', 9)).toBeNull()
    expect(embarked('A', 10)).toEqual({ x: 0, y: 10 })
    expect(embarked('A', 81)).toBeNull()
  })

  it('un document sans le moindre épisode nommé ne coûte aucun index', () => {
    const embarked = buildEmbarkedPosAt(docSansVehicules().vehicles)
    expect(embarked('A', 50)).toBeNull()
  })
})
