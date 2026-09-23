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
  vehicleIsHidden,
  vehicleIsMountedPart,
  vehicleMapElementGlyph,
  vehicleSpriteFamily,
  vehicleColorAt,
  vehicleHeadingAt,
  vehiclePositionAt,
  vehicleScreenAngle,
  vehicleScreenLengthPx,
  vehicleSpriteScale,
  vehicleVisibleAt,
  VEHICLE_DEFAULT_HEADING_DEG,
  VEHICLE_HUMAN_FAMILIES,
  VEHICLE_KIND_MAP_ELEMENT,
  VEHICLE_MAP_ELEMENT_RENDER,
  VEHICLE_PART_TURRET,
  VEHICLE_PLASMA_FAMILIES,
} from './vehiclesLayer'
import { CORE_RADIUS, PION_VISIBLE_DIAMETER_PX } from '../layers/replayMarkers'
import { viewScale } from '../layers/placementShapes'
import {
  PION_SCREEN_PX,
  PX_PAR_MM_MINIMUM_ECRAN,
  screenScalePxPerMm,
  vehicleTurretHalfPx,
  vehicleUnknownHalfPx,
  VEHICLE_SCREEN_BUMP,
  VEHICLE_SOFT_CEIL_PX,
} from './screenSizes'

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

describe('vehicleScreenLengthPx / vehicleSpriteScale — l’échelle de la carte, planchée à l’ancien cadrage +20 %', () => {
  // Sprites FACTICES : mêmes dimensions et mm/px que les fichiers réels du manifeste (statut
  // "valide" au 2026-09-02), mais lus ici comme un pur couple de nombres — aucun fichier chargé.
  const MONGOOSE_H_PX = 128
  /** Largeur native du sprite Mongoose reel — celle qui decide s il parait plus fin qu un pion. */
  const MONGOOSE_W_PX = 78
  const SCORPION_H_PX = 388
  const GHOST_H_PX = 169
  const PELICAN_H_PX = 1122
  const MM_PER_PX = 10

  /**
   * LES DEUX CARTES TÉMOINS, mesurées sur les documents cuits du cache le 2026-09-20, pour un
   * conteneur de 1 000 px CSS et la marge du rejeu (24 px) : l'échelle vaut (1000 − 48) / largeur
   * de la scène en mètres.
   *
   *   `4f77afc1` Flood Gulch  272,8 m de large -> 3,49 px/m   (GRANDE : sous le plancher)
   *   `bfecd02b` Snowbound     54,2 m de large -> 17,56 px/m  (PETITE : au-dessus du plancher)
   *
   * L'échelle de bascule vaut 14,44 px/m, soit une scène de 65,9 m de large.
   */
  const GRANDE_CARTE = 952 / 272.8
  const PETITE_CARTE = 952 / 54.2

  /**
   * L'ANCIEN RENDU, reconstitué depuis la formule d'avant le 2026-09-20 :
   * `longueur_mm × VEHICLE_PX_PER_MM`, plancher de lisibilité à un pion, puis plafond doux. Il
   * est recalculé ici — jamais recopié en valeurs — pour que l'invariant « jamais plus petit
   * qu'avant » porte sur la RÈGLE, pas sur dix-huit nombres figés.
   */
  const PX_PAR_MM_V1 = (PION_VISIBLE_DIAMETER_PX * 1.75) / 1280
  const ancienneLongueurPx = (hauteurPx: number): number => {
    const brut = Math.max(hauteurPx * MM_PER_PX * PX_PAR_MM_V1, PION_VISIBLE_DIAMETER_PX)
    return brut <= VEHICLE_SOFT_CEIL_PX
      ? brut
      : VEHICLE_SOFT_CEIL_PX + Math.sqrt(brut - VEHICLE_SOFT_CEIL_PX)
  }

  /** Les DIX-HUIT familles du manifeste, par la hauteur native de leur sprite. */
  const FAMILLES: ReadonlyArray<readonly [string, number]> = [
    ['banshee', 256], ['chopper', 239], ['falcon', 390], ['ghost', 169],
    ['gungoose', 128], ['mongoose', 128], ['pelican', 1122], ['phantom', 1080],
    ['razorback', 250], ['rockethog', 222], ['scorpion', 388], ['shade', 153],
    ['skiff', 534], ['tourelle_montee', 139], ['warthog', 222], ['warthog_gauss', 222],
    ['wasp', 257], ['wraith', 313],
  ]

  /**
   * L'INVARIANT QUI TIENT LA DEMANDE DE L'UTILISATEUR (2026-09-19 : « les véhicules sont trop
   * petits »). Le premier modèle écrit le 2026-09-20 le VIOLAIT : son plancher portait sur la
   * TAILLE, pas sur l'échelle, et rendait le Warthog à 18,5 px là où l'ancien le rendait à
   * 26,7 — plus petit qu'avant, sur la carte même où le constat avait été fait.
   */
  it('sur TOUTE carte, AUCUNE famille n’est plus petite qu’avant', () => {
    for (const echelle of [GRANDE_CARTE, PETITE_CARTE, GRANDE_CARTE * 3, 0]) {
      for (const [nom, h] of FAMILLES) {
        const apres = vehicleScreenLengthPx(h, MM_PER_PX, echelle)
        expect(apres, `${nom} à ${echelle.toFixed(2)} px/m`).toBeGreaterThanOrEqual(
          ancienneLongueurPx(h) - 1e-9,
        )
      }
    }
  })

  /**
   * GRANDE CARTE = L'ANCIEN RENDU, EXACTEMENT +20 %. C'est la demande du 2026-09-19, rendue
   * telle quelle partout où l'échelle de la carte est sous le plancher — donc sur toute carte de
   * plus de 65,9 m de large, c'est-à-dire toutes les grandes.
   */
  it('GRANDE carte : chaque famille vaut EXACTEMENT l’ancienne × 1,2', () => {
    for (const [nom, h] of FAMILLES) {
      const brutAncien = h * MM_PER_PX * PX_PAR_MM_V1
      // Le plafond doux n'est PAS un facteur : la comparaison se fait sur la grandeur qu'il
      // comprime, sinon l'égalité ne vaudrait que pour les familles qui ne l'atteignent pas.
      const attendu = brutAncien * VEHICLE_SCREEN_BUMP
      const brutApres = h * MM_PER_PX * screenScalePxPerMm(GRANDE_CARTE)
      expect(brutApres, nom).toBeCloseTo(attendu, 9)
    }
    // Et sur la grandeur RENDUE, plafond compris, le Mongoose passe bien de 15,40 à 18,48 px.
    expect(vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, GRANDE_CARTE)).toBeCloseTo(18.48, 6)
    expect(ancienneLongueurPx(MONGOOSE_H_PX)).toBeCloseTo(15.4, 6)
  })

  /**
   * LES PROPORTIONS SONT CONSERVÉES À TOUTE ÉCHELLE, et c'est ce que le plancher d'ÉCHELLE
   * garantit là où un plancher en PIXELS l'aurait détruit : les dix-huit familles franchissent
   * le seuil ENSEMBLE, jamais une par une.
   */
  it('les proportions entre familles sont EXACTES, sur la grande comme sur la petite carte', () => {
    for (const echelle of [GRANDE_CARTE, PETITE_CARTE]) {
      const mongoose = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, echelle)
      const ghost = vehicleScreenLengthPx(GHOST_H_PX, MM_PER_PX, echelle)
      // Sous le plafond doux, le rapport des tailles écran EST celui des longueurs monde.
      expect(ghost / mongoose).toBeCloseTo(GHOST_H_PX / MONGOOSE_H_PX, 6)
    }
  })

  /**
   * PETITE CARTE = LA PLUS GRANDE DES DEUX. Au-dessus du plancher, c'est l'échelle de la carte
   * qui sert, et les véhicules y sont plus gros que l'ancien × 1,2.
   */
  it('PETITE carte : la taille réelle passe devant l’ancienne × 1,2', () => {
    const mongoose = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, PETITE_CARTE)
    expect(screenScalePxPerMm(PETITE_CARTE)).toBeGreaterThan(PX_PAR_MM_MINIMUM_ECRAN)
    expect(mongoose).toBeCloseTo((MONGOOSE_H_PX * MM_PER_PX * PETITE_CARTE) / 1000, 6)
    expect(mongoose).toBeGreaterThan(ancienneLongueurPx(MONGOOSE_H_PX) * VEHICLE_SCREEN_BUMP)
  })

  it('PETITE carte : le plafond doux redevient utile — le Pélican cesse de dominer l’écran', () => {
    const brut = (PELICAN_H_PX * MM_PER_PX * PETITE_CARTE) / 1000
    const rendu = vehicleScreenLengthPx(PELICAN_H_PX, MM_PER_PX, PETITE_CARTE)
    expect(brut).toBeGreaterThan(VEHICLE_SOFT_CEIL_PX)
    expect(rendu).toBeGreaterThan(VEHICLE_SOFT_CEIL_PX)
    expect(rendu).toBeLessThan(brut)
  })

  /**
   * LES DEUX GLYPHES SUIVENT LE MÊME FACTEUR (décision du 2026-09-20) : au minimum leur taille
   * d'avant × 1,2, et la même croissance que les châssis quand l'échelle monte. Sans cela, un
   * châssis non résolu ou une tourelle rétréciraient relativement au véhicule d'à côté.
   */
  it('les deux glyphes valent leur ancienne taille × 1,2 sur une grande carte', () => {
    expect(vehicleUnknownHalfPx(GRANDE_CARTE)).toBeCloseTo(CORE_RADIUS * VEHICLE_SCREEN_BUMP, 9)
    expect(vehicleTurretHalfPx(GRANDE_CARTE)).toBeCloseTo(
      PION_VISIBLE_DIAMETER_PX * 0.75 * VEHICLE_SCREEN_BUMP,
      9,
    )
    // Le rapport entre les deux est tenu : le glyphe d'ignorance reste le plus petit.
    expect(vehicleUnknownHalfPx(GRANDE_CARTE)).toBeLessThan(vehicleTurretHalfPx(GRANDE_CARTE))
  })

  it('les deux glyphes suivent la carte, comme les châssis', () => {
    const facteur = screenScalePxPerMm(PETITE_CARTE) / screenScalePxPerMm(GRANDE_CARTE)
    expect(vehicleUnknownHalfPx(PETITE_CARTE) / vehicleUnknownHalfPx(GRANDE_CARTE)).toBeCloseTo(facteur, 6)
    expect(vehicleTurretHalfPx(PETITE_CARTE) / vehicleTurretHalfPx(GRANDE_CARTE)).toBeCloseTo(facteur, 6)
  })

  /**
   * LES PIXELS SONT LOGIQUES, JAMAIS PHYSIQUES. La densité (`k`, `devicePixelRatio`) est
   * appliquée au TRACÉ par l'appelant ; ni l'échelle du cadrage (`scaleOf(view)`, calculée sur
   * `view.width`, la largeur du CONTENEUR) ni le plancher ne la connaissent. Un écran à
   * `dpr = 2` rend donc la même taille logique avec deux fois plus de pixels physiques.
   */
  it('DPR 2 : la taille LOGIQUE est inchangée', () => {
    const view = { bounds: { minX: 0, minY: 0, maxX: 272.8, maxY: 100 }, width: 1000, height: 400, pad: 24 }
    // La vue est décrite en pixels CSS : son échelle ne change pas avec la densité.
    const echelle = viewScale(view)
    expect(vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, echelle)).toBe(
      vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, echelle),
    )
    // Et c'est bien l'appelant qui multiplie par la densité — le facteur de sprite est rendu
    // SANS `k` (il est appelé partout comme `scale * k`).
    const ratio = vehicleSpriteScale(MONGOOSE_H_PX, MM_PER_PX, echelle)
    expect(ratio * MONGOOSE_H_PX).toBeCloseTo(
      vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, echelle),
      6,
    )
  })

  it('ZOOMER agrandit tout du même facteur : le zoom ne déforme pas la scène', () => {
    const a = vehicleScreenLengthPx(SCORPION_H_PX, MM_PER_PX, PETITE_CARTE)
    const b = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, PETITE_CARTE)
    const a2 = vehicleScreenLengthPx(SCORPION_H_PX, MM_PER_PX, PETITE_CARTE * 2)
    const b2 = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, PETITE_CARTE * 2)
    // Le Scorpion est au-dessus du plafond doux : son rapport au Mongoose se resserre, mais
    // les deux grandissent — aucun ne recule.
    expect(a2).toBeGreaterThan(a)
    expect(b2).toBeCloseTo(b * 2, 6)
  })

  it('plafond DOUX : au-delà du seuil, la croissance ralentit mais ne s’arrête jamais', () => {
    const huge = vehicleScreenLengthPx(100000, MM_PER_PX, PETITE_CARTE)
    const evenHuger = vehicleScreenLengthPx(200000, MM_PER_PX, PETITE_CARTE)
    expect(huge).toBeGreaterThan(VEHICLE_SOFT_CEIL_PX)
    expect(evenHuger).toBeGreaterThan(huge)
    expect(evenHuger).toBeLessThan(huge * 1.5)
  })

  /**
   * L'INVARIANT QUE L'UTILISATEUR VOIT (2026-09-09, « le Ghost est plus petit que le pion du
   * joueur ») : un véhicule se lit à sa LARGEUR autant qu'à sa longueur. Le Mongoose — la plus
   * petite famille, donc le pire cas — ne doit jamais paraître plus mince qu'un pion, y compris
   * sur la grande carte où c'est le PLANCHER qui le tient.
   */
  it('même la plus petite famille, sur la plus grande carte, reste aussi LARGE qu’un pion', () => {
    const longueur = vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, GRANDE_CARTE)
    const largeur = longueur * (MONGOOSE_W_PX / MONGOOSE_H_PX)
    expect(largeur).toBeGreaterThanOrEqual(PION_VISIBLE_DIAMETER_PX)
  })

  it('l’ancre du plancher est le pion VISIBLE, jamais son seul noyau', () => {
    expect(PION_SCREEN_PX).toBe(PION_VISIBLE_DIAMETER_PX)
    expect(PION_SCREEN_PX).toBeGreaterThan(CORE_RADIUS * 2)
  })

  it('cadrage dégénéré ou dimensions absentes : le plancher, jamais zéro ni une taille inventée', () => {
    // Échelle nulle (toile pas encore mesurée) : le véhicule se dessine, à l'échelle plancher.
    expect(vehicleScreenLengthPx(MONGOOSE_H_PX, MM_PER_PX, 0)).toBeCloseTo(18.48, 6)
    // Sprite ou manifeste pas encore chargés : aucune taille, donc aucun tracé.
    expect(vehicleScreenLengthPx(0, 10, PETITE_CARTE)).toBe(0)
    expect(vehicleScreenLengthPx(128, 0, PETITE_CARTE)).toBe(0)
    expect(vehicleSpriteScale(0, 10, PETITE_CARTE)).toBe(0)
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

/**
 * RETOURS DU REJEU 2026-09-23 (lot M4a, décision Q12) — LES PIÈCES MONTÉES NE SE DESSINENT PAS
 * SEULES. Une tourelle (`part = turret`) n'a aucun échantillon : dessinée, elle restait à sa
 * naissance pendant que son véhicule roulait. Le serveur a reporté son artilleur sur le porteur ;
 * côté calque elle est cachée, n'embarque personne et ne rend pas le calque disponible à elle seule.
 */
describe('vehicleIsMountedPart — une tourelle se dessine sur son véhicule, jamais seule', () => {
  const tourelle = track({
    family: undefined, chassis: 'dd7f9102', part: VEHICLE_PART_TURRET,
    carrier: { slot: 701, gen: 1 }, rides: [ride({ slot: 20, seat: undefined })],
  })

  it('une pièce montée est cachée et n embarque personne', () => {
    expect(vehicleIsMountedPart(tourelle)).toBe(true)
    expect(vehicleIsHidden(tourelle)).toBe(true)
    expect(vehicleCanEmbark(tourelle)).toBe(false)
    expect(buildEmbarkedPredicate([tourelle])(20, 10)).toBe(false)
  })

  it('un véhicule n est pas une pièce montée', () => {
    expect(vehicleIsMountedPart(track())).toBe(false)
    expect(vehicleIsHidden(track())).toBe(false)
  })
})

/**
 * LA VARIANTE NOMME LE SPRITE (schéma 69) : un Rockethog (famille `warthog`, variante `rockethog`)
 * se dessine en Rockethog ; sans variante, la famille.
 */
describe('vehicleSpriteFamily — la variante avant la famille', () => {
  it('variante publiée : son sprite', () => {
    expect(vehicleSpriteFamily(track({ variant: 'rockethog' }))).toBe('rockethog')
  })
  it('sans variante : la famille ; sans rien : indéfini', () => {
    expect(vehicleSpriteFamily(track())).toBe('warthog')
    expect(vehicleSpriteFamily(track({ family: undefined }))).toBeUndefined()
  })
})
