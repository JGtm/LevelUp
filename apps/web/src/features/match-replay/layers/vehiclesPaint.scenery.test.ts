/**
 * vehiclesPaint.scenery.test.ts — LES VÉHICULES DE DÉCOR D'UNE CARTE NE SE DESSINENT PAS, ET UN
 * VÉHICULE POSÉ DANS L'AIRE DE JEU SE DESSINE TOUJOURS.
 *
 * RETOURS DU REJEU (lot L1.3, décision Q13 « masqués », 2026-09-23 ; lot M7, 2026-09-24). Sur
 * Starboard (ab526724, f0220a96), six véhicules garés à 19-24 m au sud de l'arène — 1 Scorpion,
 * 2 Wasp, 3 Warthog — ; sur Goliath (d8b13ec2), un Wasp 3 m sous le sol. DEPUIS M7, LA RÈGLE VIT
 * CÔTÉ GO (`internal/service/replay_vehicle_scenery_rule.go`, prouvée sur les fonds de carte réels par
 * `service/replay_vehicle_scenery_test.go`) : posé par la carte ET hors de la zone jouable. Le
 * client LIT le verdict (`doc.vehicleScenery.hidden`, replié en `track.scenery`) : ces tests
 * prouvent le pli, le dessin et l'embarquement ; aucune condition n'est recalculée ici.
 *
 * LES FIXTURES SONT LES VIES RÉELLES (documents reconstruits au schéma 69), recopiées champ pour
 * champ ; les négatifs aussi : un Warthog garé mais simulé (0301037e 779), une tourelle bannie de
 * bfecd02b (élément de carte, qui doit rester dessiné) et un Mongoose de Behemoth (7b0d89c4 773)
 * ramené à une POSE SEULE — posé au départ dans l'aire de jeu, personne n'y touche : il se dessine.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleTrack } from '@/lib/api/types'

import { count, recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import type { PlacementView } from './placementShapes'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'
import {
  vehicleCanEmbark,
  vehicleIsHidden,
  vehicleIsScenery,
  VEHICLE_KIND_MAP_ELEMENT,
} from '../model/vehiclesLayer'

/** Une vie POSÉE telle que le document la publie : un échantillon à t0, fin de film, aucun occupant. */
function posee(slot: number, family: string, x: number, y: number, t1: number, t1max: number): ReplayVehicleTrackReady {
  return {
    slot,
    gen: 1,
    family,
    t0: 0,
    t1,
    t1max,
    end: 'film_end',
    samples: [{ t: 0, x, y }],
    rides: [],
    spawn: { x, y, z: 81 },
  }
}

/** La même vie, déclarée décor par le serveur (le pli de `normalizeReplayDocument`). */
function decor(slot: number, family: string, x: number, y: number, t1: number, t1max: number): ReplayVehicleTrackReady {
  return { ...posee(slot, family, x, y, t1, t1max), scenery: true }
}

/** ab526724 (CTF Starboard) : les six vies de décor, positions publiées au centimètre. */
const STARBOARD: ReplayVehicleTrackReady[] = [
  decor(771, 'scorpion', 8.96, -132.79, 6457, 6457),
  decor(772, 'wasp', -1.82, -130.09, 6457, 6457),
  decor(773, 'wasp', -1.82, -133.38, 6457, 6457),
  decor(774, 'warthog', 6.42, -129.74, 6457, 6457),
  decor(776, 'warthog', 1.58, -133.84, 6457, 6457),
  decor(778, 'warthog', 1.67, -132.04, 6457, 6457),
]

/** d8b13ec2 (Goliath) : le Wasp posé sous le sol de jeu. */
const GOLIATH_WASP = decor(768, 'wasp', -0.44, -8.16, 7304, 7388)

/** 7b0d89c4 773 (Behemoth) : un Mongoose POSÉ dans l'aire de jeu, jamais touché — aucun verdict. */
const BEHEMOTH_MONGOOSE = posee(773, 'mongoose', -101.61, 27.63, 5969, 6023)

/** 0301037e 779 : un Warthog garé JAMAIS occupé, mais simulé — 52 positions (3 gardées ici). */
const REFUGE_WARTHOG: ReplayVehicleTrackReady = {
  slot: 779,
  gen: 1,
  family: 'warthog',
  t0: 1196,
  t1: 4916,
  t1max: 4934,
  end: 'film_end',
  samples: [
    { t: 1424, x: 139.39, y: 76.59 },
    { t: 1500, x: 139.4, y: 76.6 },
    { t: 1600, x: 139.41, y: 76.6 },
  ],
  rides: [],
}

