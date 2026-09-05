/**
 * vehiclesPaint.test.ts — CE QUE LE CALQUE DES VÉHICULES POSE RÉELLEMENT SUR LE CANEVAS.
 *
 * POURQUOI CE FICHIER N'EXISTAIT PAS, ET CE QUE ÇA A COÛTÉ. `vehiclesLayer.test.ts` couvrait les
 * réponses PURES (orientation, taille, occupation, prédicat) et pas une seule primitive émise :
 * `drawVehiclesLayer` n'avait AUCUN test. Trois défauts sont donc partis en production sans
 * qu'un test rougisse — des entités de DÉCOR dessinées comme des véhicules (et pire, escamotant
 * les pions des joueurs passés à côté), AUCUN cône de visée sur un véhicule conduit alors que le
 * bipède embarqué n'en produit plus, et un nom d'occupant qui disparaît dès que le pont
 * slot->joueur ne résout pas. Les trois se voient ICI, et nulle part ailleurs : ce sont des
 * propriétés du TRACÉ.
 *
 * L'OUTIL EST LE CONTEXTE ENREGISTREUR DU DÉPÔT (`test/recordingContext.ts`) : il note les
 * primitives émises, sans jsdom ni node-canvas. On n'y vérifie donc pas des pixels — on vérifie
 * qu'une image est posée, qu'un texte est écrit, qu'un dégradé de cône est créé, et avec quelles
 * valeurs.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleAim, ReplayVehicleSample } from '@/lib/api/types'

import { EXPLOSION_MS } from './explosionFx'
import type { FxInk } from './fxInk'
import { count, recordingContext, type CanvasOp } from './test/recordingContext'
import { project, type PlacementView } from './placementShapes'
import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from './replayNormalize'
import { drawVehiclesLayer, type VehicleStyle, type VehicleTime } from './vehiclesPaint'

const VIEW: PlacementView = {
  bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
  width: 400,
  height: 400,
  pad: 8,
}

/** Une vie de véhicule minimale, VISIBLE à l'image 50, complétée par le cas. */
function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'inconnue',
    family: 'warthog',
    samples: [sample({ t: 0, x: 50, y: 50, h: 90 })],
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

/** Un sprite déjà chargé et déjà teint : le calque n'en lit que les dimensions. */
const SPRITE = { width: 128, height: 128 } as unknown as CanvasImageSource

/**
 * Encres d'effet FICTIVES mais DISTINCTES par teinte : un test peut ainsi affirmer « c'est LA
 * teinte plasma qui a été peinte » sans dépendre des vraies valeurs `oklch(...)` du thème (même
 * patron que `ExplosionInk` dans `explosionFx.test.ts`, qui utilise `'FIRE'`/`'CORE'`/`'SMOKE'`).
 */
const EXPLOSION_INK: FxInk = {
  tint: {
    kinetic: 'INK-KINETIC',
    plasma_cool: 'INK-PLASMA',
    plasma_hot: 'INK-PLASMA-HOT',
    forerunner: 'INK-FORERUNNER',
    electric: 'INK-ELECTRIC',
    needle: 'INK-NEEDLE',
    blast: 'INK-NORMALE',
    neutral: 'INK-NEUTRE',
  },
  core: 'INK-CORE',
}

/** Le style par défaut : tout est allumé, tout se résout — les cas éteignent ce qu'ils testent. */
function style(over: Partial<VehicleStyle> = {}): VehicleStyle {
  return {
    neutralInk: '#neutre',
    labelStroke: '#contour',
    showNames: true,
    showAim: true,
    spriteOf: () => SPRITE,
    sizeOf: () => ({ naturalWidthPx: 128, naturalHeightPx: 128, mmPerPx: 10 }),
    colorOfSlot: () => '#equipe',
    colorOfXuid: () => null,
    nameOfSlot: () => 'PION-BRIDGE',
    nameOfXuid: () => null,
    explosionInk: EXPLOSION_INK,
    reducedMotion: false,
    ...over,
  }
}

/** `frameMs` par défaut : 100 ms/frame — des nombres ronds pour dater l'âge de l'explosion. */
function paint(
  tracks: ReplayVehicleTrackReady[],
  st: VehicleStyle = style(),
  time: Partial<VehicleTime> = {},
): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, VIEW, { frame: 50, k: 1, frameMs: 100, ...time }, st)
  return ops
}

