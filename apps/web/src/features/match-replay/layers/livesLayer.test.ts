/**
 * livesLayer.test.ts — LE DESSIN DES TRACES DES VIES, au contexte enregistreur.
 *
 * Ce qu'ils protègent : (1) la règle de forme, « ce qui vit est plein, une trace est creuse » :
 * aucun remplissage, que des traits, et des tailles sous celles du pion ; (2) la goutte pointe dans
 * le sens du cap (et suit la rotation de la carte), un anneau seul sans cap lu ; (3) la croix à
 * la position de la victime, le triangle au tueur et son trait vers la victime ; (4) le filtre
 * « qui / quand » et le fondu ; (5) aucune couleur écrite en dur : les seules encres sont l'équipe
 * de la marque et le liseré passé ; (6) l'opacité est rendue à 1 en sortie.
 */
import { describe, expect, it } from 'vitest'

import { CORE_RADIUS } from '@/lib/replay/screenSizes'

import type { DeathMark, KillerMark, LivesTime, SpawnMark } from '../model/livesMarks'
import { LIVES_FADE_FLOOR } from '../model/livesMarks'
import type { CanvasView } from '../model/replayView'
import { count, recordingContext, valuesOf, type CanvasOp } from '../test/recordingContext'
import { drawDeathMarks, drawKillerMarks, drawSpawnMarks, type LivesPaint } from './livesLayer'

/** 4 px par unité monde, Y inversé : le monde (25, 25) est en (100, 300). */
const VIEW: CanvasView = {
  bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 } as CanvasView['bounds'],
  width: 400,
  height: 400,
  pad: 0,
}

const ALL: LivesTime = { when: 'all', frame: 0, frameMs: 100 }

function paint(over: Partial<LivesPaint> = {}): LivesPaint {
  return { who: 'all', time: ALL, underlay: 'under-ink', k: 1, ...over }
}

function spawn(over: Partial<SpawnMark> = {}): SpawnMark {
  return { frame: 0, endFrame: 100, x: 25, y: 25, headingDeg: 0, color: 'team-ink', mine: true, followed: false, ...over }
}

function death(over: Partial<DeathMark> = {}): DeathMark {
  return { frame: 0, x: 25, y: 25, color: 'victim-ink', mine: true, followed: false, ...over }
}

function killer(over: Partial<KillerMark> = {}): KillerMark {
  return { frame: 0, x: 25, y: 25, victim: { x: 75, y: 25 }, color: 'killer-ink', mine: true, followed: false, ...over }
}

/** Les couleurs de trait posées, dans l'ordre. */
const strokes = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])

describe('drawSpawnMarks — la goutte creuse des apparitions', () => {
  it('une goutte pointe dans le sens du cap : cap 0 vers la droite de l’écran, cap 90 vers le haut', () => {
    const est = recordingContext()
    drawSpawnMarks(est.ctx, [spawn({ headingDeg: 0 })], VIEW, paint())
    const pointeEst = est.ops.find((o) => o.op === 'moveTo')!.args as number[]
    // Centre (100, 300) ; la pointe est à droite, à la même hauteur.
    expect(pointeEst[0]).toBeGreaterThan(100)
    expect(pointeEst[1]).toBeCloseTo(300, 6)

    const nord = recordingContext()
    drawSpawnMarks(nord.ctx, [spawn({ headingDeg: 90 })], VIEW, paint())
    const pointeNord = nord.ops.find((o) => o.op === 'moveTo')!.args as number[]
    expect(pointeNord[0]).toBeCloseTo(100, 6)
    expect(pointeNord[1]).toBeLessThan(300)
  })

  it('sans cap lu : un anneau seul, aucune pointe', () => {
    const { ctx, ops } = recordingContext()
    drawSpawnMarks(ctx, [spawn({ headingDeg: null })], VIEW, paint())
    // Un seul tracé, passé deux fois (liseré, puis couleur).
    expect(count(ops, 'arc')).toBe(1)
    expect(count(ops, 'stroke')).toBe(2)
    expect(count(ops, 'moveTo')).toBe(0)
    expect(count(ops, 'lineTo')).toBe(0)
  })

  it('RÈGLE DE FORME : jamais de remplissage, et plus petite que le pion', () => {
    const { ctx, ops } = recordingContext()
    drawSpawnMarks(ctx, [spawn({ headingDeg: null }), spawn({ headingDeg: 45 })], VIEW, paint())
    expect(count(ops, 'fill')).toBe(0)
    const rayons = ops.filter((o) => o.op === 'arc').map((o) => o.args[2] as number)
    for (const r of rayons) expect(r).toBeLessThan(CORE_RADIUS)
  })

  it('suit la rotation de la carte : à un quart de tour, la pointe d’un cap 0 descend', () => {
    // Le centre de la scène ne bouge pas à la rotation : la marque y est posée.
    const tourne: CanvasView = { ...VIEW, orient: { turns: 1, pivot: { x: 50, y: 50 } } }
    const { ctx, ops } = recordingContext()
    drawSpawnMarks(ctx, [spawn({ x: 50, y: 50, headingDeg: 0 })], tourne, paint())
    const pointe = ops.find((o) => o.op === 'moveTo')!.args as number[]
    expect(pointe[0]).toBeCloseTo(200, 6)
    expect(pointe[1]).toBeGreaterThan(200)
  })
})

