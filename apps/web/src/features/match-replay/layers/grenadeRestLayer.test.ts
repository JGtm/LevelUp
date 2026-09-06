/**
 * Tests — la fin de vol d'une grenade, et surtout LA NAPPE DYNAMO (variante 2, A4/R2-4).
 *
 * Ce qu'ils tiennent :
 *  1. LA DURÉE VISIBLE. La fenêtre déclarée ne dit pas ce que l'œil voit : l'opacité doit
 *     rester lisible AU MOINS 2,5 s (demande utilisateur du 2026-08-18), pas s'éteindre en
 *     chemin comme le faisait la droite `1 - âge/fenêtre`.
 *  2. LA GRAPHIE. Variante 2 = une nappe DIFFUSE sans anneau. L'anneau net de l'ancienne
 *     version ne doit plus être émis.
 *  3. LE DÉTERMINISME. Le même âge rend la même image — sinon un retour en arrière ne rejoue
 *     pas ce qu'on a vu.
 *  4. LA TEINTE. `--replay-fx-electric` du thème, jamais la couleur des lancers.
 */
import { describe, expect, it } from 'vitest'

import { DYNAMO_RANK, DYNAMO_REST_HOLD_MS, type GrenadeRestFx } from '../model/grenadeFx'
import { drawGrenadeRestLayer, dynamoAlpha, type GrenadeRestStyle } from './grenadeRestLayer'
import { recordingContext, type CanvasOp } from '../test/recordingContext'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 20, maxY: 20 }, width: 400, height: 400, pad: 10 }
const FRAME_MS = 100
const HOLD = Math.round(DYNAMO_REST_HOLD_MS / FRAME_MS)

const STYLE: GrenadeRestStyle = {
  ink: {
    tint: {
      kinetic: '#111', plasma_cool: '#222', plasma_hot: '#333', forerunner: '#444',
      electric: '#0ff', needle: '#555', blast: '#666', neutral: '#777',
    },
    core: '#fff',
  },
  smoke: '#888',
  halo: '#48f',
  k: 1,
  reducedMotion: false,
}

function dynamo(over: Partial<GrenadeRestFx> = {}): GrenadeRestFx {
  return { frame: 0, x: 10, y: 10, rank: DYNAMO_RANK, rest: false, seed: 4242, ...over }
}

/** trace rend les primitives émises pour la nappe à cette image. */
function trace(frame: number, style: Partial<GrenadeRestStyle> = {}, fx = dynamo()): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawGrenadeRestLayer(
    ctx, [fx], VIEW,
    { frame, holdHalo: 24, holdDynamo: HOLD, frameMs: FRAME_MS },
    { ...STYLE, ...style },
  )
  return ops
}

/** L'opacité MAXIMALE d'une primitive de tracé — ce que l'œil a de plus visible. */
function opaciteMax(ops: CanvasOp[]): number {
  let alpha = 1
  let max = 0
  for (const o of ops) {
    if (o.op === 'set globalAlpha') alpha = Number(o.args[0])
    if (o.op === 'stroke' || o.op === 'fill') max = Math.max(max, alpha)
  }
  return max
}

