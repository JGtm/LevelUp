/**
 * shownAtSlot.test.ts — SEUL ET MASQUER RETIRENT CE QUI EST PROPRE AU JOUEUR, ET RIEN D'AUTRE
 * (lot 4 du plan « maquettes du rejeu »).
 *
 * Un prédicat `shownAtSlot(slot, image)`, branché sur quatre calques : les pistes (pion, nom, cône,
 * traînée), les marques de tir (« ! »), les éclairs de bouche et le sillage du propulseur.
 *
 * Ce qu'ils protègent :
 *  1. SANS PRÉDICAT, RIEN NE CHANGE : le dessin est celui d'avant.
 *  2. UN SLOT FILTRÉ N'ÉCRIT PLUS RIEN, un autre slot dessine comme avant.
 *  3. LA CROIX DE MORT N'EST PAS FILTRÉE : une mort est un événement du match, elle reste
 *     visible même quand le joueur est masqué — et celles des autres quand on est en Seul.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import type { ShotFxEntry } from '../model/shotFx'
import { recordingContext } from '../test/recordingContext'
import { drawFireMarks, buildFireMarks } from './fireMark'
import { drawShotsLayer } from './replayDraw'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'
import { drawThrusterDashLayer, type ThrusterDashFx } from './thrusterDashFx'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 100, pad: 4 }
const HIDE_512 = (slot: number) => slot !== 512

function vie(over: Partial<ReplayTrackReady> = {}): ReplayTrackReady {
  return {
    slot: 512, team: -1, xuid: 'bot:B', startFrame: 0, endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z: 0, h: 90 }],
    ...over,
  } as ReplayTrackReady
}

function style(over: Partial<MarkerStyle> = {}): MarkerStyle {
  return {
    colorOfSlot: () => 'rgb(1 2 3)', ink: 'rgb(9 9 9)', frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 }, z: { min: 0, max: 10 }, k: 1,
    showAim: true, markOfSlot: () => undefined, nameOfSlot: () => 'Bot', showTrail: true,
    selfInk: 'rgb(4 4 4)', deathInk: 'rgb(5 5 5)', labelStroke: 'rgb(8 12 18)',
    offscreenLabelOf: (name, meters) => `${name} ${Math.round(meters)}`,
    ...over,
  }
}

function pistes(tracks: ReplayTrackReady[], over: Partial<MarkerStyle>) {
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, tracks, VIEW, style(over))
  return ops
}

describe('pistes — drawTracksLayer', () => {
  it('sans prédicat, le pion se dessine (comportement inchangé)', () => {
    expect(pistes([vie()], {}).some((o) => o.op === 'fill')).toBe(true)
  })

  it('un slot filtré ne dessine plus ni pion, ni nom, ni traînée', () => {
    const ops = pistes([vie()], { shownAtSlot: HIDE_512 })
    expect(ops.filter((o) => o.op === 'fill' || o.op === 'fillText' || o.op === 'stroke')).toEqual([])
  })

  it('un autre slot garde tout son dessin', () => {
    const normal = pistes([vie({ slot: 7 })], {})
    const filtre = pistes([vie({ slot: 7 })], { shownAtSlot: HIDE_512 })
    expect(filtre.length).toBe(normal.length)
  })

  it('le prédicat reçoit le slot ET l’image lue : un slot repris par un autre joueur reste affiché', () => {
    const seen: Array<[number, number]> = []
    pistes([vie()], { shownAtSlot: (slot, frame) => { seen.push([slot, frame]); return true } })
    expect(seen).toEqual([[512, 50]])
  })

  it('la CROIX DE MORT reste dessinée : une mort est un événement, jamais filtrée', () => {
    // Vie close à l'image 40 ; à l'image 50 la croix survit (fenêtre de 20 images).
    const morte = vie({ endFrame: 40, points: [{ t: 30, x: 5, y: 5, z: 0, h: 90 }, { t: 40, x: 5, y: 5, z: 0, h: 90 }] })
    const sans = pistes([morte], {})
    const filtre = pistes([morte], { shownAtSlot: () => false })
    expect(sans.length).toBeGreaterThan(0)
    expect(filtre.length).toBe(sans.length)
  })
})

describe('marques de tir — drawFireMarks', () => {
  const doc = {
    shots: [{ t: 12, x: 2, y: 2, slot: 5 }], bursts: [],
    tracks: [{ slot: 5, team: -1, points: [{ t: 10, x: 2, y: 2 }, { t: 20, x: 4, y: 2 }] }],
  } as unknown as ReplayDocumentReady
  const base = { frame: 15, hold: 4, colorOfSlot: () => 'teinte', ink: 'encre', k: 1 }

  it('sans prédicat le « ! » se dessine ; filtré, plus rien ; un autre slot, comme avant', () => {
    const marks = buildFireMarks(doc)
    const run = (shownAtSlot?: (slot: number, frame: number) => boolean) => {
      const { ops, ctx } = recordingContext()
      drawFireMarks(ctx, marks, VIEW, { ...base, shownAtSlot })
      return ops.filter((o) => o.op === 'stroke').length
    }
    expect(run()).toBe(1)
    expect(run((slot) => slot !== 5)).toBe(0)
    expect(run((slot) => slot === 5)).toBe(1)
  })
})

describe('éclairs de bouche — drawShotsLayer', () => {
  const shot: ShotFxEntry = {
    frame: 10, slot: 5, x: 5, y: 5, h: 90, fam: 'bullet', tint: 'kinetic', seed: 3, vehicleShot: null,
  } as unknown as ShotFxEntry
  const INK = { tint: { kinetic: 'K', plasma_cool: 'K', plasma_hot: 'K', forerunner: 'K', electric: 'K', needle: 'K', blast: 'K', neutral: 'K' }, core: 'C' }
  const run = (shownAtSlot?: (slot: number, frame: number) => boolean) => {
    const { ops, ctx } = recordingContext()
    drawShotsLayer(ctx, [shot], VIEW, { frame: 11, hold: 5, frameMs: 33 }, { ink: INK as never, k: 1, reducedMotion: false, shownAtSlot })
    return ops.filter((o) => o.op !== 'set globalAlpha').length
  }

  it('un tir du slot filtré ne dessine rien ; sans prédicat il dessine', () => {
    expect(run()).toBeGreaterThan(0)
    expect(run((slot) => slot !== 5)).toBe(0)
  })

  it('un tir d’un autre slot se dessine comme avant', () => {
    expect(run((slot) => slot === 5)).toBe(run())
  })
})

describe('sillage du propulseur — drawThrusterDashLayer', () => {
  const fx: ThrusterDashFx = {
    frame: 30, slot: 5, points: [{ t: 0, x: 1, y: 5 }, { t: 60, x: 9, y: 5 }] as ThrusterDashFx['points'],
    from: { x: 4, y: 5 }, to: { x: 6, y: 5 },
  }
  const run = (shownAtSlot?: (slot: number, frame: number) => boolean) => {
    const { ops, ctx } = recordingContext()
    drawThrusterDashLayer(ctx, [fx], VIEW, { frame: 30, frameMs: 33.3, k: 1, reducedMotion: false }, {
      colorOfSlot: () => 'teinte', shownAtSlot,
    })
    return ops.length
  }

  it('filtré : aucun sillage ; sinon : comme avant', () => {
    expect(run()).toBeGreaterThan(0)
    expect(run((slot) => slot !== 5)).toBe(0)
    expect(run((slot) => slot === 5)).toBe(run())
  })
})
