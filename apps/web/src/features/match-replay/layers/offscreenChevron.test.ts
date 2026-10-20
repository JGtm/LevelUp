/**
 * Tests — offscreenChevron (le gabarit de la flèche hors cadre, plan escouade hors cadre B1).
 *
 * Ce que ces tests verrouillent :
 *  - le TRACÉ ferme exactement 4 sommets, dans l'ordre du gabarit fourni par l'utilisateur
 *    (pointe, arrière-gauche, encoche, arrière-droite — la base CONCAVE) ;
 *  - l'ÉCHELLE est celle de l'ÉCRAN (`k`), jamais du canevas — `ctx.scale` porte `k`, pas 1 ;
 *  - le PIVOT est `angle`, tel quel ;
 *  - le REMPLISSAGE est la couleur passée, jamais une couleur d'ici ;
 *  - l'ÉTIQUETTE se pose du côté INTÉRIEUR de la flèche (à l'opposé de `angle`, jamais dans le
 *    prolongement qui la sortirait de la toile) et son encre de contour est celle du thème
 *    passée par l'appelant, jamais un littéral.
 */
import { describe, expect, it } from 'vitest'

import {
  CHEVRON_SIZE_PX,
  drawOffscreenChevron,
  drawOffscreenLabel,
  offscreenLabelAnchor,
} from './offscreenChevron'
import { recordingContext } from '../test/recordingContext'

describe('drawOffscreenChevron — le gabarit, à l’échelle de l’écran, pivoté, coloré', () => {
  it('ferme exactement 4 sommets, dans l’ordre du gabarit (base concave)', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenChevron(ctx, { x: 10, y: 20 }, 0, 1, 'rouge')
    const moveTo = ops.find((o) => o.op === 'moveTo')
    const lineTo = ops.filter((o) => o.op === 'lineTo')
    expect(moveTo?.args).toEqual([1, 0]) // pointe
    expect(lineTo.map((o) => o.args)).toEqual([
      [-0.75, -0.7], // arrière-gauche
      [-0.35, 0], // encoche
      [-0.75, 0.7], // arrière-droite
    ])
    expect(ops.some((o) => o.op === 'closePath')).toBe(true)
  })

  it('translate au point servi, pivote de `angle`, met à l’échelle de l’ÉCRAN (k), jamais du canevas', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenChevron(ctx, { x: 10, y: 20 }, Math.PI / 3, 2, 'rouge')
    expect(ops.find((o) => o.op === 'translate')?.args).toEqual([10, 20])
    expect(ops.find((o) => o.op === 'rotate')?.args).toEqual([Math.PI / 3])
    // La mise à l'échelle porte `k` (2 ici) : un gabarit déclaré à `CHEVRON_SIZE_PX` en
    // pixels d'écran, jamais un facteur de densité du canevas laissé de côté.
    expect(ops.find((o) => o.op === 'scale')?.args).toEqual([CHEVRON_SIZE_PX * 2, CHEVRON_SIZE_PX * 2])
  })

  it('est rempli de la couleur passée, jamais une couleur d’ici', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenChevron(ctx, { x: 0, y: 0 }, 0, 1, 'un-token-resolu')
    const fillStyleSets = ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])
    expect(fillStyleSets).toContain('un-token-resolu')
    expect(ops.some((o) => o.op === 'fill')).toBe(true)
  })

  it('encadre son geste de save/restore : l’état ne fuit pas vers les calques suivants', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenChevron(ctx, { x: 0, y: 0 }, 0, 1, 'rouge')
    expect(ops[0].op).toBe('save')
    expect(ops[ops.length - 1].op).toBe('restore')
  })
})

describe('offscreenLabelAnchor — le côté INTÉRIEUR de la flèche', () => {
  it('flèche pointant à DROITE (angle 0) : l’étiquette se pose à GAUCHE du repère', () => {
    const at = { x: 50, y: 50 }
    const anchor = offscreenLabelAnchor(at, 0, 1)
    expect(anchor.x).toBeLessThan(at.x)
    expect(anchor.y).toBeCloseTo(at.y, 6)
  })

  it('flèche pointant vers le BAS (angle PI/2, canevas +Y bas) : l’étiquette se pose au-DESSUS', () => {
    const at = { x: 50, y: 50 }
    const anchor = offscreenLabelAnchor(at, Math.PI / 2, 1)
    expect(anchor.y).toBeLessThan(at.y)
    expect(anchor.x).toBeCloseTo(at.x, 6)
  })

  it('l’écart grandit avec `k` (échelle de l’écran)', () => {
    const at = { x: 50, y: 50 }
    const a1 = offscreenLabelAnchor(at, 0, 1)
    const a2 = offscreenLabelAnchor(at, 0, 3)
    const gap1 = at.x - a1.x
    const gap2 = at.x - a2.x
    expect(gap2).toBeCloseTo(gap1 * 3, 6)
  })
})

describe('drawOffscreenLabel — le texte, à l’encre du thème passée par l’appelant', () => {
  it('écrit le texte au double trait (contour puis remplissage), centré sur l’ancre intérieure', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenLabel(ctx, { x: 50, y: 50 }, 0, 'Nilton410 · 42 m', { k: 1, labelStroke: 'encre-sombre' }, 'encre-equipe')
    const anchor = offscreenLabelAnchor({ x: 50, y: 50 }, 0, 1)
    const stroke = ops.find((o) => o.op === 'strokeText')
    const fill = ops.find((o) => o.op === 'fillText')
    expect(stroke?.args).toEqual(['Nilton410 · 42 m', anchor.x, anchor.y])
    expect(fill?.args).toEqual(['Nilton410 · 42 m', anchor.x, anchor.y])
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toContain('encre-sombre')
    expect(ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])).toContain('encre-equipe')
  })

  it('`labelStroke` vide (variable de thème absente) : AUCUN strokeText — jamais un contour inventé', () => {
    const { ops, ctx } = recordingContext()
    drawOffscreenLabel(ctx, { x: 50, y: 50 }, 0, 'X', { k: 1, labelStroke: '' }, 'encre-equipe')
    expect(ops.some((o) => o.op === 'strokeText')).toBe(false)
    expect(ops.some((o) => o.op === 'fillText')).toBe(true)
  })
})
