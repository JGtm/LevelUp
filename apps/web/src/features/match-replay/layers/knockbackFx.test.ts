/**
 * Tests — knockbackFx (la poussée du répulseur : onde de l'auteur, sillage du poussé).
 *
 * Ce que ces tests verrouillent :
 *  - L'EFFET NE PART QU'AUX POUSSÉES PUBLIÉES, et dure `KNOCKBACK_FX_MS` en temps RÉEL
 *  - L'ONDE est centrée sur l'AUTEUR, à sa position à l'image de la poussée ; sans auteur publié,
 *    aucune onde
 *  - LE SILLAGE suit le pion du poussé et s'oriente par l'IMPULSION publiée, jamais par le
 *    déplacement ; sans impulsion, aucun sillage
 *  - MOUVEMENT RÉDUIT : l'anneau est posé à plein rayon, identique à tout âge de la fenêtre.
 */
import { describe, expect, it } from 'vitest'

import { buildKnockbackFx, drawKnockbackLayer, KNOCKBACK_FX_MS, waveProgress } from './knockbackFx'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}
const POUSSE = {
  slot: 5,
  team: -1,
  points: [
    { t: 0, x: 5, y: 5 },
    { t: 60, x: 5, y: 5 },
  ],
}
const AUTEUR = {
  slot: 7,
  team: -1,
  points: [
    { t: 0, x: 2, y: 5 },
    { t: 60, x: 2, y: 5 },
  ],
}

function docWith(knockbacks: ReplayDocumentReady['knockbacks']): ReplayDocumentReady {
  return {
    knockbacks,
    tracks: [POUSSE, AUTEUR],
  } as unknown as ReplayDocumentReady
}

const INK = 'var(--team-ally)'
const STYLE = { colorOfSlot: () => INK }

describe('buildKnockbackFx', () => {
  it('joint la poussée à la vie du poussé et à la position de l’auteur', () => {
    const fx = buildKnockbackFx(docWith([{ t: 30, slot: 5, by: 7, vx: 6, vy: 0, vz: 2 }]))
    expect(fx).toHaveLength(1)
    expect(fx[0].origin).toEqual({ x: 2, y: 5 })
    expect(fx[0].push).toEqual({ x: 6, y: 0 })
  })

  it('sans auteur publié, pas d’origine ; sans impulsion, pas de sillage', () => {
    const fx = buildKnockbackFx(docWith([{ t: 30, slot: 5 }]))
    expect(fx[0].origin).toBeNull()
    expect(fx[0].push).toBeNull()
  })
})

describe('waveProgress', () => {
  it('nulle hors de la fenêtre, croissante dedans, posée sous mouvement réduit', () => {
    expect(waveProgress(-1, false)).toBeNull()
    expect(waveProgress(KNOCKBACK_FX_MS + 1, false)).toBeNull()
    expect(waveProgress(100, false)!.reach).toBeLessThan(waveProgress(300, false)!.reach)
    expect(waveProgress(10, true)).toEqual(waveProgress(400, true))
  })
})

describe('drawKnockbackLayer', () => {
  it('dessine l’anneau à l’auteur et le sillage au poussé pendant la fenêtre, rien après', () => {
    const fx = buildKnockbackFx(docWith([{ t: 30, slot: 5, by: 7, vx: 6, vy: 0, vz: 2 }]))
    const time = { frame: 32, frameMs: 100, k: 1, reducedMotion: false }
    const pendant = recordingContext()
    drawKnockbackLayer(pendant.ctx, fx, VIEW, time, STYLE)
    const arcs = pendant.ops.filter((c) => c.op === 'arc')
    expect(arcs).toHaveLength(1)
    expect(arcs[0].args.slice(0, 2)).toEqual([20, 50])
    expect(pendant.ops.some((c) => c.op === 'fill')).toBe(true)

    const apres = recordingContext()
    drawKnockbackLayer(apres.ctx, fx, VIEW, { ...time, frame: 40 }, STYLE)
    expect(apres.ops).toEqual([])
  })

  it('sans auteur ni impulsion, aucune forme', () => {
    const fx = buildKnockbackFx(docWith([{ t: 30, slot: 5 }]))
    const rec = recordingContext()
    drawKnockbackLayer(
      rec.ctx,
      fx,
      VIEW,
      { frame: 31, frameMs: 100, k: 1, reducedMotion: false },
      STYLE,
    )
    expect(rec.ops.filter((c) => c.op === 'arc' || c.op === 'fill')).toEqual([])
  })
})

describe('drawKnockbackLayer en lecture (images fractionnaires)', () => {
  it('le sillage suit le pion entre deux images et l’onde grandit continûment', () => {
    // Le poussé file vers l'est : à l'image 30,5 son pion est à mi-chemin des images 30 et 31.
    const doc = {
      knockbacks: [{ t: 30, slot: 5, by: 7, vx: 0, vy: 6, vz: 2 }],
      tracks: [
        {
          slot: 5,
          team: -1,
          points: [
            { t: 30, x: 5, y: 5 },
            { t: 31, x: 6, y: 5 },
            { t: 40, x: 6, y: 5 },
          ],
        },
        AUTEUR,
      ],
    } as unknown as ReplayDocumentReady
    const fx = buildKnockbackFx(doc)
    const rayons: number[] = []
    for (const frame of [30.25, 30.5, 31.75, 33.5]) {
      const rec = recordingContext()
      drawKnockbackLayer(
        rec.ctx,
        fx,
        VIEW,
        { frame, frameMs: 100, k: 1, reducedMotion: false },
        STYLE,
      )
      const arc = rec.ops.find((c) => c.op === 'arc')
      rayons.push(arc ? (arc.args[2] as number) : -1)
      if (frame === 30.5) {
        // Le sillage part derrière le pion interpolé (55, 50) et s'oriente par l'impulsion (+y monde).
        const moves = rec.ops.filter((c) => c.op === 'moveTo')
        expect(moves.length).toBeGreaterThan(0)
        const [x] = moves[0].args as number[]
        expect(Math.abs(x - 55)).toBeLessThan(5)
      }
    }
    expect(rayons.every((r) => r > 0)).toBe(true)
    expect(rayons).toEqual([...rayons].sort((a, b) => a - b))
  })
})
