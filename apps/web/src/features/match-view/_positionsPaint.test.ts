/**
 * _positionsPaint.test.ts — « Occupation du terrain » suit l'orientation du rejeu : le fond, le plan des
 * murs et la chaleur tournent ensemble, un point monde connu tombe au pixel de la carte tournée, et rien
 * n'est tracé pour un plan absent. À 0 quart de tour le tracé est celui d'avant (aucune transformation).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'
import { orientAbout, orientBounds, pivotOf, QUARTER_TURNS, type QuarterTurns } from '@/lib/replay/orientation'
import type { WallPlanReady } from '@/lib/replay/wallPlan'

import type { MapFrame } from './_positionsHeat'
import { fenetreDuMonde, peindreLesPositions, type PeintureDesPositions } from './_positionsPaint'

interface Op {
  op: string
  args: unknown[]
}

/** Un contexte 2D qui note chaque appel et chaque affectation. */
function contexte() {
  const ops: Op[] = []
  const etat: Record<string, unknown> = {}
  const ctx = new Proxy(etat, {
    get(cible, nom: string) {
      if (nom in cible) return cible[nom]
      return (...args: unknown[]) => {
        ops.push({ op: nom, args })
      }
    },
    set(cible, nom: string, valeur: unknown) {
      cible[nom] = valeur
      return true
    },
  })
  return { ctx: ctx as unknown as CanvasRenderingContext2D, ops }
}

// Un cadre de fond de 80 x 40 m : x de -20 à 60, y de 20 à 60.
const FRAME: MapFrame = { originX: -20, originY: 60, widthM: 80, heightM: 40 }
const MONDE: ReplayBounds = { minX: -20, maxX: 60, minY: 20, maxY: 60, minZ: 0, maxZ: 0 }
const PIVOT = pivotOf(MONDE)
const IMAGE = { width: 160, height: 80 } as unknown as CanvasImageSource & { width: number }
const PLAN: WallPlanReady = { edges: [[-20, 60, 60, 60], [0, 30, 0, 50]], dropped: 0 }

/** La toile à l'écran : 400 px de large, au rapport de la carte tournée. */
function toile(turns: QuarterTurns) {
  const scene = orientBounds(MONDE, turns)
  const width = 400
  return { scene, width, height: width * ((scene.maxY - scene.minY) / (scene.maxX - scene.minX)) }
}

function peinture(turns: QuarterTurns, over: Partial<PeintureDesPositions> = {}): PeintureDesPositions {
  return {
    frame: FRAME, image: IMAGE, lissee: null, cell: 4, ramp: ['a', 'b'],
    bounds: toile(turns).scene, turns, pivot: PIVOT, plan: PLAN, ink: 'encre', ...over,
  }
}

/** Applique à un point de la toile virtuelle la transformation posée par le tracé (translate, rotate, translate). */
function surLaToile(ops: Op[], v: { x: number; y: number }): { x: number; y: number } {
  let m = { a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 } // matrice courante : (x, y) -> (a x + c y + e, b x + d y + f)
  for (const o of ops) {
    if (o.op === 'translate') {
      const [tx, ty] = o.args as [number, number]
      m = { ...m, e: m.e + m.a * tx + m.c * ty, f: m.f + m.b * tx + m.d * ty }
    } else if (o.op === 'rotate') {
      const t = o.args[0] as number
      const cos = Math.cos(t)
      const sin = Math.sin(t)
      m = {
        a: m.a * cos + m.c * sin, b: m.b * cos + m.d * sin,
        c: -m.a * sin + m.c * cos, d: -m.b * sin + m.d * cos,
        e: m.e, f: m.f,
      }
    }
  }
  return { x: m.a * v.x + m.c * v.y + m.e, y: m.b * v.x + m.d * v.y + m.f }
}

describe('fenetreDuMonde', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour : la rotation de la fenêtre est défaite', (turns) => {
    const monde = fenetreDuMonde(orientBounds(MONDE, turns), turns, PIVOT)
    expect(monde).toEqual({ ...MONDE })
  })
})

