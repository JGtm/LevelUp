/**
 * Tests — vehicleCyclesLayer : LE MARQUEUR D'UN EMPLACEMENT DE NAISSANCE, ET QUAND IL SE TAIT.
 *
 * CE QUE CE FICHIER VERROUILLE, clause par clause du lot 5.8 :
 *  - CYCLE ÉTABLI ET EMPLACEMENT LIBRE : un losange au lieu mesuré, plus le compte à rebours ;
 *  - CYCLE NON ÉTABLI : rien du tout — un emplacement sans récurrence mesurée n'est même pas
 *    publié, et le calque n'en invente aucun ;
 *  - VÉHICULE PRÉSENT : AUCUNE marque (ni losange, ni chiffre) — le sprite occupe déjà le lieu ;
 *  - PAS DE SOURCE DE COMPTE : le losange reste seul, jamais un tiret ;
 *  - LE SURVOL se rejoue sur la donnée, et il vise le lieu même quand il est occupé (une cible
 *    qui apparaîtrait avec l'image serait impossible à viser).
 *
 * Le contexte enregistreur observe la GÉOMÉTRIE ÉMISE, jamais un pixel (cf. recordingContext).
 */
import { describe, expect, it } from 'vitest'

import { count, diamondCentres, recordingContext } from '../test/recordingContext'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import type { VehicleCycle } from '../model/vehicleCycleTime'
import { drawVehicleCyclesLayer, vehicleCycleIndexAt, type VehicleCycleStyle } from './vehicleCyclesLayer'

/** 10 m de côté sur 100 px : 10 px par mètre — le même cadrage que les tests de socles. */
const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }

/** Une image de 100 ms : 10 images = 1 s. */
const FRAME_MS = 100

const STYLE: VehicleCycleStyle = {
  ink: 'encre',
  fill: 'remplissage',
  outline: 'contour',
  countdownLabel: (s) => `${Math.ceil(s)} s`,
}

function cycle(over: Partial<VehicleCycle> = {}): VehicleCycle {
  return { x: 5, y: 5, family: 'warthog', medianS: 40, p10S: 38, p90S: 42, gaps: 2, missing: 0, ...over }
}

function life(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 1,
    gen: 0,
    family: 'warthog',
    end: 'destroyed',
    t0: 0,
    t1: 100,
    t1max: 100,
    tEnd: 100,
    spawn: { x: 5, y: 5 },
    samples: [],
    rides: [],
    ...over,
  } as ReplayVehicleTrackReady
}

function draw(cycles: VehicleCycle[], lives: ReplayVehicleTrackReady[][], frame: number) {
  const { ops, ctx } = recordingContext()
  drawVehicleCyclesLayer(ctx, cycles, (i) => lives[i] ?? [], VIEW, { frame, frameMs: FRAME_MS, k: 1 }, STYLE)
  return ops
}

/** Les textes écrits, dans l'ordre (`fillText` porte le corps du compte à rebours). */
const textes = (ops: ReturnType<typeof draw>): string[] =>
  ops.filter((o) => o.op === 'fillText').map((o) => o.args[0] as string)

describe('vehicleCyclesLayer — le marqueur', () => {
  it('cycle ÉTABLI et emplacement LIBRE : un losange au lieu mesuré, et le compte à rebours', () => {
    const ops = draw([cycle()], [[life({ tEnd: 100, t1max: 100 })]], 150)
    const centres = diamondCentres(ops)
    expect(centres).toHaveLength(1)
    // 5 m sur 10 px/m : le losange est au centre du cadrage.
    expect(centres[0]).toEqual({ x: 50, y: 50 })
    // 5 s après la destruction, cycle de 40 s : 35 s restent.
    expect(textes(ops)).toEqual(['35 s'])
  })

  it('VÉHICULE PRÉSENT : aucune marque du tout (le sprite occupe déjà le lieu)', () => {
    const ops = draw([cycle()], [[life({ t0: 0, tEnd: 400, t1max: 400 })]], 50)
    expect(diamondCentres(ops)).toHaveLength(0)
    expect(textes(ops)).toEqual([])
  })

  it('cycle NON ÉTABLI : la liste est vide, donc rien ne se dessine', () => {
    const ops = draw([], [], 150)
    expect(count(ops, 'moveTo')).toBe(0)
    expect(textes(ops)).toEqual([])
  })

  it('PAS DE SOURCE DE COMPTE : le losange reste seul, jamais un tiret', () => {
    // Fin ni datée ni destructrice, et aucune naissance suivante : rien à prédire.
    const ops = draw([cycle()], [[life({ end: 'unknown', tEnd: undefined, t1max: 100 })]], 150)
    expect(diamondCentres(ops)).toHaveLength(1)
    expect(textes(ops)).toEqual([])
  })

  it('un compte ÉPUISÉ laisse le losange seul', () => {
    const ops = draw([cycle()], [[life({ tEnd: 100, t1max: 100 })]], 700)
    expect(diamondCentres(ops)).toHaveLength(1)
    expect(textes(ops)).toEqual([])
  })

  it('un cadrage dégénéré ne dessine rien', () => {
    const { ops, ctx } = recordingContext()
    drawVehicleCyclesLayer(
      ctx,
      [cycle()],
      () => [],
      { ...VIEW, width: 0 },
      { frame: 150, frameMs: FRAME_MS, k: 1 },
      STYLE,
    )
    expect(ops).toEqual([])
  })
})

describe('vehicleCyclesLayer — le survol', () => {
  it('vise le lieu, et rend le RANG de l emplacement', () => {
    const cycles = [cycle({ x: 2, y: 2 }), cycle({ x: 8, y: 8 })]
    // L'axe Y du monde monte, celui du canvas descend (`worldToCanvas`) : (8, 8) tombe en haut
    // à droite, (2, 2) en bas à gauche.
    expect(vehicleCycleIndexAt(cycles, VIEW, 1, { x: 80, y: 20 })).toBe(1)
    expect(vehicleCycleIndexAt(cycles, VIEW, 1, { x: 20, y: 80 })).toBe(0)
  })

  it('rend -1 loin de tout emplacement', () => {
    expect(vehicleCycleIndexAt([cycle()], VIEW, 1, { x: 5, y: 5 })).toBe(-1)
  })

  it('le plus proche l emporte quand deux emplacements se recouvrent', () => {
    const cycles = [cycle({ x: 5, y: 5 }), cycle({ x: 5.2, y: 5 })]
    expect(vehicleCycleIndexAt(cycles, VIEW, 1, { x: 52, y: 50 })).toBe(1)
  })

  it('la cible ne dépend PAS de l occupation : elle se vise aussi sous un véhicule', () => {
    // Aucune vie n'est passée au test de survol — il ne connaît que la géométrie.
    expect(vehicleCycleIndexAt([cycle()], VIEW, 1, { x: 50, y: 50 })).toBe(0)
  })
})
