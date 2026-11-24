import { describe, expect, it } from 'vitest'

import { metersToGameUnits } from '@/lib/formatters/distance'

import { peindreLaScene } from './coequipiersPaint'
import { SCENE_VIDE } from './mort.logic'

/** Une toile qui note le rayon de chaque arc tracé. */
function toileQuiNote() {
  const rayons: number[] = []
  const ctx = new Proxy(
    {},
    {
      get: (_t, nom) => {
        if (nom === 'arc') return (_x: number, _y: number, r: number) => void rayons.push(r)
        return () => undefined
      },
      set: () => true,
    },
  ) as unknown as CanvasRenderingContext2D
  return { ctx, rayons }
}

describe('le cercle du rayon du radar est dessiné en unités du jeu', () => {
  it('18 m de radar = 5,9 unités du jeu, pas 18 unités', () => {
    const { ctx, rayons } = toileQuiNote()
    const scene = { ...SCENE_VIDE, cercle: { centre: { x: 0, y: 0 }, rayonM: 18 } }
    const couleurs = { encre: '#000', fond: '#fff', allie: '#0f0', adverse: '#f00' }
    const dim = { epaisseurTrait: 1, rayonMarque: 1 } as unknown as Parameters<typeof peindreLaScene>[5]
    peindreLaScene(ctx, scene, (p) => p, 10, couleurs, dim)
    // 10 px par unité du jeu : un cercle de 18 m couvre 5,9 unités, soit ~59 px (et non 180).
    expect(rayons[0]).toBeCloseTo(metersToGameUnits(18) * 10, 6)
    expect(rayons[0]).toBeLessThan(60)
  })
})
