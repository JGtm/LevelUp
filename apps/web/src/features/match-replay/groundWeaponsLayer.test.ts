/**
 * Tests — groundWeaponsLayer : CE QUI EST POSÉ, OÙ, ET CE QUI NE L'EST JAMAIS.
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - la vignette est posée à la POSITION DE REPOS projetée, et nulle part ailleurs ;
 *  - l'opacité SUIT la mesure (pleine tant qu'une preuve tient, descendante ensuite) ;
 *  - une arme hors fenêtre n'émet RIEN ;
 *  - une famille SANS vignette n'émet rien non plus : ni glyphe de repli, ni icône voisine ;
 *  - il n'y a NI losange NI texte — la grammaire du socle appartient au socle.
 *
 * La lecture temporelle est testée à part (`groundWeaponTime.test.ts`) : ce fichier ne
 * vérifie que ce que le pinceau émet.
 *
 * Le contexte enregistreur observe la GÉOMÉTRIE ÉMISE, jamais un pixel (cf. recordingContext).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayGroundWeapon } from '@/lib/api/types'

import { count, diamondCentres, recordingContext, valuesOf } from './test/recordingContext'
import { GROUND_WEAPON_ALPHA_FULL } from './groundWeaponTime'
import { drawGroundWeaponsLayer, type GroundWeaponStyle } from './groundWeaponsLayer'
import { worldToCanvas } from './replayLogic'

/** 10 m de côté sur 100 px : 10 px par mètre — le même cadrage que les tests de socles. */
const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }

const IMAGE = { width: 40, height: 16 } as unknown as CanvasImageSource
const ICON = { fill: IMAGE, outline: IMAGE }

function item(over: Partial<ReplayGroundWeapon> = {}): ReplayGroundWeapon {
  return {
    t0: 0,
    t1: 100,
    t1max: 120,
    x: 5,
    y: 5,
    w: '0a1992bc',
    origin: 'dropped',
    dropper: 3,
    end: 'seen',
    picker: -1,
    ...over,
  }
}

function draw(
  items: ReplayGroundWeapon[],
  frame: number,
  over: Partial<GroundWeaponStyle> = {},
) {
  const { ops, ctx } = recordingContext()
  drawGroundWeaponsLayer(ctx, items, VIEW, { frame, k: 1 }, { iconOf: () => ICON, ...over })
  return ops
}

/** Les coins hauts-gauches des images posées, dans l'ordre d'émission. */
const imageCorners = (ops: ReturnType<typeof draw>) =>
  ops.filter((o) => o.op === 'drawImage').map((o) => ({ x: o.args[1] as number, y: o.args[2] as number }))

describe('drawGroundWeaponsLayer — la vignette et sa place', () => {
  it('pose la vignette CENTRÉE sur la position de repos projetée', () => {
    const ops = draw([item({ x: 2, y: 8 })], 10)
    const attendu = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    // Le CORPS est la dernière image posée (le liseré la précède, tout autour) : son centre
    // est le point mesuré, aux demi-dimensions près.
    const coins = imageCorners(ops)
    const corps = coins[coins.length - 1]
    const last = ops.filter((o) => o.op === 'drawImage').at(-1)!
    const w = last.args[3] as number
    const h = last.args[4] as number
    expect(corps.x + w / 2).toBeCloseTo(attendu.x, 6)
    expect(corps.y + h / 2).toBeCloseTo(attendu.y, 6)
  })

  it('cerne la vignette : le liseré est la même forme reposée tout autour', () => {
    const ops = draw([item()], 10)
    // Un corps + huit dépôts de silhouette : sans eux, une arme au sol se perd dans le gris
    // des cartes reconstruites.
    expect(count(ops, 'drawImage')).toBe(9)
  })

  it('n’emprunte NI le losange NI le texte du calque des socles', () => {
    const ops = draw([item()], 10)
    expect(diamondCentres(ops)).toHaveLength(0)
    expect(count(ops, 'fillText')).toBe(0)
    expect(count(ops, 'strokeText')).toBe(0)
    expect(count(ops, 'arc')).toBe(0)
  })
})

describe('drawGroundWeaponsLayer — l’opacité SUIT la mesure', () => {
  it('peint au plein tant qu’une preuve de présence tient', () => {
    const alphas = valuesOf(draw([item()], 50), 'globalAlpha')
    expect(alphas).toContain(GROUND_WEAPON_ALPHA_FULL)
  })

  it('peint plus faible dans l’intervalle où la disparition n’est pas datée', () => {
    const dansIntervalle = valuesOf(draw([item()], 110), 'globalAlpha')[0]
    expect(dansIntervalle).toBeLessThan(GROUND_WEAPON_ALPHA_FULL)
    expect(dansIntervalle).toBeGreaterThan(0)
  })

  it('n’émet RIEN passé la première preuve d’absence', () => {
    expect(count(draw([item()], 121), 'drawImage')).toBe(0)
  })

  it('n’émet RIEN avant l’apparition', () => {
    expect(count(draw([item({ t0: 30 })], 29), 'drawImage')).toBe(0)
  })
})

describe('drawGroundWeaponsLayer — ce qu’on refuse d’inventer', () => {
  it('sans vignette, ne dessine RIEN : ni glyphe de repli, ni icône voisine', () => {
    const ops = draw([item()], 10, { iconOf: () => null })
    expect(count(ops, 'drawImage')).toBe(0)
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(0)
  })

  it('ne dessine que les armes visibles quand plusieurs se partagent le terrain', () => {
    const items = [
      item({ x: 1, y: 1, t0: 0, t1: 10, t1max: 10, end: 'pickup' }),
      item({ x: 9, y: 9, t0: 0, t1: 200, t1max: 200, end: 'open' }),
    ]
    const coins = imageCorners(draw(items, 50))
    // Neuf dépôts pour la seule arme encore au sol — la ramassée a disparu à l'image 11.
    expect(coins).toHaveLength(9)
    const attendu = worldToCanvas({ x: 9, y: 9 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const dernier = coins[coins.length - 1]
    expect(dernier.x).toBeGreaterThan(attendu.x - 20)
    expect(dernier.y).toBeGreaterThan(attendu.y - 20)
  })

  it('ne peint rien sur une liste vide ni sur un cadrage dégénéré', () => {
    expect(count(draw([], 0), 'drawImage')).toBe(0)
    const { ops, ctx } = recordingContext()
    drawGroundWeaponsLayer(ctx, [item()], { ...VIEW, width: 0 }, { frame: 0, k: 1 }, {
      iconOf: () => ICON,
    })
    expect(ops).toHaveLength(0)
  })
})
