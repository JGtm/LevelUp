/**
 * useReplayAbilityFx — Seul et Masquer pour le câble du grappin et la poussée du répulseur : le câble est à
 * son porteur, la poussée à la personne poussée ET à son auteur.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { useReplayAbilityFx } from './useReplayAbilityFx'

const VIEW: CanvasView = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 200, pad: 0 }

const doc = testReplayDoc({
  frameIntervalMs: 100,
  tracks: [
    { slot: 5, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 2, y: 2 }, { t: 100, x: 4, y: 2 }] },
    { slot: 7, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 8, y: 8 }, { t: 100, x: 8, y: 6 }] },
  ],
  grappleLines: [{ slot: 5, t0: 20, t1: 40, ax: 8, ay: 2 }],
  knockbacks: [{ t: 60, slot: 5, by: 7, vx: 0, vy: 6, vz: 0 }],
})

const GRAPPLE_FRAME = 30
const KNOCKBACK_FRAME = 61
const tout = { slotAt: () => true, knockback: () => true }

function ops(frame: number, focusFilter?: Parameters<typeof useReplayAbilityFx>[0]['focusFilter']) {
  const { result } = renderHook(() =>
    useReplayAbilityFx({
      doc, view: VIEW, grappleInk: 'g', colorOfSlot: () => 'c', reducedMotion: false, focusFilter,
    }),
  )
  const { ops: out, ctx } = recordingContext()
  result.current.paint(ctx, frame, 1)
  return out.filter((o) => o.op === 'stroke' || o.op === 'fill').length
}

describe('useReplayAbilityFx — Seul et Masquer', () => {
  it('le câble du grappin est à son porteur', () => {
    expect(ops(GRAPPLE_FRAME)).toBeGreaterThan(0)
    expect(ops(GRAPPLE_FRAME, tout)).toBe(ops(GRAPPLE_FRAME))
    expect(ops(GRAPPLE_FRAME, { ...tout, slotAt: () => false })).toBe(0)
  })

  it('le prédicat du câble reçoit le slot du porteur et l’image de départ', () => {
    const vus: number[][] = []
    ops(GRAPPLE_FRAME, { ...tout, slotAt: (slot, frame) => { vus.push([slot, frame]); return true } })
    expect(vus).toEqual([[5, 20]])
  })

  it('la poussée du répulseur est à la personne poussée ET à son auteur', () => {
    expect(ops(KNOCKBACK_FRAME)).toBeGreaterThan(0)
    expect(ops(KNOCKBACK_FRAME, { ...tout, knockback: () => false })).toBe(0)
    const vus: unknown[] = []
    ops(KNOCKBACK_FRAME, { ...tout, knockback: (k) => { vus.push([k.slot, k.by, k.frame]); return true } })
    expect(vus).toEqual([[5, 7, 60]])
  })
})