/** Les couleurs (fillStyle/strokeStyle/addColorStop) réellement peintes, dans l'ordre. */
const paintedColors = (ops: CanvasOp[]): unknown[] =>
  ops
    .filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle' || o.op === 'addColorStop')
    .map((o) => (o.op === 'addColorStop' ? o.args[1] : o.args[0]))

/** Les textes réellement écrits, dans l'ordre. */
const texts = (ops: CanvasOp[]): string[] =>
  ops.filter((o) => o.op === 'fillText').map((o) => String(o.args[0]))

/** La couleur en vigueur au moment du n-ième `fillText` (le calque pose `fillStyle` juste avant). */
function fillColorAtText(ops: CanvasOp[], nth = 0): unknown {
  let color: unknown
  let seen = 0
  for (const o of ops) {
    if (o.op === 'set fillStyle') color = o.args[0]
    if (o.op === 'fillText') {
      if (seen === nth) return color
      seen++
    }
  }
  return undefined
}

describe('drawVehiclesLayer — familles non jouables (verdict utilisateur 2026-09-02)', () => {
  it('une famille de DÉCOR ne pose RIEN : ni sprite, ni losange, ni nom', () => {
    for (const family of ['falcon', 'pelican', 'phantom', 'skiff']) {
      const ops = paint([track({ family, rides: [ride({ slot: 7, seat: 0 })] })])
      expect(count(ops, 'drawImage'), family).toBe(0)
      expect(count(ops, 'fillText'), family).toBe(0)
      expect(count(ops, 'fill'), family).toBe(0)
      expect(count(ops, 'createRadialGradient'), family).toBe(0)
    }
  })

  it('une famille JOUABLE est inchangée : sprite posé, nom écrit, cône tracé', () => {
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })])
    expect(count(ops, 'drawImage')).toBe(1)
    expect(texts(ops)).toEqual(['PION-BRIDGE'])
    expect(count(ops, 'createRadialGradient')).toBe(1)
  })

  it('un CHÂSSIS NON RÉSOLU garde son losange neutre (ce n’est pas du décor, c’est un inconnu)', () => {
    const ops = paint([track({ family: undefined })])
    expect(count(ops, 'fill')).toBe(1)
    expect(count(ops, 'drawImage')).toBe(0)
  })
})

describe('drawVehiclesLayer — le NOM de l’occupant (retour utilisateur : « je n’en vois aucun »)', () => {
  it('le nom du CONDUCTEUR est écrit sous le véhicule quand le calque des noms est allumé', () => {
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0, t0: 0, t1: 100 })] })])
    expect(texts(ops)).toEqual(['PION-BRIDGE'])
  })

  it('les occupants sont EMPILÉS conducteur d’abord, chacun sur SA ligne', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 8, seat: 1 }), ride({ slot: 7, seat: 0 })] })],
      style({ nameOfSlot: (slot) => (slot === 7 ? 'CONDUCTEUR' : 'PASSAGER') }),
    )
    expect(texts(ops)).toEqual(['CONDUCTEUR', 'PASSAGER'])
    const ys = ops.filter((o) => o.op === 'fillText').map((o) => o.args[2] as number)
    expect(ys[1]).toBeGreaterThan(ys[0])
  })

  it('AUCUN nom hors de la fenêtre de l’épisode, ni quand le calque des noms est éteint', () => {
    expect(texts(paint([track({ rides: [ride({ slot: 7, seat: 0, t0: 200, t1: 300 })] })]))).toEqual([])
    const off = style({ showNames: false })
    expect(texts(paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], off))).toEqual([])
  })

  it('LE DOCUMENT NOMME L’OCCUPANT : `ride.xuid` répond même quand le pont slot->joueur ne résout pas', () => {
    // C'EST LE CAS OBSERVÉ SUR FILM : la trace de bipède du slot n'a pas de xuid, donc
    // `rosterLogic.buildPlayers` l'écarte et `nameOfSlot` rend `null` — mais le document, lui,
    // porte le xuid de l'occupant sur l'épisode.
    const ops = paint(
      [track({ rides: [ride({ slot: 7, seat: 0, xuid: '2535400000000001' })] })],
      style({ nameOfSlot: () => null, nameOfXuid: () => 'PAR-XUID' }),
    )
    expect(texts(ops)).toEqual(['PAR-XUID'])
  })

  it('le XUID du document PRIME sur le pont slot->joueur (source la plus directe)', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 7, seat: 0, xuid: '2535400000000001' })] })],
      style({ nameOfXuid: () => 'PAR-XUID' }),
    )
    expect(texts(ops)).toEqual(['PAR-XUID'])
  })

  it('un nom SANS couleur d’équipe n’est jamais rempli à l’encre de son propre contour', () => {
    // Sinon le nom est un pâté illisible : `drawNameLabel` cerne les lettres avec `labelStroke`.
    const st = style({ colorOfSlot: () => null })
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], st)
    expect(texts(ops)).toEqual(['PION-BRIDGE'])
    expect(fillColorAtText(ops)).toBe(st.neutralInk)
    expect(fillColorAtText(ops)).not.toBe(st.labelStroke)
  })
})

