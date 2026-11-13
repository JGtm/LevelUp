/**
 * plan.orientation.test.ts — LE PLAN DE L'ONGLET TACTIQUE SUIT L'ORIENTATION DE LA CARTE (la même
 * que le rejeu 2D) : projection tournée d'un point connu, clic converti exactement en monde,
 * cellules et sélection tournées avec la carte, image du fond tournée autour de son centre, plan
 * des murs tracé sous les zones et la chaleur (rien sans plan).
 */
import { describe, expect, it } from 'vitest'

import { buildTacticalGrid } from '@/lib/replay/heatPaint'
import { orientAbout, QUARTER_TURNS, type QuarterTurns } from '@/lib/replay/orientation'
import type { WallPlanReady } from '@/lib/replay/wallPlan'

import { cadrageDuFond } from './plan.logic'
import { peindreLePlan } from './planPaint'
import { aspectDuPlan } from './tacticalLecture.logic'
import {
  celluleDuClic,
  grilleDuPlan,
  pointDuClic,
  projectionDuPlan,
  rectSelection,
  repereAspect,
  repereDuPlan,
  versLeMonde,
  type RepereTactique,
} from './tacticalView.logic'

// Le cadre du fond : origine (-20, 60), 80 x 40 m, donc x de -20 à 60 et y de 20 à 60. Pas de 4 m.
const FOND = { originX: -20, originY: 60, widthM: 80, heightM: 40 }
const BORNES = { min_x: 0, max_x: 40, min_y: 20, max_y: 60, valide: true }
const PAS = 4
const CANVAS_L = 400

function repereAt(turns: QuarterTurns): RepereTactique {
  const r = repereDuPlan(FOND, BORNES, PAS, turns)
  if (!r) throw new Error('repère attendu')
  return r
}

/** Le pivot du monde : le centre du cadre du fond. */
const PIVOT = { x: 20, y: 40 }

describe('repereDuPlan — carte tournée', () => {
  it('à 0 quart de tour : le repère d’avant, sans champ d’orientation', () => {
    expect(repereAt(0)).toEqual({ minX: -20, maxX: 60, minY: 20, maxY: 60, pasM: PAS })
  })

  it('à 1 et 3 quarts de tour : largeur et hauteur échangées autour du même centre', () => {
    for (const turns of [1, 3] as QuarterTurns[]) {
      const r = repereAt(turns)
      expect(r.maxX - r.minX).toBe(40)
      expect(r.maxY - r.minY).toBe(80)
      expect((r.minX + r.maxX) / 2).toBe(PIVOT.x)
      expect((r.minY + r.maxY) / 2).toBe(PIVOT.y)
      expect(r.orient).toEqual({ turns, pivot: PIVOT })
    }
    expect(repereAspect(repereAt(1))).toBeCloseTo(0.5, 12)
    expect(repereAspect(repereAt(0))).toBeCloseTo(2, 12)
  })

  it('le rapport du cadre suit la carte tournée, y compris avant la lecture (calage seul)', () => {
    expect(aspectDuPlan(FOND as never, null, 0)).toBeCloseTo(2, 12)
    expect(aspectDuPlan(FOND as never, null, 1)).toBeCloseTo(0.5, 12)
    expect(aspectDuPlan(FOND as never, null, 2)).toBeCloseTo(2, 12)
    expect(aspectDuPlan(FOND as never, repereAt(1), 1)).toBeCloseTo(0.5, 12)
  })
})

describe('projectionDuPlan — un point monde connu tombe au pixel de la carte tournée', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour', (turns) => {
    const repere = repereAt(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)
    expect(proj).not.toBeNull()
    const scale = CANVAS_L / (repere.maxX - repere.minX)
    for (const p of [{ x: -20, y: 60 }, { x: 60, y: 20 }, { x: 5, y: 33 }, { x: 20, y: 40 }]) {
      const q = orientAbout(p, turns, PIVOT)
      const attendu = { x: (q.x - repere.minX) * scale, y: (repere.maxY - q.y) * scale }
      const got = proj!(p)
      expect(got.x).toBeCloseTo(attendu.x, 9)
      expect(got.y).toBeCloseTo(attendu.y, 9)
    }
  })

  it('le coin haut-gauche du monde tombe dans le coin attendu : à 1 quart de tour, en haut à droite', () => {
    // Rotation horaire : le coin nord-ouest du monde passe nord-est.
    const proj = projectionDuPlan(repereAt(1), CANVAS_L)!
    const coin = proj({ x: -20, y: 60 })
    expect(coin.x).toBeCloseTo(CANVAS_L, 9)
    expect(coin.y).toBeCloseTo(0, 9)
  })
})

