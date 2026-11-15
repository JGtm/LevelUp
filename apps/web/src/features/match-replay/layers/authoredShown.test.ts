/**
 * Les calques d'objectif honorent Seul et Masquer : ce qu'un joueur PORTE (couronne, crâne, bombe, drapeau)
 * ou ACCOMPLIT (capture, explosion, pulse) part avec lui ; ce qui n'est à personne (bombe libre, base et
 * drapeau tombé) reste.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBombCarry, ReplayFlagCarryReady, ReplaySkullCarry, ReplayVipPeriod } from '../../../lib/replay/replayNormalize'
import type { CanvasView } from '../model/replayView'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import { drawBombBlastFx } from './bombBlastFx'
import { drawBombCarrier } from './bombCarrierLayer'
import { drawFlagCaptureFx } from './flagCaptureFx'
import { drawFlagCarries, flagAt } from './flagCarriesLayer'
import { drawObjectivePulses } from './objectivesLayer'
import { drawSkullCarrier } from './skullCarrierLayer'
import { drawVipCrown } from './vipCrownLayer'

const VIEW: CanvasView = { bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 }, width: 200, height: 200, pad: 0 }
const posOf = () => ({ x: 50, y: 50 })
const ONLY_A = (xuid: string) => xuid === 'A'
const NOT_A = (xuid: string) => xuid !== 'A'

/** Le nombre d'opérations de remplissage ou de trait : 0 = rien n'a été dessiné. */
function ink(run: (ctx: CanvasRenderingContext2D) => void): number {
  const { ops, ctx } = recordingContext()
  run(ctx)
  return ops.filter((o: CanvasOp) => o.op === 'fill' || o.op === 'stroke').length
}

describe('couronne, crâne, bombe portés', () => {
  const vip: ReplayVipPeriod = { xuid: 'A', t0: 0, t1: 100, closed: true }
  const skull: ReplaySkullCarry = { xuid: 'A', t0: 0, t1: 100, closed: true }
  const bomb: ReplayBombCarry = { xuid: 'A', t0: 0, t1: 100, closed: true }
  const style = { ink: 'i', inkOf: () => 'i', outline: 'o', reducedMotion: true, frameMs: 100 }

  it('la couronne du VIP : Seul la garde pour lui, Masquer la retire', () => {
    const run = (shownCarrier?: (x: string) => boolean) =>
      ink((ctx) => drawVipCrown(ctx, { style, posOf, shownCarrier }, [vip], VIEW, 10))
    expect(run()).toBeGreaterThan(0)
    expect(run(ONLY_A)).toBe(run())
    expect(run(NOT_A)).toBe(0)
  })

  it('le crâne porté : Seul le garde pour lui, Masquer le retire', () => {
    const run = (shownCarrier?: (x: string) => boolean) =>
      ink((ctx) => drawSkullCarrier(ctx, { style, posOf, shownCarrier }, [skull], VIEW, 10))
    expect(run()).toBeGreaterThan(0)
    expect(run(ONLY_A)).toBe(run())
    expect(run(NOT_A)).toBe(0)
  })

  it('la bombe est à son porteur, portée comme libre faute de position (même règle que le drapeau)', () => {
    const porte = (shownCarrier?: (x: string) => boolean) =>
      ink((ctx) => drawBombCarrier(ctx, { style, posOf, shownCarrier }, [bomb], [], VIEW, 10))
    expect(porte()).toBeGreaterThan(0)
    expect(porte(NOT_A)).toBe(0)
    // Porteur muet à partir de l'image 5 : la bombe est libre, aucun joueur ne la porte plus.
    const libre = (shownCarrier?: (x: string) => boolean) =>
      ink((ctx) =>
        drawBombCarrier(ctx, { style, posOf: (_x, f) => (f <= 5 ? { x: 50, y: 50 } : null), shownCarrier }, [bomb], [], VIEW, 10))
    expect(libre()).toBeGreaterThan(0)
    expect(libre(ONLY_A)).toBe(libre())
    expect(libre(NOT_A)).toBe(0)
  })
})

