/**
 * Le calque des morts honore Seul et Masquer : un effet que le prédicat écarte ne se dessine pas, un
 * effet qu'il garde se dessine comme sans filtre.
 */
import { describe, expect, it } from 'vitest'

import { killFxShown } from '../model/playerPanelLogic'
import { killFxEntry } from '../test/killFxEntry'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import { drawKillFxLayer, type KillFxStyle } from './replayDraw'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 20, maxY: 20 }, width: 400, height: 400, pad: 0 }
const STYLE: KillFxStyle = { colorOfSlot: () => 'tueur', fallback: 'repli', reducedMotion: false, k: 1 }

function trace(style: Partial<KillFxStyle>, ...slots: number[]): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  const fx = slots.map((slot) => killFxEntry({ fam: 'melee', slot, slotFrame: 10, victimSlot: 9, victimFrame: 10 }))
  drawKillFxLayer(ctx, fx, VIEW, { frame: 1, hold: 14, frameMs: 100 }, { ...STYLE, ...style })
  return ops
}

const stars = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'closePath').length

const A = { xuid: 'A', lives: [{ slot: 1, startFrame: 0, endFrame: 50, points: [] }] } as never
const focus = (over: object) => ({ player: A, follow: false, solo: false, hide: false, ...over })

describe('drawKillFxLayer — Seul et Masquer', () => {
  it('sans prédicat, tout se dessine', () => {
    expect(stars(trace({}, 1, 2))).toBe(2)
  })

  it('Seul : seul l’effet du joueur se dessine', () => {
    expect(stars(trace({ shown: killFxShown(focus({ solo: true }))! }, 1, 2))).toBe(1)
  })

  it('Masquer : l’effet du joueur est retiré, l’autre se dessine', () => {
    expect(stars(trace({ shown: killFxShown(focus({ hide: true }))! }, 1, 2))).toBe(1)
    expect(stars(trace({ shown: killFxShown(focus({ hide: true }))! }, 1))).toBe(0)
  })

  it('un effet filtré ne laisse aucune opération de dessin', () => {
    expect(trace({ shown: () => false }, 1, 2).filter((o) => o.op === 'moveTo')).toHaveLength(0)
  })
})