describe('drawVehiclesLayer — LE CÔNE DE VISÉE DE CHAQUE OCCUPANT (schéma 39)', () => {
  const coneCount = (ops: CanvasOp[]) => count(ops, 'createRadialGradient')

  it('un véhicule avec CONDUCTEUR actif porte un cône', () => {
    expect(coneCount(paint([track({ rides: [ride({ slot: 7, seat: 0 })] })]))).toBe(1)
  })

  it('le cône est orienté par le CAP du véhicule, pas par l’angle du sprite nez-en-haut', () => {
    // Cap 0° (est) : l'angle canevas du cône vaut 0 rad — `arc` reçoit donc ±la demi-ouverture,
    // symétriques autour de zéro. L'angle du SPRITE, lui, vaudrait +90° (cf. vehicleScreenAngle).
    const ops = paint([
      track({ samples: [sample({ t: 0, x: 50, y: 50, h: 0 })], rides: [ride({ slot: 7, seat: 0 })] }),
    ])
    const arc = ops.find((o) => o.op === 'arc')
    expect(arc).toBeDefined()
    const from = arc?.args[3] as number
    const to = arc?.args[4] as number
    expect(from + to).toBeCloseTo(0, 10)
    expect(to).toBeGreaterThan(0)
  })

  it('AUCUN cône sur un véhicule VIDE : sans occupant, rien ne regarde nulle part', () => {
    expect(coneCount(paint([track({ rides: [] })]))).toBe(0)
  })

  it('UN CÔNE PAR OCCUPANT (schéma 39) : l’artilleur et le passager en ont un, eux aussi', () => {
    // C'EST LE POINT DU LOT V11. Avant le schéma 39, seul le siège 0 obtenait un cône : la visée
    // d'un artilleur était réputée absente du film. Elle y est — sur SON slot bipède, en continu.
    const trois = track({
      rides: [
        ride({ slot: 7, seat: 0, aim: [{ t: 50, h: 10 }] }),
        ride({ slot: 8, seat: 1, aim: [{ t: 50, h: 200 }] }),
        ride({ slot: 9, seat: 2, aim: [{ t: 50, h: 300 }] }),
      ],
    })
    expect(coneCount(paint([trois]))).toBe(3)
  })

  it('CHAQUE cône prend l’angle de SA visée, pas celui du voisin ni celui du châssis', () => {
    const deux = track({
      samples: [sample({ t: 0, x: 50, y: 50, h: 0 })], // châssis plein est : angle canevas 0
      rides: [
        ride({ slot: 7, seat: 0, aim: [{ t: 50, h: 90 }] }),
        ride({ slot: 8, seat: 1, aim: [{ t: 50, h: 180 }] }),
      ],
    })
    // Le milieu de l'arc est l'angle du cône : `(from + to) / 2`.
    const milieux = paint([deux])
      .filter((o) => o.op === 'arc')
      .map((o) => (((o.args[3] as number) + (o.args[4] as number)) / 2))
    expect(milieux).toHaveLength(2)
    expect(milieux[0]).toBeCloseTo((-90 * Math.PI) / 180, 10)
    expect(milieux[1]).toBeCloseTo((-180 * Math.PI) / 180, 10)
  })

  it('REPLI sur le cap du châssis quand l’épisode n’a pas de visée à cet instant', () => {
    // Châssis plein est (cap 0°) : le cône du conducteur vaut l'angle canevas 0, exactement comme
    // avant la série de visée — un artefact antérieur rend donc le même dessin qu'avant.
    const sansVisee = track({
      samples: [sample({ t: 0, x: 50, y: 50, h: 0 })],
      rides: [ride({ slot: 7, seat: 0, aim: [] })],
    })
    const arc = paint([sansVisee]).find((o) => o.op === 'arc')
    expect(((arc?.args[3] as number) + (arc?.args[4] as number)) / 2).toBeCloseTo(0, 10)
    // Une lecture TROP ANCIENNE (au-delà du maintien) est traitée comme une absence : même repli.
    const perimee = track({
      samples: [sample({ t: 0, x: 50, y: 50, h: 0 })],
      rides: [ride({ slot: 7, seat: 0, t0: 0, t1: 100, aim: [{ t: 5, h: 200 }] })],
    })
    const arc2 = paint([perimee]).find((o) => o.op === 'arc')
    expect(((arc2?.args[3] as number) + (arc2?.args[4] as number)) / 2).toBeCloseTo(0, 10)
  })

  it('L’ÉLÉVATION allonge ou raccourcit le cône, comme pour un pion', () => {
    const rayon = (aim: ReplayVehicleAim[]) => {
      const ops = paint([track({ rides: [ride({ slot: 7, seat: 0, aim })] })])
      return ops.find((o) => o.op === 'createRadialGradient')?.args[5] as number
    }
    const plat = rayon([{ t: 50, h: 90 }])
    expect(rayon([{ t: 50, h: 90, p: 60 }])).toBeGreaterThan(plat)
    expect(rayon([{ t: 50, h: 90, p: -60 }])).toBeLessThan(plat)
  })

  it('AUCUN cône quand le calque de VISÉE est éteint (même bouton que les pions)', () => {
    const off = style({ showAim: false })
    expect(coneCount(paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], off))).toBe(0)
  })

  it('AUCUN cône pour un occupant dont l’identité n’est pas résolue (on ne colorie pas un inconnu)', () => {
    const anon = style({ colorOfSlot: () => null })
    const deux = track({
      rides: [
        ride({ slot: 7, seat: 0, aim: [{ t: 50, h: 90 }] }),
        ride({ slot: 8, seat: 1, aim: [{ t: 50, h: 180 }] }),
      ],
    })
    expect(coneCount(paint([deux], anon))).toBe(0)
  })

  it('le cône est posé SOUS le sprite : il se dessine avant lui', () => {
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })])
    const cone = ops.findIndex((o) => o.op === 'createRadialGradient')
    const sprite = ops.findIndex((o) => o.op === 'drawImage')
    expect(cone).toBeGreaterThanOrEqual(0)
    expect(sprite).toBeGreaterThan(cone)
  })
})