describe('drawDeathMarks — la croix des morts', () => {
  it('une croix : deux segments croisés à la position de la victime, sans remplissage', () => {
    const { ctx, ops } = recordingContext()
    drawDeathMarks(ctx, [death()], VIEW, paint())
    // Deux passes (liseré, couleur) du même tracé : deux `stroke`, un seul tracé de deux segments.
    expect(count(ops, 'stroke')).toBe(2)
    expect(count(ops, 'moveTo')).toBe(2)
    expect(count(ops, 'lineTo')).toBe(2)
    expect(count(ops, 'fill')).toBe(0)
    const [m1, m2] = ops.filter((o) => o.op === 'moveTo').map((o) => o.args as number[])
    const [l1, l2] = ops.filter((o) => o.op === 'lineTo').map((o) => o.args as number[])
    // Les deux segments se croisent au centre (100, 300), en diagonale.
    expect([(m1[0] + l1[0]) / 2, (m1[1] + l1[1]) / 2]).toEqual([100, 300])
    expect([(m2[0] + l2[0]) / 2, (m2[1] + l2[1]) / 2]).toEqual([100, 300])
    expect(m2[0] - 100).toBeCloseTo(-(m1[0] - 100), 9)
  })

  it('porte la couleur de la victime et le liseré, rien d’autre', () => {
    const { ctx, ops } = recordingContext()
    drawDeathMarks(ctx, [death()], VIEW, paint())
    expect(new Set(strokes(ops))).toEqual(new Set(['under-ink', 'victim-ink']))
  })
})

