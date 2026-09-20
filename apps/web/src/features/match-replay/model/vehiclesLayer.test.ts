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

import type { ReplayVehicleSample } from '@/lib/api/types'

import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  buildEmbarkedPredicate,
  vehicleActiveRides,
  vehicleAimAngle,
  vehicleCanEmbark,
  vehicleDestructionFrame,
  vehicleDriverAt,
  vehicleExplosionKindOf,
  vehicleIsDecor,
  vehicleMapElementGlyph,
  vehicleColorAt,
  vehicleHeadingAt,
  vehiclePositionAt,
  vehicleScreenAngle,
  vehicleScreenLengthPx,
  vehicleSpriteScale,
  vehicleVisibleAt,
  VEHICLE_CADRAGE_BUMP,
  VEHICLE_DEFAULT_HEADING_DEG,
  VEHICLE_FLOOR_PX,
  VEHICLE_HUMAN_FAMILIES,
  VEHICLE_KIND_MAP_ELEMENT,
  VEHICLE_MAP_ELEMENT_RENDER,
  VEHICLE_PLASMA_FAMILIES,
  VEHICLE_SOFT_CEIL_PX,
  VEHICLE_UNKNOWN_HALF_PX,
} from './vehiclesLayer'
import { CORE_RADIUS, PION_VISIBLE_DIAMETER_PX } from '../layers/replayMarkers'

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

function ride(over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady {
  return { t0: 0, t1: 100, slot: 1, src: 'event', aim: [], ...over }
}

describe('vehicleIsDecor / vehicleCanEmbark — le refus des familles non jouables', () => {
  it('les quatre familles de DÉCOR sont refusées (Falcon, Pelican, Phantom, Skiff)', () => {
    for (const f of ['falcon', 'pelican', 'phantom', 'skiff']) expect(vehicleIsDecor(f)).toBe(true)
  })

  it('une famille JOUABLE et un CHÂSSIS NON RÉSOLU ne sont pas du décor', () => {
    for (const f of ['warthog', 'mongoose', 'ghost', 'wasp']) expect(vehicleIsDecor(f)).toBe(false)
    expect(vehicleIsDecor(undefined)).toBe(false)
  })

  it('EMBARQUER — donc effacer un pion — n’est permis qu’à une famille jouable ET résolue', () => {
    expect(vehicleCanEmbark(track({ family: 'warthog' }))).toBe(true)
    expect(vehicleCanEmbark(track({ family: 'falcon' }))).toBe(false)
    expect(vehicleCanEmbark(track({ family: undefined }))).toBe(false)
    expect(vehicleCanEmbark(track({ family: '' }))).toBe(false)
  })
})

describe('vehicleAimAngle — le cap du CÔNE, distinct de celui du sprite nez-en-haut', () => {
  it('inverse le seul signe (convention monde -> canevas), sans le quart de tour du sprite', () => {
    expect(vehicleAimAngle(0)).toBeCloseTo(0, 10)
    expect(vehicleAimAngle(90)).toBeCloseTo(-Math.PI / 2, 10)
    expect(vehicleAimAngle(180)).toBeCloseTo(-Math.PI, 10)
  })

  it('N’EST PAS `vehicleScreenAngle` : les confondre ferait pointer le cône à 90° du nez', () => {
    expect(vehicleAimAngle(0)).not.toBeCloseTo(vehicleScreenAngle(0), 3)
    expect(vehicleAimAngle(90)).not.toBeCloseTo(vehicleScreenAngle(90), 3)
  })
})

describe('vehicleDriverAt — le conducteur, et lui seul', () => {
  it('rend l’épisode du siège 0 quand il couvre l’image', () => {
    const t = track({ rides: [ride({ slot: 9, seat: 1 }), ride({ slot: 7, seat: 0 })] })
    expect(vehicleDriverAt(t, 50)?.slot).toBe(7)
  })

  it('null pour un passager seul, un siège NON LU, ou hors de la fenêtre de l’épisode', () => {
    expect(vehicleDriverAt(track({ rides: [ride({ slot: 9, seat: 2 })] }), 50)).toBeNull()
    expect(vehicleDriverAt(track({ rides: [ride({ slot: 9, seat: undefined })] }), 50)).toBeNull()
    expect(vehicleDriverAt(track({ rides: [ride({ slot: 7, seat: 0, t0: 0, t1: 10 })] }), 50)).toBeNull()
  })
})

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

  it('`tEnd` FAIT AUTORITÉ SUR `t1max` quand la destruction est établie (schéma 39)', () => {
    const t = track({ t0: 10, t1: 90, t1max: 100, end: 'destroyed', tEnd: 60 })
    expect(vehicleVisibleAt(t, 60)).toBe(true)
    expect(vehicleVisibleAt(t, 61)).toBe(false)
    expect(vehicleVisibleAt(t, 100)).toBe(false)
  })

  it('AUCUN CHANGEMENT tant que `end` vaut `"unknown"` : `t1max` reste seul maître', () => {
    const t = track({ t0: 10, t1: 90, t1max: 100, end: 'unknown' })
    expect(vehicleVisibleAt(t, 100)).toBe(true)
    expect(vehicleVisibleAt(t, 101)).toBe(false)
  })
})

