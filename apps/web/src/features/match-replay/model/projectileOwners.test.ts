/**
 * Tests — projectileOwners : à qui appartient chaque projectile du film.
 *
 * Seul et Masquer retirent les projectiles et les grenades du joueur concerné : il faut donc dire,
 * pour chaque projectile, quelle vie l'a lancé. Une grenade porte son auteur (le film l'écrit) ;
 * un projectile de tir le tient de son tir ; un projectile sans lien n'a pas de propriétaire, et
 * rien n'en est deviné.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { projectileOwners } from './projectileOwners'

const VOL = (t0: number) => ({ t0, p: [[0, 0, 0], [5, 1, 1]], detoT: t0 + 5, detoX: 1, detoY: 1 })

describe('projectileOwners', () => {
  it('une grenade liée à son projectile : le slot et l’image du lancer', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      projectiles: [VOL(10), VOL(30)],
      grenades: [
        { t: 10, slot: 3, i: 1, x: 0, y: 0, rank: 0, s: 'projectile', proj: 0 },
        { t: 30, slot: 4, i: 2, x: 0, y: 0, rank: 1, s: 'projectile', proj: 1 },
      ],
    })
    expect(projectileOwners(doc)).toEqual([{ slot: 3, frame: 10 }, { slot: 4, frame: 30 }])
  })

  it('un projectile de tir : le slot et l’image de son tir', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      shots: [{ slot: 7, t: 20, x: 0, y: 0, w: '0xW' }],
      projectiles: [{ ...VOL(20), shot: 0 }],
    })
    expect(projectileOwners(doc)).toEqual([{ slot: 7, frame: 20 }])
  })

  it('un projectile sans lien, ou dont le lien sort des bornes : pas de propriétaire', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      projectiles: [VOL(10), { ...VOL(20), shot: 9 }],
      grenades: [{ t: 10, slot: 3, i: 1, x: 0, y: 0, rank: 0, s: 'biped' }],
    })
    expect(projectileOwners(doc)).toEqual([null, null])
  })

  it('la grenade prime sur le tir quand les deux lient le même projectile', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      shots: [{ slot: 7, t: 20, x: 0, y: 0 }],
      projectiles: [{ ...VOL(20), shot: 0 }],
      grenades: [{ t: 20, slot: 3, i: 1, x: 0, y: 0, rank: 0, s: 'projectile', proj: 0 }],
    })
    expect(projectileOwners(doc)[0]).toEqual({ slot: 3, frame: 20 })
  })
})