describe('drapeaux', () => {
  const flag: ReplayFlagCarryReady = {
    team: 0,
    spans: [
      { state: 'home', t0: 0, t1: 9, xuid: null, x: 10, y: 10, returnProgress: [] },
      { state: 'carried', t0: 10, t1: 19, xuid: 'A', x: 50, y: 50, returnProgress: [] },
      { state: 'dropped', t0: 20, t1: 29, xuid: 'A', x: 70, y: 70, returnProgress: [] },
    ],
  }
  const layer = (shownCarrier?: (x: string) => boolean) => ({
    style: { colorOfTeam: () => 'c', outline: 'o', reducedMotion: true, frameMs: 100 },
    posOf,
    shownCarrier,
  })
  const draw = (frame: number, shownCarrier?: (x: string) => boolean) =>
    ink((ctx) => drawFlagCarries(ctx, layer(shownCarrier), [flag], VIEW, frame))

  it('le drapeau PORTÉ est à son porteur : Seul le garde pour lui, Masquer le retire', () => {
    expect(draw(15)).toBeGreaterThan(0)
    expect(draw(15, ONLY_A)).toBe(draw(15))
    expect(draw(15, NOT_A)).toBeLessThan(draw(15))
  })

  it('Masquer le porteur retire le glyphe mais laisse la base rappelée', () => {
    // Porté : le rappel creux de la base + le glyphe. Retiré : le rappel seul, donc ni zéro ni tout.
    expect(draw(15, NOT_A)).toBeGreaterThan(0)
  })

  it('un drapeau TOMBÉ ou à la base n’est à personne, même quand son dernier porteur est masqué', () => {
    expect(draw(25, NOT_A)).toBe(draw(25))
    expect(draw(5, NOT_A)).toBe(draw(5))
  })

  it('le survol ne trouve pas un glyphe retiré', () => {
    const at = { x: 100, y: 100 }
    expect(flagAt([flag], layer(), VIEW, 15, at)).not.toBeNull()
    expect(flagAt([flag], layer(NOT_A), VIEW, 15, at)).toBeNull()
  })

  it('l’onde de capture est à son auteur', () => {
    const fx = [{ frame: 10, x: 50, y: 50, xuid: 'A' }]
    const run = (shownAuthor?: (x: string) => boolean) =>
      ink((ctx) =>
        drawFlagCaptureFx(ctx, fx, VIEW, { frame: 11, hold: 10 }, { inkOf: () => 'c', reducedMotion: false, shownAuthor }))
    expect(run()).toBeGreaterThan(0)
    expect(run(NOT_A)).toBe(0)
    expect(run(ONLY_A)).toBe(run())
  })
})

describe('déflagration et pulses d’objectif', () => {
  it('l’explosion est à son auteur', () => {
    const fx = [{ frame: 10, x: 50, y: 50, xuid: 'A' }]
    const run = (shownAuthor?: (x: string) => boolean) =>
      ink((ctx) =>
        drawBombBlastFx(ctx, fx, VIEW, { frame: 11, hold: 20 }, { inkOf: () => 'c', reducedMotion: false, shownAuthor }))
    expect(run()).toBeGreaterThan(0)
    expect(run(NOT_A)).toBe(0)
    expect(run(ONLY_A)).toBe(run())
  })

  it('le pulse d’une prise de zone est à son auteur', () => {
    const pulses = [{ frame: 10, x: 50, y: 50, team: 0, xuid: 'A' }]
    const run = (shownAuthor?: (x: string) => boolean) =>
      ink((ctx) =>
        drawObjectivePulses(
          ctx, pulses, VIEW, { frame: 11, hold: 20 }, { colorOfTeam: () => 'c', shownAuthor }, false,
        ))
    expect(run()).toBeGreaterThan(0)
    expect(run(NOT_A)).toBe(0)
    expect(run(ONLY_A)).toBe(run())
  })
})
