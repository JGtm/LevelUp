/**
 * grenadeFx.test.ts — les règles de la fin de vol : l'effet se pose au DERNIER pas du
 * projectile LIÉ (jamais au point de lancer — c'est la « dernière position connue »), la
 * certification `at-rest` est PORTÉE mais pas exigée (mesure : elle n'existe que sur les
 * Spike), et un lien hors bornes est ignoré plutôt que de poser un effet au hasard.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { EXPLOSION_MS } from './layers/explosionFx'
import {
  buildGrenadeRestFx,
  DYNAMO_RANK,
  explosionTintOf,
  GRENADE_REST_HOLD_MS,
  grenadeThrowActive,
  restKindOf,
} from './grenadeFx'
import { testReplayDoc } from './test/testDoc'

function docWith(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    projectiles: [
      // Vol certifié : repos en (8, 9) à la frame 10 + 14 = 24.
      { t0: 10, p: [[0, 2, 3], [7, 5, 6], [14, 8, 9]], rest: true },
      // Réplication interrompue SANS repos certifié (le cas des frag/plasma/Dynamo, qui
      // détonent) : l'effet se pose quand même, au dernier point répliqué.
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
  it('pose l effet au DERNIER pas du projectile lié, à sa frame d arrêt', () => {
    const fx = buildGrenadeRestFx(docWith())
    expect(fx).toHaveLength(2)
    expect(fx[0]).toMatchObject({ frame: 24, x: 8, y: 9, rank: DYNAMO_RANK, rest: true })
  })

  it('un vol NON certifié produit AUSSI son effet (frag/plasma/Dynamo détonent sans at-rest), marqué non certifié', () => {
    const fx = buildGrenadeRestFx(docWith())
    expect(fx[1]).toMatchObject({ frame: 35, x: 2, y: 2, rank: 0, rest: false })
  })

  it('un lien hors bornes est ignoré — jamais un effet posé au hasard', () => {
    const doc = docWith({
      grenades: [{ t: 10, slot: 0, i: 1, x: 2, y: 3, rank: 1, s: 'projectile', proj: 9 }],
    })
    expect(buildGrenadeRestFx(doc)).toHaveLength(0)
  })
})

describe('grenadeThrowActive', () => {
  const doc = docWith()

  it('rend le lancer du joueur dans la fenêtre, par son index de FILM (auteur écrit)', () => {
    expect(grenadeThrowActive(doc, 1, 12, 14)).toEqual({ rank: DYNAMO_RANK, age: 2 })
  })

  it('hors fenêtre ou avant le lancer : rien', () => {
    expect(grenadeThrowActive(doc, 1, 9, 14)).toBeNull()
    expect(grenadeThrowActive(doc, 1, 25, 14)).toBeNull()
  })

  it("l'index d'un autre joueur ne déclenche rien — l'auteur n'est pas deviné", () => {
    expect(grenadeThrowActive(doc, 7, 12, 14)).toBeNull()
  })
})

describe('ce que fait une grenade au bout de son vol', () => {
  it('la Dynamo n EXPLOSE PAS : elle pose sa nappe électrique (règle du lot 2.3)', () => {
    expect(restKindOf(DYNAMO_RANK)).toBe('nappe')
  })

  it('Frag, Plasma et Spike détonent', () => {
    expect(restKindOf(0)).toBe('explosion')
    expect(restKindOf(1)).toBe('explosion')
    expect(restKindOf(3)).toBe('explosion')
  })

  it('un rang inconnu garde le halo discret — jamais l effet d un type voisin', () => {
    expect(restKindOf(9)).toBe('halo')
    expect(restKindOf(-1)).toBe('halo')
  })

  it('la teinte dit CE QUI explose : chimique, plasma, cristal', () => {
    expect(explosionTintOf(0)).toBe('blast')
    expect(explosionTintOf(1)).toBe('plasma_cool')
    expect(explosionTintOf(3)).toBe('needle')
  })

  it('la fenêtre de rémanence VAUT la timeline de l explosion — sinon elle la coupe', () => {
    // INVARIANT, pas coïncidence : `drawGrenadeRestLayer` sort dès `age > hold`. Une fenêtre
    // plus courte que `EXPLOSION_MS` amputerait chaque détonation de sa fin — exactement le
    // défaut signalé à la planche du 16/08 (« trop bref »), déplacé d'un cran.
    expect(GRENADE_REST_HOLD_MS).toBe(EXPLOSION_MS)
  })
})
