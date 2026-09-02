/**
 * vehiclesLayer.test.ts — les QUATRE responsabilités pures du calque véhicules (lot C, gate C) :
 * orientation (mouvement / arrêt / pré-mouvement / jamais mobile), taille (manifeste factice),
 * prédicat embarqué (multi-passagers, sorties indépendantes), et occupation/teinte.
 *
 * AUCUN CANVAS ICI : ce fichier ne teste que la géométrie et la logique, exactement la même
 * distinction que `weaponPadTime.test.ts` face à `weaponPadsLayer.ts`. Le tracé canvas lui-même
 * (drawVehiclesLayer) est vérifié à l'œil au gate C6 (film réel) faute d'un contexte 2D sous
 * jsdom — même repli que documenté pour `tintedIconCanvas` avant ce lot.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleRide, ReplayVehicleSample } from '@/lib/api/types'

import type { ReplayVehicleTrackReady } from './replayNormalize'
import {
  buildEmbarkedPredicate,
  vehicleActiveRides,
  vehicleColorAt,
  vehicleHeadingAt,
  vehiclePositionAt,
  vehicleScreenAngle,
  vehicleScreenLengthPx,
  vehicleSpriteScale,
  vehicleVisibleAt,
  VEHICLE_DEFAULT_HEADING_DEG,
  VEHICLE_FLOOR_PX,
  VEHICLE_SOFT_CEIL_PX,
} from './vehiclesLayer'

/** Une vie de véhicule minimale, complétée par le test. */
function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'inconnue',
    family: 'warthog',
    samples: [],
    rides: [],
    ...over,
  }
}

function sample(over: Partial<ReplayVehicleSample>): ReplayVehicleSample {
  return { t: 0, x: 0, y: 0, ...over }
}

function ride(over: Partial<ReplayVehicleRide>): ReplayVehicleRide {
  return { t0: 0, t1: 100, slot: 1, src: 'event', ...over }
}

describe('vehicleHeadingAt — orientation (décision de cadrage)', () => {
  it('JAMAIS MOBILE (aucun échantillon) : cap par défaut, nez vers le haut de l’écran', () => {
    expect(vehicleHeadingAt(track({ samples: [] }), 500)).toBe(VEHICLE_DEFAULT_HEADING_DEG)
  })

  it('JAMAIS MOBILE (des échantillons existent, aucun ne porte de cap — tourelle immobile)', () => {
    const t = track({ samples: [sample({ t: 0, x: 1, y: 1 }), sample({ t: 500, x: 1, y: 1 })] })
    expect(vehicleHeadingAt(t, 500)).toBe(VEHICLE_DEFAULT_HEADING_DEG)
  })

  it('EN MOUVEMENT : l’échantillon couvrant l’image porte le cap de la vélocité', () => {
    const t = track({
      samples: [sample({ t: 0, x: 0, y: 0 }), sample({ t: 100, x: 5, y: 5, h: 45 })],
    })
    expect(vehicleHeadingAt(t, 100)).toBe(45)
  })

  it('À L’ARRÊT : le dernier cap connu est reporté (aucune interpolation vers un cap futur)', () => {
    const t = track({
      samples: [
        sample({ t: 0, x: 0, y: 0, h: 30 }),
        sample({ t: 10, x: 5, y: 5, h: 60 }),
        sample({ t: 20, x: 5, y: 5, h: 60 }), // arrêté : le serveur reporte le même cap
      ],
    })
    // Entre les deux derniers échantillons (immobile) : le cap CONNU au dernier point couvrant.
    expect(vehicleHeadingAt(t, 15)).toBe(60)
    // Après le dernier échantillon (véhicule au repos, plus aucun flux) : même cap maintenu.
    expect(vehicleHeadingAt(t, 999)).toBe(60)
  })

  it('AVANT LE PREMIER MOUVEMENT : le cap du PREMIER échantillon mobile à venir', () => {
    const t = track({
      samples: [
        sample({ t: 0, x: 0, y: 0 }), // à quai, pas encore de cap
        sample({ t: 50, x: 0, y: 0 }), // toujours à quai
        sample({ t: 100, x: 5, y: 5, h: 45 }), // premier mouvement
      ],
    })
    expect(vehicleHeadingAt(t, 0)).toBe(45)
    expect(vehicleHeadingAt(t, 75)).toBe(45)
  })
})

describe('vehicleScreenAngle — la constante d’écart d’écran (gate C6)', () => {
  it('90° monde (nord, +Y) -> 0 rad écran : AUCUNE rotation, le sprite nez-en-haut reste nez-en-haut', () => {
    expect(vehicleScreenAngle(90)).toBeCloseTo(0, 10)
  })

  it('0° monde (est, +X) -> +90° écran : le nez pivote vers la droite de l’écran', () => {
    expect(vehicleScreenAngle(0)).toBeCloseTo(Math.PI / 2, 10)
  })

  it('180° monde (ouest, -X) -> -90° écran : le nez pivote vers la gauche', () => {
    expect(vehicleScreenAngle(180)).toBeCloseTo(-Math.PI / 2, 10)
  })

  it('270° monde (sud, -Y) -> -180° écran : le nez pointe vers le bas', () => {
    expect(vehicleScreenAngle(270)).toBeCloseTo(-Math.PI, 10)
  })
})