/** bfecd02b 768 : tourelle automatique bannie — AUCUN échantillon, un spawn (élément de carte). */
const TOURELLE: ReplayVehicleTrackReady = {
  slot: 768,
  gen: 1,
  family: 'tourelle_auto_bannie',
  t0: 0,
  t1: 5032,
  t1max: 5032,
  end: 'film_end',
  samples: [],
  rides: [],
  spawn: { x: -8.98, y: -90.14, z: 56.61 },
}

/** ab526724 770 (schéma 68) : une seule position, mais à t=3601 (pas à la naissance). */
const TARDIVE: ReplayVehicleTrackReady = {
  slot: 770,
  gen: 1,
  t0: 0,
  t1: 6457,
  t1max: 6457,
  end: 'film_end',
  samples: [{ t: 3601, x: -6.79, y: -200.55 }],
  rides: [],
}

const VIEW: PlacementView = {
  bounds: { minX: -20, minY: -140, maxX: 20, maxY: -60 },
  width: 400,
  height: 400,
  pad: 8,
}

const SPRITE = { width: 128, height: 128 } as unknown as CanvasImageSource

function style(): VehicleStyle {
  return {
    neutralInk: '#neutre',
    labelStroke: '#contour',
    showNames: true,
    showAim: true,
    spriteOf: () => SPRITE,
    sizeOf: () => ({ naturalWidthPx: 128, naturalHeightPx: 128, mmPerPx: 10 }),
    kindOf: (f) => (f === 'tourelle_auto_bannie' ? VEHICLE_KIND_MAP_ELEMENT : undefined),
    labelOfFamily: () => null,
    colorOfSlot: () => '#equipe',
    colorOfXuid: () => null,
    nameOfSlot: () => null,
    nameOfXuid: () => null,
    explosionInk: {
      tint: {
        kinetic: 'K', plasma_cool: 'PC', plasma_hot: 'PH', forerunner: 'F', electric: 'E', needle: 'N',
        blast: 'B', neutral: 'X',
      },
      core: 'C',
    },
    reducedMotion: false,
    offscreenLabelOf: (name) => name,
    offscreenGroupLabelOf: (n) => String(n),
  }
}

function paint(tracks: ReplayVehicleTrackReady[], frame = 100) {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, VIEW, { frame, k: 1, frameMs: 100 }, style())
  return ops
}

/** Une vie au transport (avant normalisation) : sans le pli `scenery`, tel que le serveur la sert. */
function auTransport(t: ReplayVehicleTrackReady): ReplayVehicleTrack {
  const reste: Partial<ReplayVehicleTrackReady> = { ...t }
  delete reste.scenery
  return reste as ReplayVehicleTrack
}

describe('vehicleIsScenery — le verdict du serveur, replié sur la vie', () => {
  it('reconnaît les six véhicules de Starboard et le Wasp de Goliath déclarés décor', () => {
    for (const t of [...STARBOARD, GOLIATH_WASP]) expect(vehicleIsScenery(t), `slot ${t.slot}`).toBe(true)
  })

  it('M7 : un Mongoose posé dans l’aire de jeu de Behemoth, jamais touché, reste dessiné', () => {
    expect(vehicleIsScenery(BEHEMOTH_MONGOOSE)).toBe(false)
    expect(vehicleIsHidden(BEHEMOTH_MONGOOSE)).toBe(false)
    expect(count(paint([BEHEMOTH_MONGOOSE]), 'drawImage')).toBe(1)
  })

  it('ne touche ni un véhicule garé simulé, ni une tourelle, ni une position tardive', () => {
    expect(vehicleIsScenery(REFUGE_WARTHOG)).toBe(false)
    expect(vehicleIsScenery(TOURELLE)).toBe(false)
    expect(vehicleIsScenery(TARDIVE)).toBe(false)
  })

  it('vehicleIsHidden réunit les familles non jouables et le décor de carte', () => {
    expect(vehicleIsHidden(STARBOARD[0])).toBe(true)
    expect(vehicleIsHidden({ ...REFUGE_WARTHOG, family: 'falcon' })).toBe(true)
    expect(vehicleIsHidden(REFUGE_WARTHOG)).toBe(false)
    expect(vehicleIsHidden(TOURELLE)).toBe(false)
  })
})

