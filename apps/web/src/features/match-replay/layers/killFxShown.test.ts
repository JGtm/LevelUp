/**
 * Le calque des morts honore les joueurs masqués : un effet que le prédicat écarte ne se dessine pas, un
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

const joueur = (xuid: string, slot: number) => ({ xuid, lives: [{ slot, startFrame: 0, endFrame: 50, points: [] }] }) as never
// A tient le slot 1, B le slot 2, la victime V le slot 9 : chaque effet a un tueur et une victime lus dans leur vie.
const A = joueur('A', 1)
const B = joueur('B', 2)
const V = joueur('V', 9)

describe('drawKillFxLayer — les joueurs masqués', () => {
  it('sans prédicat, tout se dessine', () => {
    expect(stars(trace({}, 1, 2))).toBe(2)
  })

  it('un joueur masqué : l’effet dont l’autre joueur est affiché se dessine encore', () => {
    expect(stars(trace({ shown: killFxShown([A])! }, 1, 2))).toBe(2)
  })

  it('tous les joueurs de l’effet masqués : il est retiré, l’autre se dessine', () => {
    // A tue V ; B tue V. A et V masqués : le premier effet n'a plus de joueur affiché, le second a B.
    expect(stars(trace({ shown: killFxShown([A, V])! }, 1, 2))).toBe(1)
    expect(stars(trace({ shown: killFxShown([A, V])! }, 1))).toBe(0)
  })

  it('Seul A (B et V masqués) : seul l’effet de A se dessine', () => {
    expect(stars(trace({ shown: killFxShown([B, V])! }, 1, 2))).toBe(1)
  })

  it('un effet filtré ne laisse aucune opération de dessin', () => {
    expect(trace({ shown: () => false }, 1, 2).filter((o) => o.op === 'moveTo')).toHaveLength(0)
  })
})
