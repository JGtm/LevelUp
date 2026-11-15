/**
 * Tests — drawProjectilesLayer : le filtre de Seul et Masquer. Un vol dont l'auteur n'est pas
 * affiché n'est pas tracé ; sans filtre, tout se trace comme avant.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayProjectileReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'
import { drawProjectilesLayer } from './replayProjectiles'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }
const vol = (t0: number): ReplayProjectileReady =>
  ({ t0, p: [[0, 1, 1], [5, 4, 4]], detoT: t0 + 5, detoX: 4, detoY: 4 }) as ReplayProjectileReady

function strokes(shown?: (index: number) => boolean) {
  const { ctx, ops } = recordingContext()
  drawProjectilesLayer(ctx, [vol(10), vol(10)], VIEW, { frame: 12, tailFrames: 7 }, '#c', shown)
  return ops.filter((o) => o.op === 'stroke').length
}

describe('drawProjectilesLayer — Seul et Masquer', () => {
  it('sans filtre, chaque vol en cours est tracé', () => {
    expect(strokes()).toBe(2)
  })

  it('un vol dont l’auteur n’est pas affiché n’est pas tracé', () => {
    expect(strokes((i) => i === 0)).toBe(1)
    expect(strokes(() => false)).toBe(0)
  })
})