describe('survol et clic — l’inverse est exact', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour : du pixel au monde, le point projeté est retrouvé', (turns) => {
    const repere = repereAt(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)!
    const hauteur = CANVAS_L / repereAspect(repere)
    for (const p of [{ x: -17.5, y: 58 }, { x: 59, y: 21 }, { x: 12.25, y: 36.5 }]) {
      const px = proj(p)
      const clic = pointDuClic(px.x, px.y, { width: CANVAS_L, height: hauteur }, repere)
      expect(clic).not.toBeNull()
      const monde = versLeMonde(repere, clic!)
      expect(monde.x).toBeCloseTo(p.x, 9)
      expect(monde.y).toBeCloseTo(p.y, 9)
      // La cellule serveur visée est celle du point monde.
      expect(celluleDuClic(px.x, px.y, { width: CANVAS_L, height: hauteur }, repere)).toEqual({
        col: Math.floor(p.x / PAS),
        row: Math.floor(p.y / PAS),
      })
    }
  })
})

describe('cellules et sélection — tournées avec la carte', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour : la cellule choisie est encadrée là où son centre se projette', (turns) => {
    const repere = repereAt(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)!
    const cellule = { col: 3, row: 11 } // adresse serveur : x = 12..16, y = 44..48
    const rect = rectSelection(cellule, repere, CANVAS_L)
    expect(rect).not.toBeNull()
    const centre = proj({ x: (cellule.col + 0.5) * PAS, y: (cellule.row + 0.5) * PAS })
    expect(rect!.x + rect!.size / 2).toBeCloseTo(centre.x, 9)
    expect(rect!.y + rect!.size / 2).toBeCloseTo(centre.y, 9)
  })

  it.each(QUARTER_TURNS)('à %i quart(s) de tour : la grille peinte porte la valeur de chaque cellule à son centre projeté', (turns) => {
    const repere = repereAt(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)!
    const cellules = [
      { col: 3, lig: 11, valeur: 7, brut: 7, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 14, centre_y: 46 },
      { col: -4, lig: 6, valeur: 2, brut: 2, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: -14, centre_y: 26 },
    ]
    const grille = grilleDuPlan(cellules, repere, { p50: 1, p95: 7, borne: 7, n_cellules: 2, symetrique: false })
    expect(grille).not.toBeNull()
    const echelle = CANVAS_L / (repere.maxX - repere.minX)
    for (const c of cellules) {
      const centre = proj({ x: c.centre_x, y: c.centre_y })
      // La cellule de la grille qui contient ce pixel (ligne 0 en haut, cadre tourné).
      const col = Math.floor(centre.x / (PAS * echelle))
      const row = Math.floor(centre.y / (PAS * echelle))
      const trouvee = grille!.cells.find((k) => k.col === col && k.row === row)
      expect(trouvee?.value).toBe(c.valeur)
    }
  })
})