describe('nappe Dynamo — la durée que l’œil voit (A4, 2026-08-18)', () => {
  it('reste lisible AU MOINS 2,5 s au sol', () => {
    // Convention d'écran, écrite ici : en dessous de 0,15 un trait de 1 px ne se distingue
    // plus d'un fond de carte. La demande était « au moins 2,5 s ».
    const SEUIL = 0.15
    let dernierVisible = -1
    for (let f = 0; f <= HOLD; f++) {
      if (opaciteMax(trace(f)) >= SEUIL) dernierVisible = f * FRAME_MS
    }
    expect(dernierVisible).toBeGreaterThanOrEqual(2_500)
  })

  it('la courbe est un PLATEAU puis une chute, pas une droite', () => {
    expect(dynamoAlpha(0, 3_000)).toBe(1)
    expect(dynamoAlpha(1_500, 3_000)).toBe(1)
    expect(dynamoAlpha(2_250, 3_000)).toBe(1)
    expect(dynamoAlpha(2_625, 3_000)).toBeCloseTo(0.5, 6)
    expect(dynamoAlpha(3_000, 3_000)).toBe(0)
    // Une droite serait déjà à 0,25 là où le plateau vaut encore 1.
    expect(dynamoAlpha(2_250, 3_000)).toBeGreaterThan(1 - 2_250 / 3_000)
  })

  it('la fenêtre borne le dessin : passé la fenêtre, plus rien', () => {
    expect(trace(HOLD).some((o) => o.op === 'stroke')).toBe(true)
    // Le calque remet l'opacité à 1 en sortant : aucune PRIMITIVE de tracé au-delà.
    expect(trace(HOLD + 1).some((o) => o.op === 'stroke' || o.op === 'fill')).toBe(false)
  })
})

describe('nappe Dynamo — la graphie de la variante 2', () => {
  it('n’émet AUCUN anneau : la variante 2 est une nappe diffuse', () => {
    const ops = trace(5)
    // L'ancienne version traçait un cercle complet PUIS le remplissait ou le cernait. Ici le
    // seul `arc` est celui du dégradé (rempli), jamais suivi d'un `stroke`.
    const arcs = ops.map((o, i) => ({ o, i })).filter((e) => e.o.op === 'arc')
    expect(arcs).toHaveLength(1)
    const suivant = ops.slice(arcs[0].i + 1).find((o) => o.op === 'stroke' || o.op === 'fill')
    expect(suivant?.op).toBe('fill')
  })

  it('pose une nappe en dégradé radial, et neuf arcs qui rebondissent', () => {
    const ops = trace(5)
    expect(ops.filter((o) => o.op === 'createRadialGradient')).toHaveLength(1)
    // Un arc = un `moveTo` puis DEUX `lineTo` (bord -> creux -> bord).
    expect(ops.filter((o) => o.op === 'moveTo')).toHaveLength(9)
    expect(ops.filter((o) => o.op === 'lineTo')).toHaveLength(18)
  })

  it('prend la teinte ÉLECTRIQUE du thème, jamais la couleur des lancers', () => {
    const encres = trace(5).filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toContain('#0ff')
    expect(encres).not.toContain('#48f')
  })

  it('thème sans teinte électrique : repli sur l’encre du halo, jamais une couleur inventée', () => {
    const style = { ink: { ...STYLE.ink, tint: { ...STYLE.ink.tint, electric: '' } } }
    const encres = trace(5, style).filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toContain('#48f')
  })
})

describe('nappe Dynamo — déterminisme et mouvement réduit', () => {
  // Les jetons de dégradé sont des objets neufs à chaque appel : on compare la TRACE écrite
  // (nom de primitive + arguments), pas des identités d'objets.
  const empreinte = (ops: CanvasOp[]) =>
    ops.map((o) => `${o.op}(${o.args.map((a) => (typeof a === 'object' ? '~' : a)).join(',')})`)

  it('le même âge rend la MÊME image', () => {
    expect(empreinte(trace(12))).toEqual(empreinte(trace(12)))
  })

  it('deux Dynamos de germes différents ne portent pas les mêmes arcs', () => {
    const a = empreinte(trace(5, {}, dynamo({ seed: 1 })))
    const b = empreinte(trace(5, {}, dynamo({ seed: 999 })))
    expect(a).not.toEqual(b)
  })

  it('sous mouvement réduit, les arcs se FIGENT — la nappe ne disparaît pas', () => {
    const style = { reducedMotion: true }
    const geo = (ops: CanvasOp[]) =>
      ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo').map((o) => o.args)
    expect(geo(trace(3, style))).toEqual(geo(trace(19, style)))
    expect(trace(19, style).length).toBeGreaterThan(0)
  })
})