describe('drawVehiclesLayer — LA DESTRUCTION (schéma 39, demande utilisateur : « il faut aussi la destruction et un effet UI »)', () => {
  /** Aucun cône, aucun nom : seule l'explosion peut produire un dégradé radial ou un arc ici. */
  const DESTROYED = track({ end: 'destroyed', tEnd: 50, rides: [] })

  it('AUCUN effet AVANT `tEnd`, et le déclenchement tombe PILE à `tEnd`', () => {
    expect(count(paint([DESTROYED], style(), { frame: 49 }), 'createRadialGradient')).toBe(0)
    expect(count(paint([DESTROYED], style(), { frame: 50 }), 'createRadialGradient')).toBeGreaterThan(0)
  })

  it('RIEN tant que `end` vaut `"unknown"` — même avec un `tEnd` publié (mesure Go non aboutie)', () => {
    const unresolved = track({ end: 'unknown', tEnd: 50, rides: [] })
    const ops = paint([unresolved], style(), { frame: 50 })
    expect(count(ops, 'createRadialGradient')).toBe(0)
    // Le sprite, lui, continue comme avant (repli sur `t1max`) : zéro changement visible.
    expect(count(ops, 'drawImage')).toBe(1)
  })

  it('EXPLOSION PLASMA pour une famille Covenant/Bannis (Ghost)', () => {
    const ghost = track({ end: 'destroyed', tEnd: 50, rides: [], family: 'ghost' })
    const colors = paintedColors(paint([ghost], style(), { frame: 50 }))
    expect(colors).toContain('INK-PLASMA')
    expect(colors).not.toContain('INK-NORMALE')
  })

  it('EXPLOSION NORMALE pour un véhicule humain (Warthog)', () => {
    const warthog = track({ end: 'destroyed', tEnd: 50, rides: [], family: 'warthog' })
    const colors = paintedColors(paint([warthog], style(), { frame: 50 }))
    expect(colors).toContain('INK-NORMALE')
    expect(colors).not.toContain('INK-PLASMA')
  })

  it('REPLI NORMAL pour un châssis non résolu (famille vide)', () => {
    const inconnu = track({ end: 'destroyed', tEnd: 50, rides: [], family: undefined })
    const colors = paintedColors(paint([inconnu], style(), { frame: 50 }))
    expect(colors).toContain('INK-NORMALE')
    expect(colors).not.toContain('INK-PLASMA')
  })

  it('le SPRITE NE SE DESSINE PLUS après `tEnd` — il cesse à la destruction, pas à `t1max`', () => {
    const t = track({ end: 'destroyed', tEnd: 50, t1max: 1000, rides: [], family: 'warthog' })
    expect(count(paint([t], style(), { frame: 50 }), 'drawImage')).toBe(1)
    expect(count(paint([t], style(), { frame: 51 }), 'drawImage')).toBe(0)
  })

  it('L’EXPLOSION S’ÉTEINT à `EXPLOSION_MS` : rien au-delà, quelque chose jusqu’à la borne', () => {
    // frameMs = 100 : la borne tombe pile sur une frame (50 + 24), l'image suivante en sort.
    // `drawVehiclesLayer` pose toujours son PROPRE save/restore (même sans rien à peindre), donc
    // le signal n'est pas « zéro opération » mais « zéro trace de PARTICULE d'explosion ».
    const atBound = 50 + EXPLOSION_MS / 100
    const ops = paint([DESTROYED], style(), { frame: atBound })
    const opsAfter = paint([DESTROYED], style(), { frame: atBound + 1 })
    expect(count(ops, 'set globalCompositeOperation')).toBeGreaterThan(0)
    expect(count(opsAfter, 'set globalCompositeOperation')).toBe(0)
  })

  it('ANCRÉE À LA POSITION DE `tEnd`, pas à la position courante du véhicule', () => {
    // Le véhicule continue d'écrire des échantillons après sa destruction (l'artefact ne le
    // garantit pas encore, mais le rendu ne doit pas en dépendre) : l'explosion reste au point
    // de la destruction plutôt que de suivre une trajectoire post-mortem. `drawWave` (un seul
    // `arc`, sans dérive de particule) pose son centre exactement à l'ancre — c'est le signal le
    // plus direct pour lire une position posée par le calque.
    const t = track({
      end: 'destroyed',
      tEnd: 50,
      rides: [],
      family: undefined,
      samples: [sample({ t: 0, x: 50, y: 50 }), sample({ t: 50, x: 50, y: 50 }), sample({ t: 90, x: 90, y: 90 })],
    })
    // ageMs = 100 : toujours dans la fenêtre de l'onde de choc (~650 ms), avant toute dérive.
    const ops = paint([t], style(), { frame: 51 })
    const wave = ops.find((o) => o.op === 'arc')
    expect(wave).toBeDefined()
    const atDestruction = project({ x: 50, y: 50 }, VIEW)
    const atCurrentFrame = project({ x: 51, y: 51 }, VIEW) // position COURANTE interpolée à t=51 sur [50,90]
    expect(wave?.args[0]).toBeCloseTo(atDestruction.x, 5)
    expect(wave?.args[1]).toBeCloseTo(atDestruction.y, 5)
    expect(wave?.args[0]).not.toBeCloseTo(atCurrentFrame.x, 3)
  })

  it('LE DÉCOR N’EXPLOSE JAMAIS : une famille non jouable reste muette même détruite', () => {
    // Même refus qu'à l'accoutumée (cf. le premier `describe` du fichier) : `drawVehiclesLayer`
    // pose SON PROPRE save/restore quel que soit le contenu de la boucle — le signal reste donc
    // l'ABSENCE de toute primitive DE VÉHICULE, explosion comprise.
    const falcon = track({ end: 'destroyed', tEnd: 50, rides: [], family: 'falcon' })
    const ops = paint([falcon], style(), { frame: 50 })
    expect(count(ops, 'drawImage')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'createRadialGradient')).toBe(0)
    expect(count(ops, 'set globalCompositeOperation')).toBe(0)
  })
})