describe('normalizeReplayDocument — `vehicleScenery.hidden` se replie sur la vie', () => {
  it('seules les vies nommées par le serveur deviennent décor, à (slot, gen) près', () => {
    const doc = testReplayDoc({
      vehicles: [...STARBOARD.slice(0, 2), BEHEMOTH_MONGOOSE].map(auTransport),
      vehicleScenery: {
        zone: 'map',
        floor: 'played',
        candidates: 3,
        inPlayArea: 1,
        zoneUnknown: 0,
        hidden: [
          { slot: 771, gen: 1, reason: 'off_play_area' },
          { slot: 772, gen: 1, reason: 'off_play_area' },
          { slot: 773, gen: 2, reason: 'off_play_area' },
        ],
      },
    })
    expect(doc.vehicles.map((v) => vehicleIsScenery(v))).toEqual([true, true, false])
  })

  it('verdict absent (carte sans zone connue, document sans ce calque) : rien n’est masqué', () => {
    const doc = testReplayDoc({ vehicles: STARBOARD.map(auTransport) })
    expect(doc.vehicles.some((v) => vehicleIsHidden(v))).toBe(false)
  })
})

describe('drawVehiclesLayer — le décor de carte est MASQUÉ (décision Q13 du 2026-09-23)', () => {
  it('ab526724 771-778 : aucun sprite, aucun losange', () => {
    const ops = paint(STARBOARD)
    expect(count(ops, 'drawImage')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
  })

  it('d8b13ec2 768 : le Wasp sous le sol ne se dessine pas', () => {
    const ops = paint([GOLIATH_WASP])
    expect(count(ops, 'drawImage')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
  })

  it('négatif : le Warthog garé de 0301037e reste dessiné', () => {
    expect(count(paint([REFUGE_WARTHOG], 1500), 'drawImage')).toBe(1)
  })

  it('négatif : la même vie de Starboard SANS verdict du serveur se dessine', () => {
    expect(count(paint([posee(771, 'scorpion', 8.96, -132.79, 6457, 6457)]), 'drawImage')).toBe(1)
  })

  it('négatif : la tourelle bannie de bfecd02b garde son pictogramme', () => {
    const ops = paint([TOURELLE])
    expect(count(ops, 'drawImage') + count(ops, 'fill') + count(ops, 'stroke')).toBeGreaterThan(0)
  })
})

/**
 * L'EMBARQUEMENT SE PROUVE PAR `vehicleCanEmbark`, ET PAR LUI SEUL. Une vie de décor n'a, PAR
 * DÉFINITION, aucun occupant (`rides` vide est une des conditions de pose, côté Go) : un test de
 * `buildEmbarkedPredicate` sur ces vies n'aurait rien à refuser et resterait vert avec ou sans le
 * refus (revue RR-L1-02, 2026-09-23 — ce test-là a été retiré). Le refus porte sur la porte commune
 * des deux lecteurs (`buildEmbarkedPredicate`, `carrierPosition`), et c'est elle qu'on verrouille.
 */
describe('le décor de carte n’embarque personne', () => {
  it('vehicleCanEmbark refuse une vie de décor, accepte le Warthog garé simulé et le Mongoose posé', () => {
    for (const t of [...STARBOARD, GOLIATH_WASP]) expect(vehicleCanEmbark(t), `slot ${t.slot}`).toBe(false)
    expect(vehicleCanEmbark(REFUGE_WARTHOG)).toBe(true)
    expect(vehicleCanEmbark(BEHEMOTH_MONGOOSE)).toBe(true)
  })
})

/**
 * LA VARIANTE NOMME LE SPRITE DESSINÉ (schéma 69, revue adverse du lot M4a, F6). Un Rockethog est
 * un Warthog dont la tourelle nomme la variante : sa famille (moteur, explosion) reste `warthog`,
 * mais c'est le sprite `rockethog` qui se dessine — taille et image lues sous ce nom.
 */
describe('drawVehiclesLayer — la variante nomme le sprite', () => {
  it('un Warthog de variante `rockethog` se dessine avec le sprite du Rockethog', () => {
    const lus: string[] = []
    const s: VehicleStyle = {
      ...style(),
      sizeOf: (f) => {
        lus.push(f)
        return { naturalWidthPx: 128, naturalHeightPx: 128, mmPerPx: 10 }
      },
    }
    const rockethog: ReplayVehicleTrackReady = { ...REFUGE_WARTHOG, variant: 'rockethog' }
    const { ops, ctx } = recordingContext()
    drawVehiclesLayer(ctx, [rockethog], VIEW, { frame: 1500, k: 1, frameMs: 100 }, s)
    expect(count(ops, 'drawImage')).toBe(1)
    expect(lus).toContain('rockethog')
    expect(lus).not.toContain('warthog')
  })
})
