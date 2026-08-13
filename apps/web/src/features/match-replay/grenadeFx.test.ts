/**
 * grenadeFx.test.ts — les règles de la fin de vol : SEUL un vol certifié `rest` produit un
 * effet, le point est le DERNIER pas du projectile LIÉ (jamais le point de lancer), et un
 * lien hors bornes est ignoré plutôt que de poser un effet au hasard.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { buildGrenadeRestFx, DYNAMO_RANK } from './grenadeFx'
import { testReplayDoc } from './test/testDoc'

function docWith(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    projectiles: [
      // Vol certifié : repos en (8, 9) à la frame 10 + 14 = 24.
      { t0: 10, p: [[0, 2, 3], [7, 5, 6], [14, 8, 9]], rest: true },
      // Réplication interrompue SANS repos certifié : aucun effet.
      { t0: 30, p: [[0, 1, 1], [5, 2, 2]] },
    ],
    grenades: [
      { t: 10, slot: 0, i: 1, x: 2, y: 3, rank: DYNAMO_RANK, s: 'projectile', proj: 0 },
      { t: 30, slot: 0, i: 2, x: 1, y: 1, rank: 0, s: 'projectile', proj: 1 },
      // Sans lien : rien à poser.
      { t: 50, slot: 0, i: 3, x: 4, y: 4, rank: 0, s: 'biped' },
    ],
    ...over,
  })
}

describe('buildGrenadeRestFx', () => {
  it('pose l effet au DERNIER pas du projectile lié, à sa frame de repos', () => {
    const fx = buildGrenadeRestFx(docWith())
    expect(fx).toHaveLength(1)
    expect(fx[0]).toMatchObject({ frame: 24, x: 8, y: 9, rank: DYNAMO_RANK })
  })

  it('un vol NON certifié rest ne produit rien : l arrêt de réplication n est pas une preuve', () => {
    const fx = buildGrenadeRestFx(docWith())
    expect(fx.some((e) => e.frame === 35)).toBe(false)
  })

  it('un lien hors bornes est ignoré — jamais un effet posé au hasard', () => {
    const doc = docWith({
      grenades: [{ t: 10, slot: 0, i: 1, x: 2, y: 3, rank: 1, s: 'projectile', proj: 9 }],
    })
    expect(buildGrenadeRestFx(doc)).toHaveLength(0)
  })
})