describe('peindreLesPositions', () => {
  it('à 0 quart de tour : le tracé d’avant, sans save ni transformation', () => {
    const { ctx, ops } = contexte()
    const { width, height } = toile(0)
    peindreLesPositions(ctx, width, height, peinture(0))
    expect(ops.filter((o) => ['save', 'translate', 'rotate', 'restore'].includes(o.op))).toHaveLength(
      // Le plan des murs pose son propre save/restore, jamais de translate ni de rotate.
      2,
    )
    expect(ops.find((o) => o.op === 'drawImage')?.args).toEqual([IMAGE, 0, 0, 160, 80, 0, 0, width, height])
  })

  it.each(QUARTER_TURNS)('à %i quart(s) de tour : le plan des murs tombe aux pixels de la carte tournée', (turns) => {
    const { ctx, ops } = contexte()
    const { scene, width, height } = toile(turns)
    peindreLesPositions(ctx, width, height, peinture(turns))
    const k = width / (scene.maxX - scene.minX)
    const segments = ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
    expect(segments).toHaveLength(4)
    PLAN.edges.forEach(([x1, y1, x2, y2], i) => {
      for (const [j, p] of [[0, { x: x1, y: y1 }], [1, { x: x2, y: y2 }]] as const) {
        // L'oracle : tourner le point monde, puis le projeter dans la fenêtre tournée.
        const q = orientAbout(p, turns, PIVOT)
        const attendu = { x: (q.x - scene.minX) * k, y: (scene.maxY - q.y) * k }
        const [vx, vy] = segments[2 * i + j].args as [number, number]
        const got = surLaToile(ops, { x: vx, y: vy })
        expect(got.x).toBeCloseTo(attendu.x, 6)
        expect(got.y).toBeCloseTo(attendu.y, 6)
      }
    })
  })

  it.each(QUARTER_TURNS)('à %i quart(s) de tour : l’image couvre exactement la toile tournée', (turns) => {
    const { ctx, ops } = contexte()
    const { scene, width, height } = toile(turns)
    peindreLesPositions(ctx, width, height, peinture(turns))
    const image = ops.find((o) => o.op === 'drawImage')!
    const [, , , , , dx, dy, dw, dh] = image.args as number[]
    const k = width / (scene.maxX - scene.minX)
    const monde = { nw: { x: -20, y: 60 }, se: { x: 60, y: 20 } }
    for (const [coin, v] of [[monde.nw, { x: dx, y: dy }], [monde.se, { x: dx + dw, y: dy + dh }]] as const) {
      const q = orientAbout(coin, turns, PIVOT)
      const attendu = { x: (q.x - scene.minX) * k, y: (scene.maxY - q.y) * k }
      const got = surLaToile(ops, v)
      expect(got.x).toBeCloseTo(attendu.x, 6)
      expect(got.y).toBeCloseTo(attendu.y, 6)
    }
  })

  it('sans plan (carte sans plan, interrupteur éteint) : aucune arête tracée', () => {
    for (const turns of QUARTER_TURNS) {
      const { ctx, ops } = contexte()
      const { width, height } = toile(turns)
      peindreLesPositions(ctx, width, height, peinture(turns, { plan: null }))
      expect(ops.filter((o) => o.op === 'moveTo' || o.op === 'stroke')).toHaveLength(0)
    }
  })

  it('le plan des murs est tracé APRÈS le fond et AVANT la chaleur', () => {
    const { ctx, ops } = contexte()
    const { width, height } = toile(1)
    const lissee = {
      cell: 4, nx: 2, ny: 2, minX: -20, minY: 20, cells: [], lo: 0, hi: 1, filled: 0,
    } as never
    peindreLesPositions(ctx, width, height, peinture(1, { lissee }))
    const iImage = ops.findIndex((o) => o.op === 'drawImage')
    const iPlan = ops.findIndex((o) => o.op === 'stroke')
    expect(iImage).toBeGreaterThanOrEqual(0)
    expect(iPlan).toBeGreaterThan(iImage)
  })
})