describe('cadrageDuFond — l’image tourne autour de son centre', () => {
  /** Simule le CSS : l'image posée en pourcentages du cadre, puis tournée de `turns` x 90 deg autour de son centre. */
  function coinsDeLImage(turns: QuarterTurns) {
    const repere = repereAt(turns)
    const cadre = cadrageDuFond(repere, repere, turns)
    const l = CANVAS_L
    const h = CANVAS_L / repereAspect(repere)
    const pct = (s: string) => parseFloat(s) / 100
    const x0 = pct(cadre.left) * l
    const y0 = pct(cadre.top) * h
    const w0 = pct(cadre.width) * l
    const h0 = pct(cadre.height) * h
    const cx = x0 + w0 / 2
    const cy = y0 + h0 / 2
    const theta = (turns * Math.PI) / 2
    const tourne = (x: number, y: number) => ({
      x: cx + (x - cx) * Math.cos(theta) - (y - cy) * Math.sin(theta),
      y: cy + (x - cx) * Math.sin(theta) + (y - cy) * Math.cos(theta),
    })
    return { cadre, haut_gauche: tourne(x0, y0), bas_droite: tourne(x0 + w0, y0 + h0), repere }
  }

  it.each(QUARTER_TURNS)('à %i quart(s) de tour : ses coins tombent aux pixels de la projection du plan', (turns) => {
    const { cadre, haut_gauche, bas_droite, repere } = coinsDeLImage(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)!
    // Les coins de l'image : (minX monde, maxY monde) en haut à gauche avant rotation, (maxX, minY) en bas à droite.
    const nw = proj({ x: -20, y: 60 })
    const se = proj({ x: 60, y: 20 })
    expect(haut_gauche.x).toBeCloseTo(nw.x, 6)
    expect(haut_gauche.y).toBeCloseTo(nw.y, 6)
    expect(bas_droite.x).toBeCloseTo(se.x, 6)
    expect(bas_droite.y).toBeCloseTo(se.y, 6)
    expect(cadre.transform).toBe(turns === 0 ? undefined : `rotate(${turns * 90}deg)`)
  })

  it('à 0 quart de tour : le cadrage d’avant, sans transformation', () => {
    const scene = { minX: 0, maxX: 10, minY: 0, maxY: 5 }
    expect(cadrageDuFond(scene, scene)).toEqual({ left: '0%', top: '0%', width: '100%', height: '100%' })
  })
})

describe('peindreLePlan — le plan des murs, sous tout le reste', () => {
  function contexte() {
    const appels: Array<{ op: string; args: unknown[] }> = []
    const etat: Record<string, unknown> = {}
    const ctx = new Proxy(etat, {
      get(cible, nom: string) {
        if (nom in cible) return cible[nom]
        return (...args: unknown[]) => {
          appels.push({ op: nom, args })
        }
      },
      set(cible, nom: string, valeur: unknown) {
        cible[nom] = valeur
        return true
      },
    })
    return { ctx: ctx as unknown as CanvasRenderingContext2D, appels }
  }

  const GRILLE = buildTacticalGrid(
    [{ col: 2, row: 1, value: 3 }],
    { cell: 4, nx: 20, ny: 10, minX: -20, minY: 20 },
    { lo: 1, hi: 5, signee: false },
    1,
  )
  const PLAN: WallPlanReady = {
    edges: [
      [-20, 60, 60, 60],
      [0, 30, 0, 50],
    ],
    dropped: 0,
  }
  const peinture = (plan: WallPlanReady | null) => ({ grid: GRILLE, ramp: ['a', 'b'], zones: [], locale: 'fr' as const, encre: 'encre', plan })

  it.each(QUARTER_TURNS)('à %i quart(s) de tour : chaque arête est tracée entre les pixels de la projection tournée', (turns) => {
    const repere = repereAt(turns)
    const proj = projectionDuPlan(repere, CANVAS_L)!
    const { ctx, appels } = contexte()
    peindreLePlan(ctx, repere, CANVAS_L, peinture(PLAN))
    const segments = appels.filter((a) => a.op === 'moveTo' || a.op === 'lineTo')
    expect(segments).toHaveLength(4)
    PLAN.edges.forEach(([x1, y1, x2, y2], i) => {
      const a = proj({ x: x1, y: y1 })
      const b = proj({ x: x2, y: y2 })
      expect(segments[2 * i].args).toEqual([a.x, a.y])
      expect(segments[2 * i + 1].args).toEqual([b.x, b.y])
    })
  })

  it('le plan est tracé AVANT la chaleur et avant tout remplissage', () => {
    const { ctx, appels } = contexte()
    peindreLePlan(ctx, repereAt(0), CANVAS_L, peinture(PLAN))
    const premierTrait = appels.findIndex((a) => a.op === 'stroke')
    const premierRemplissage = appels.findIndex((a) => a.op === 'fillRect' || a.op === 'fill')
    expect(premierTrait).toBeGreaterThanOrEqual(0)
    if (premierRemplissage >= 0) expect(premierTrait).toBeLessThan(premierRemplissage)
  })

  it('sans plan (carte sans plan, ou interrupteur éteint) : aucune arête', () => {
    const { ctx, appels } = contexte()
    peindreLePlan(ctx, repereAt(1), CANVAS_L, peinture(null))
    expect(appels.filter((a) => a.op === 'moveTo')).toHaveLength(0)
  })
})