describe('vehiclePositionAt / vehicleVisibleAt', () => {
  it('avant le premier échantillon : la position de NAISSANCE (spawn) répond', () => {
    const t = track({
      spawn: { x: 10, y: 20 },
      samples: [sample({ t: 100, x: 50, y: 50 })],
    })
    expect(vehiclePositionAt(t, 0)).toEqual({ x: 10, y: 20 })
  })

  it('sans spawn ni échantillon : aucune position, rien à dessiner', () => {
    expect(vehiclePositionAt(track({ spawn: undefined, samples: [] }), 500)).toBeNull()
  })

  it('après le dernier échantillon : la dernière position connue est maintenue', () => {
    const t = track({ samples: [sample({ t: 0, x: 1, y: 1 }), sample({ t: 100, x: 9, y: 9 })] })
    expect(vehiclePositionAt(t, 500)).toEqual({ x: 9, y: 9 })
  })

  it('la fenêtre [t0, t1max] est INCLUSIVE aux deux bornes, rien au-delà', () => {
    const t = track({ t0: 10, t1: 90, t1max: 100 })
    expect(vehicleVisibleAt(t, 9)).toBe(false)
    expect(vehicleVisibleAt(t, 10)).toBe(true)
    expect(vehicleVisibleAt(t, 100)).toBe(true)
    expect(vehicleVisibleAt(t, 101)).toBe(false)
  })
})

describe('vehicleActiveRides — tri conducteur puis sièges croissants (C7)', () => {
  it('trie siège 0 (conducteur) en premier, puis par siège croissant, sièges inconnus en dernier', () => {
    const t = track({
      rides: [
        ride({ slot: 3, seat: 2, t0: 0, t1: 100 }),
        ride({ slot: 4, seat: undefined, t0: 0, t1: 100 }),
        ride({ slot: 1, seat: 0, t0: 0, t1: 100 }),
        ride({ slot: 2, seat: 1, t0: 0, t1: 100 }),
      ],
    })
    expect(vehicleActiveRides(t, 50).map((r) => r.slot)).toEqual([1, 2, 3, 4])
  })

  it('ne rend que les épisodes qui COUVRENT l’image (bornes inclusives)', () => {
    const t = track({ rides: [ride({ slot: 1, t0: 10, t1: 20 })] })
    expect(vehicleActiveRides(t, 9)).toHaveLength(0)
    expect(vehicleActiveRides(t, 10)).toHaveLength(1)
    expect(vehicleActiveRides(t, 20)).toHaveLength(1)
    expect(vehicleActiveRides(t, 21)).toHaveLength(0)
  })
})

describe('vehicleColorAt — teinte du véhicule (C7)', () => {
  const colors = new Map<number, string>([
    [1, '#111'],
    [2, '#222'],
  ])
  const colorOfSlot = (slot: number) => colors.get(slot) ?? null

  it('couleur du CONDUCTEUR (siège 0) quand elle est résolue', () => {
    const t = track({ rides: [ride({ slot: 1, seat: 0 }), ride({ slot: 2, seat: 1 })] })
    expect(vehicleColorAt(t, 50, colorOfSlot)).toBe('#111')
  })

  it('à défaut (conducteur inconnu) : la couleur de N’IMPORTE QUEL occupant connu', () => {
    const t = track({ rides: [ride({ slot: 99, seat: 0 }), ride({ slot: 2, seat: 1 })] })
    expect(vehicleColorAt(t, 50, colorOfSlot)).toBe('#222')
  })

  it('aucun occupant, ou aucun résolu : null (neutre, l’appelant pose son encre de thème)', () => {
    expect(vehicleColorAt(track({ rides: [] }), 50, colorOfSlot)).toBeNull()
    const t = track({ rides: [ride({ slot: 98, seat: 0 }), ride({ slot: 99, seat: 1 })] })
    expect(vehicleColorAt(t, 50, colorOfSlot)).toBeNull()
  })
})