describe('drawKillerMarks — le triangle creux et son trait', () => {
  it('un triangle de trois sommets fermé, et un trait vers la victime quand elle est positionnée', () => {
    const { ctx, ops } = recordingContext()
    drawKillerMarks(ctx, [killer()], VIEW, paint())
    expect(count(ops, 'closePath')).toBe(1)
    expect(count(ops, 'stroke')).toBe(4) // le trait puis le triangle, chacun en deux passes
    // Le trait : un moveTo au tueur (100, 300), un lineTo à la victime (300, 300).
    const premier = ops.find((o) => o.op === 'moveTo')!.args as number[]
    const premierLine = ops.find((o) => o.op === 'lineTo')!.args as number[]
    expect(premier).toEqual([100, 300])
    expect(premierLine).toEqual([300, 300])
    expect(count(ops, 'fill')).toBe(0)
  })

  it('sans victime positionnée : le triangle seul, pointe en haut, sans trait', () => {
    const { ctx, ops } = recordingContext()
    drawKillerMarks(ctx, [killer({ victim: null })], VIEW, paint())
    const sommets = ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
    expect(sommets).toHaveLength(3)
    const premier = sommets[0].args as number[]
    expect(premier[0]).toBeCloseTo(100, 6)
    expect(premier[1]).toBeLessThan(300)
  })

  it('tueur et victime au même endroit à l’écran : pas de trait', () => {
    const { ctx, ops } = recordingContext()
    drawKillerMarks(ctx, [killer({ victim: { x: 25, y: 25 } })], VIEW, paint())
    expect(ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo')).toHaveLength(3)
  })

  it('porte la couleur d’équipe du tueur', () => {
    const { ctx, ops } = recordingContext()
    drawKillerMarks(ctx, [killer()], VIEW, paint())
    expect(new Set(strokes(ops))).toEqual(new Set(['under-ink', 'killer-ink']))
  })
})

describe('le filtre qui / quand et le fondu, vus du dessin', () => {
  it('« mon équipe » ne peint pas les marques adverses, « joueur suivi » que celles du suivi', () => {
    const marks = [death({ mine: true, followed: true }), death({ mine: false, followed: false, x: 75 })]
    const equipe = recordingContext()
    drawDeathMarks(equipe.ctx, marks, VIEW, paint({ who: 'team' }))
    expect(count(equipe.ops, 'moveTo')).toBe(2)
    const suivi = recordingContext()
    drawDeathMarks(suivi.ctx, [death({ followed: false }), ...marks], VIEW, paint({ who: 'followed' }))
    expect(count(suivi.ops, 'moveTo')).toBe(2)
    const tous = recordingContext()
    drawDeathMarks(tous.ctx, marks, VIEW, paint())
    expect(count(tous.ops, 'moveTo')).toBe(4)
  })

  it('jusqu’ici : une mort à venir n’est pas peinte, une ancienne est plus pâle que la récente', () => {
    const time = (frame: number): LivesTime => ({ when: 'sofar', frame, frameMs: 100 })
    const futur = recordingContext()
    drawDeathMarks(futur.ctx, [death({ frame: 50 })], VIEW, paint({ time: time(10) }))
    expect(count(futur.ops, 'moveTo')).toBe(0)
    const recente = recordingContext()
    drawDeathMarks(recente.ctx, [death({ frame: 10 })], VIEW, paint({ time: time(10) }))
    const ancienne = recordingContext()
    drawDeathMarks(ancienne.ctx, [death({ frame: 0 })], VIEW, paint({ time: time(100_000) }))
    // Hors la remise à 1 finale : la récente pose 1 pour la couleur, l'ancienne le plancher.
    const alphaRecente = valuesOf(recente.ops, 'globalAlpha').slice(0, -1)
    const alphaAncienne = valuesOf(ancienne.ops, 'globalAlpha').slice(0, -1)
    expect(Math.max(...alphaRecente)).toBe(1)
    expect(Math.max(...alphaAncienne)).toBeCloseTo(LIVES_FADE_FLOOR, 9)
    expect(Math.min(...alphaAncienne)).toBeGreaterThan(0)
  })

  it('l’apparition d’une vie en cours est pleine en « jusqu’ici », même ancienne', () => {
    const { ctx, ops } = recordingContext()
    const time: LivesTime = { when: 'sofar', frame: 90_000, frameMs: 100 }
    drawSpawnMarks(ctx, [spawn({ frame: 0, endFrame: 100_000 })], VIEW, paint({ time }))
    expect(Math.max(...valuesOf(ops, 'globalAlpha').slice(0, -1))).toBe(1)
  })

  it('rend l’opacité à 1 en sortie, pour les calques qui suivent', () => {
    const { ctx, ops } = recordingContext()
    drawKillerMarks(ctx, [killer()], VIEW, paint({ time: { when: 'sofar', frame: 5, frameMs: 100 } }))
    const alphas = valuesOf(ops, 'globalAlpha')
    expect(alphas[alphas.length - 1]).toBe(1)
  })

  it('les tailles suivent la densité k', () => {
    const un = recordingContext()
    drawSpawnMarks(un.ctx, [spawn({ headingDeg: null })], VIEW, paint({ k: 1 }))
    const deux = recordingContext()
    drawSpawnMarks(deux.ctx, [spawn({ headingDeg: null })], VIEW, paint({ k: 2 }))
    const r = (ops: CanvasOp[]) => ops.find((o) => o.op === 'arc')!.args[2] as number
    expect(r(deux.ops)).toBeCloseTo(2 * r(un.ops), 9)
  })
})
