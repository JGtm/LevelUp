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

import type { ReplayVehicleRide, ReplayVehicleSample } from '@/lib/api/types'

import { count, recordingContext, type CanvasOp } from './test/recordingContext'
import type { PlacementView } from './placementShapes'
import type { ReplayVehicleTrackReady } from './replayNormalize'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

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

function ride(over: Partial<ReplayVehicleRide>): ReplayVehicleRide {
  return { t0: 0, t1: 100, slot: 1, src: 'event', ...over }
}

/** Un sprite déjà chargé et déjà teint : le calque n'en lit que les dimensions. */
const SPRITE = { width: 128, height: 128 } as unknown as CanvasImageSource

/** Le style par défaut : tout est allumé, tout se résout — les cas éteignent ce qu'ils testent. */
function style(over: Partial<VehicleStyle> = {}): VehicleStyle {
  return {
    neutralInk: '#neutre',
    labelStroke: '#contour',
    showNames: true,
    showAim: true,
    spriteOf: () => SPRITE,
    sizeOf: () => ({ naturalHeightPx: 128, mmPerPx: 10 }),
    colorOfSlot: () => '#equipe',
    colorOfXuid: () => null,
    nameOfSlot: () => 'PION-BRIDGE',
    nameOfXuid: () => null,
    ...over,
  }
}

function paint(tracks: ReplayVehicleTrackReady[], st: VehicleStyle = style()): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, VIEW, { frame: 50, k: 1 }, st)
  return ops
}

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

describe('drawVehiclesLayer — le CÔNE DE VISÉE du conducteur (retour utilisateur : « absent »)', () => {
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

  it('AUCUN cône sans conducteur : passager seul, siège non lu, ou véhicule vide', () => {
    expect(coneCount(paint([track({ rides: [] })]))).toBe(0)
    expect(coneCount(paint([track({ rides: [ride({ slot: 8, seat: 2 })] })]))).toBe(0)
    expect(coneCount(paint([track({ rides: [ride({ slot: 8, seat: undefined })] })]))).toBe(0)
  })

  it('AUCUN cône quand le calque de VISÉE est éteint (même bouton que les pions)', () => {
    const off = style({ showAim: false })
    expect(coneCount(paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], off))).toBe(0)
  })

  it('AUCUN cône quand l’équipe du conducteur n’est pas résolue (on ne colorie pas un inconnu)', () => {
    const anon = style({ colorOfSlot: () => null })
    expect(coneCount(paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], anon))).toBe(0)
  })

  it('le cône est posé SOUS le sprite : il se dessine avant lui', () => {
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })])
    const cone = ops.findIndex((o) => o.op === 'createRadialGradient')
    const sprite = ops.findIndex((o) => o.op === 'drawImage')
    expect(cone).toBeGreaterThanOrEqual(0)
    expect(sprite).toBeGreaterThan(cone)
  })
})