describe('buildEmbarkedPredicate — pion embarqué, MULTI-PASSAGERS (C7, rappel utilisateur)', () => {
  it('plusieurs occupants SIMULTANÉS du même véhicule sont TOUS embarqués pendant leur épisode', () => {
    const t = track({
      rides: [
        ride({ slot: 10, seat: 0, t0: 0, t1: 100 }), // conducteur
        ride({ slot: 11, seat: 1, t0: 10, t1: 90 }), // passager, fenêtre plus courte
      ],
    })
    const isEmbarkedAt = buildEmbarkedPredicate([t])
    expect(isEmbarkedAt(10, 50)).toBe(true)
    expect(isEmbarkedAt(11, 50)).toBe(true)
  })

  it('SORTIES INDÉPENDANTES : chaque occupant reprend son pion à SA propre sortie', () => {
    const t = track({
      rides: [
        ride({ slot: 10, seat: 0, t0: 0, t1: 100 }),
        ride({ slot: 11, seat: 1, t0: 10, t1: 90 }),
      ],
    })
    const isEmbarkedAt = buildEmbarkedPredicate([t])
    // Le passager (11) est sorti à 90 : il reprend son pion, le conducteur (10) reste embarqué.
    expect(isEmbarkedAt(11, 95)).toBe(false)
    expect(isEmbarkedAt(10, 95)).toBe(true)
    // Le conducteur sort à 100 à son tour, indépendamment.
    expect(isEmbarkedAt(10, 101)).toBe(false)
  })

  it('un slot jamais embarqué (aucune occupation, aucun véhicule) : toujours faux', () => {
    const isEmbarkedAt = buildEmbarkedPredicate([track({ rides: [] })])
    expect(isEmbarkedAt(42, 0)).toBe(false)
    expect(isEmbarkedAt(42, 100000)).toBe(false)
  })

  it('regroupe les occupations de PLUSIEURS véhicules par slot (un joueur change de monture)', () => {
    const first = track({ slot: 700, rides: [ride({ slot: 5, seat: 0, t0: 0, t1: 50 })] })
    const second = track({ slot: 701, rides: [ride({ slot: 5, seat: 0, t0: 60, t1: 120 })] })
    const isEmbarkedAt = buildEmbarkedPredicate([first, second])
    expect(isEmbarkedAt(5, 25)).toBe(true)
    expect(isEmbarkedAt(5, 55)).toBe(false) // à pied entre les deux véhicules
    expect(isEmbarkedAt(5, 90)).toBe(true)
  })
})

describe('vehicleScreenLengthPx / vehicleSpriteScale — taille (manifeste factice, Mongoose vs Scorpion)', () => {
  // Sprites FACTICES : mêmes dimensions et mm/px que les fichiers réels du lot A (statut
  // "valide" au 2026-08-31), mais lus ici comme un pur couple de nombres — aucun fichier chargé.
  const MONGOOSE_H_PX = 128
  const SCORPION_H_PX = 388
  const MM_PER_PX = 10

  it('le Mongoose (référence de calibration) mesure entre 1,5 et 2 pions de long', () => {
    const pionLengthPx = VEHICLE_FLOOR_PX // = CORE_RADIUS * 2, l’ancre de la règle
    const mongoose = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX)
    expect(mongoose).toBeGreaterThanOrEqual(1.5 * pionLengthPx)
    expect(mongoose).toBeLessThanOrEqual(2 * pionLengthPx)
  })

  it('proportionnalité ENTRE véhicules : le Scorpion (3,03x la hauteur native) est visiblement plus grand', () => {
    const mongoose = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX)
    const scorpion = vehicleScreenLengthPx(SCORPION_H_PX, MM_PER_PX)
    expect(scorpion).toBeGreaterThan(mongoose)
    // Sous le plafond doux, la proportion RÉELLE (mm-monde) est préservée exactement.
    expect(scorpion / mongoose).toBeCloseTo(SCORPION_H_PX / MONGOOSE_H_PX, 5)
  })

  it('vehicleSpriteScale rend un facteur qui, appliqué à la hauteur native, redonne la longueur d’écran', () => {
    const scale = vehicleSpriteScale(MONGOOSE_H_PX, MM_PER_PX)
    expect(scale * MONGOOSE_H_PX).toBeCloseTo(vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX), 6)
  })

  it('plancher de lisibilité : un sprite minuscule ne descend jamais sous VEHICLE_FLOOR_PX', () => {
    expect(vehicleScreenLengthPx(1, 0.001)).toBe(VEHICLE_FLOOR_PX)
  })

  it('plafond DOUX : au-delà du seuil, la croissance ralentit mais ne s’arrête jamais', () => {
    const huge = vehicleScreenLengthPx(100000, MM_PER_PX)
    const evenHuger = vehicleScreenLengthPx(200000, MM_PER_PX)
    // Les deux sont bien dans la zone compressée (au-delà du plafond doux)...
    expect(huge).toBeGreaterThan(VEHICLE_SOFT_CEIL_PX)
    // ... toujours strictement croissant (le plafond n'est pas un mur)...
    expect(evenHuger).toBeGreaterThan(huge)
    // ... mais SOUS-LINÉAIRE : doubler l'entrée est très loin de doubler la sortie, alors que la
    // partie non compressée (`vehicleScreenLengthPx` sous le plafond) est, elle, EXACTEMENT
    // linéaire (cf. le test de proportionnalité Mongoose/Scorpion ci-dessus).
    expect(evenHuger).toBeLessThan(huge * 1.5)
  })

  it('dimensions dégénérées (image pas encore chargée, manifeste absent) : longueur nulle', () => {
    expect(vehicleScreenLengthPx(0, 10)).toBe(0)
    expect(vehicleScreenLengthPx(128, 0)).toBe(0)
    expect(vehicleSpriteScale(0, 10)).toBe(0)
  })
})