describe('vehicleDestructionFrame — schéma 39, EN AVANCE DE PHASE', () => {
  it('rend `null` tant que `end` vaut `"unknown"` (état actuel de CHAQUE artefact)', () => {
    expect(vehicleDestructionFrame(track({ end: 'unknown', tEnd: 42 }))).toBeNull()
  })

  it('rend `null` quand `end` vaut `"destroyed"` mais que `tEnd` est absent (mesure partielle)', () => {
    expect(vehicleDestructionFrame(track({ end: 'destroyed', tEnd: undefined }))).toBeNull()
  })

  it('rend `tEnd` UNIQUEMENT quand LES DEUX conditions sont réunies', () => {
    expect(vehicleDestructionFrame(track({ end: 'destroyed', tEnd: 42 }))).toBe(42)
  })
})

describe('vehicleExplosionKindOf — la table FACTION -> EFFET (demande utilisateur)', () => {
  it('les CINQ familles Covenant/Bannis reçoivent l’explosion PLASMA', () => {
    for (const f of VEHICLE_PLASMA_FAMILIES) expect(vehicleExplosionKindOf(f)).toBe('plasma')
  })

  it('les familles UNSC/humaines citées par l’utilisateur reçoivent l’explosion NORMALE', () => {
    for (const f of VEHICLE_HUMAN_FAMILIES) expect(vehicleExplosionKindOf(f)).toBe('normal')
  })

  it('une famille VIDE ou INCONNUE reçoit le repli NORMAL (neutre, documenté)', () => {
    expect(vehicleExplosionKindOf(undefined)).toBe('normal')
    expect(vehicleExplosionKindOf('')).toBe('normal')
    expect(vehicleExplosionKindOf('un_chassis_qui_n_existe_pas_encore')).toBe('normal')
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
  const ink = {
    colorOfSlot: (slot: number) => colors.get(slot) ?? null,
    colorOfXuid: () => null,
  }

  it('couleur du CONDUCTEUR (siège 0) quand elle est résolue', () => {
    const t = track({ rides: [ride({ slot: 1, seat: 0 }), ride({ slot: 2, seat: 1 })] })
    expect(vehicleColorAt(t, 50, ink)).toBe('#111')
  })

  it('à défaut (conducteur inconnu) : la couleur de N’IMPORTE QUEL occupant connu', () => {
    const t = track({ rides: [ride({ slot: 99, seat: 0 }), ride({ slot: 2, seat: 1 })] })
    expect(vehicleColorAt(t, 50, ink)).toBe('#222')
  })

  it('aucun occupant, ou aucun résolu : null (neutre, l’appelant pose son encre de thème)', () => {
    expect(vehicleColorAt(track({ rides: [] }), 50, ink)).toBeNull()
    const t = track({ rides: [ride({ slot: 98, seat: 0 }), ride({ slot: 99, seat: 1 })] })
    expect(vehicleColorAt(t, 50, ink)).toBeNull()
  })

  // ESCALADE N°1 DU RAPPORT DE VISIONNAGE (2026-09-02) : le contrat serveur de `VehicleRide.xuid`
  // promet que c'est LUI qui donne sa couleur au véhicule. Le pont slot->joueur est muet pendant
  // l'épisode (le bipède ne réplique plus) — le xuid doit donc PRIMER, et suffire seul.
  it('le XUID du document PRIME sur le pont slot->joueur', () => {
    const inkBoth = {
      colorOfSlot: () => '#pont',
      colorOfXuid: (x: string) => (x === 'X1' ? '#document' : null),
    }
    const t = track({ rides: [ride({ slot: 1, seat: 0, xuid: 'X1' })] })
    expect(vehicleColorAt(t, 50, inkBoth)).toBe('#document')
  })

  it('xuid inconnu du roster : REPLI sur le pont slot->joueur', () => {
    const inkBoth = { colorOfSlot: () => '#pont', colorOfXuid: () => null }
    const t = track({ rides: [ride({ slot: 1, seat: 0, xuid: 'inconnu' })] })
    expect(vehicleColorAt(t, 50, inkBoth)).toBe('#pont')
  })

  it('épisode SANS xuid : le pont reste la seule source', () => {
    const inkBoth = { colorOfSlot: () => '#pont', colorOfXuid: () => '#jamais' }
    const t = track({ rides: [ride({ slot: 1, seat: 0 })] })
    expect(vehicleColorAt(t, 50, inkBoth)).toBe('#pont')
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

  it('un FAUX épisode posé sur un prop de DÉCOR n’embarque PERSONNE (bug du 2026-09-02)', () => {
    // Le liant « trou de position » a prêté trois épisodes à un prop Falcon : le pion des
    // joueurs passés à côté disparaissait de la carte. Un décor n’embarque plus.
    const prop = track({ family: 'falcon', rides: [ride({ slot: 10, seat: 0, t0: 0, t1: 100 })] })
    expect(buildEmbarkedPredicate([prop])(10, 50)).toBe(false)
  })

  it('un épisode posé sur un CHÂSSIS NON RÉSOLU n’embarque personne non plus', () => {
    const unknown = track({ family: undefined, rides: [ride({ slot: 10, seat: 0, t0: 0, t1: 100 })] })
    expect(buildEmbarkedPredicate([unknown])(10, 50)).toBe(false)
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
  /** Largeur native du sprite Mongoose reel — celle qui decide s il parait plus fin qu un pion. */
  const MONGOOSE_W_PX = 78
  const SCORPION_H_PX = 388
  const MM_PER_PX = 10

  /**
   * LA FOURCHETTE A CHANGÉ LE 2026-09-19, ET C'EST UNE DÉCISION DE L'UTILISATEUR, PAS UNE
   * RÉGRESSION : « les véhicules sont trop petits, +20 % ». La cible de cadrage passe de 1,75 à
   * 2,10 pion de long pour le Mongoose (`MONGOOSE_TO_PION_RATIO × VEHICLE_CADRAGE_BUMP`), donc
   * la fourchette admise passe de [1,5 ; 2] à [1,8 ; 2,4] — les mêmes bornes, grossies du même
   * facteur. Le test épingle la FOURCHETTE, pas la valeur : il doit échouer si quelqu'un touche
   * à l'échelle sans toucher à la décision de cadrage.
   */
  it('le Mongoose (référence de calibration) mesure entre 1,8 et 2,4 pions de long', () => {
    const pionLengthPx = VEHICLE_FLOOR_PX // = le pion VISIBLE, l’ancre de la règle
    const mongoose = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX)
    expect(mongoose).toBeGreaterThanOrEqual(1.5 * VEHICLE_CADRAGE_BUMP * pionLengthPx)
    expect(mongoose).toBeLessThanOrEqual(2 * VEHICLE_CADRAGE_BUMP * pionLengthPx)
  })

  /**
   * LE GROSSISSEMENT EST GLOBAL, et c'est l'invariant que la seule fourchette ne porte pas :
   * les deux GLYPHES de la couche (losange d'un châssis non résolu, pictogramme de tourelle)
   * sont dérivés de la même constante que les châssis. Sans ce test, un futur ajustement du
   * cadrage laisserait les glyphes derrière et le calque perdrait ses proportions.
   */
  it('le grossissement de cadrage est le MÊME pour les châssis et pour les glyphes', () => {
    expect(VEHICLE_UNKNOWN_HALF_PX).toBeCloseTo(CORE_RADIUS * VEHICLE_CADRAGE_BUMP, 10)
    expect(VEHICLE_SOFT_CEIL_PX).toBeCloseTo(
      4 * 1.75 * VEHICLE_CADRAGE_BUMP * PION_VISIBLE_DIAMETER_PX,
      10,
    )
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

  /**
   * GARDE-RAIL DU 2026-09-09 (retour utilisateur : « le Ghost … est plus petit que le pion du
   * joueur ça fait hyper bizarre »).
   *
   * L'ancre était `CORE_RADIUS * 2` — le NOYAU SEUL, 6,80 px — alors qu'un pion au
   * rez-de-chaussée en mesure 8,80 de large, lisere compris. Tous les véhicules étaient donc
   * calibrés 1,29 fois trop petit. CE N'ÉTAIT PAS LE PLANCHER : mesuré sur les PNG réels et le
   * manifeste, aucune famille ne l'atteignait (la plus petite en était à 75 % au-dessus).
   */
  it('l’ancre est le pion VISIBLE, jamais son seul noyau', () => {
    expect(VEHICLE_FLOOR_PX).toBe(PION_VISIBLE_DIAMETER_PX)
    expect(VEHICLE_FLOOR_PX).toBeGreaterThan(CORE_RADIUS * 2)
  })

  /**
   * L'INVARIANT QUE L'UTILISATEUR VOIT, et que la longueur seule ne portait pas : un véhicule se
   * lit à sa LARGEUR autant qu'à sa longueur. Le Mongoose — la plus petite famille, donc le pire
   * cas — sortait à 7,25 px de large contre 8,80 pour le pion : plus étroit que le pion dont il
   * est censé faire 1,75 fois la LONGUEUR. Un sprite allongé ne doit jamais paraître plus mince
   * qu'un pion.
   */
  it('même la plus petite famille reste au moins aussi LARGE qu’un pion', () => {
    const longueur = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX)
    const largeur = longueur * (MONGOOSE_W_PX / MONGOOSE_H_PX)
    expect(largeur).toBeGreaterThanOrEqual(PION_VISIBLE_DIAMETER_PX)
  })

  it('dimensions dégénérées (image pas encore chargée, manifeste absent) : longueur nulle', () => {
    expect(vehicleScreenLengthPx(0, 10)).toBe(0)
    expect(vehicleScreenLengthPx(128, 0)).toBe(0)
    expect(vehicleSpriteScale(0, 10)).toBe(0)
  })
})

describe('éléments de carte (lot 1.9.9, décision utilisateur du 2026-09-14)', () => {
  it('la tourelle automatique bannie a un pictogramme DÉDIÉ dans la table', () => {
    expect(VEHICLE_MAP_ELEMENT_RENDER.tourelle_auto_bannie).toBe('turret')
  })

  it('les DEUX conditions sont nécessaires : la nature du document ET la forme de la table', () => {
    // La nature seule (famille inconnue de la table) ne donne pas de forme…
    expect(vehicleMapElementGlyph('famille_future', VEHICLE_KIND_MAP_ELEMENT)).toBeNull()
    // …et la table seule ne suffit pas : c'est le SERVEUR qui dit qu'une famille est un élément
    // de carte, jamais le calque. Un document muet (artefact antérieur au lot) ne déclenche rien.
    expect(vehicleMapElementGlyph('tourelle_auto_bannie', undefined)).toBeNull()
    expect(vehicleMapElementGlyph('tourelle_auto_bannie', VEHICLE_KIND_MAP_ELEMENT)).toBe('turret')
  })

  it('un élément de carte N’EMBARQUE PERSONNE — un pion effacé à tort est le pire défaut', () => {
    const t = track({ family: 'tourelle_auto_bannie' })
    expect(vehicleCanEmbark(t, VEHICLE_KIND_MAP_ELEMENT)).toBe(false)
    // CONTRÔLE : sans la nature, la même vie reste embarquable — la garde tient sur `kind`, pas
    // sur le nom de la famille (le calque ne connaît aucun châssis).
    expect(vehicleCanEmbark(t)).toBe(true)
  })

  it('le prédicat embarqué écarte les épisodes d’un élément de carte', () => {
    const tracks = [
      track({ family: 'tourelle_auto_bannie', rides: [ride({ slot: 7, t0: 0, t1: 100 })] }),
      track({ family: 'warthog', rides: [ride({ slot: 8, t0: 0, t1: 100 })] }),
    ]
    const embarque = buildEmbarkedPredicate(tracks, (f) =>
      f === 'tourelle_auto_bannie' ? VEHICLE_KIND_MAP_ELEMENT : undefined,
    )
    expect(embarque(7, 50)).toBe(false)
    expect(embarque(8, 50)).toBe(true)
  })

  it('un élément de carte N’EST PAS du décor : le décor ne se dessine pas, lui SI', () => {
    expect(vehicleIsDecor('tourelle_auto_bannie')).toBe(false)
  })
})
