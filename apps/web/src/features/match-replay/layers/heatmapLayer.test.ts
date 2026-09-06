/**
 * heatmapLayer.test.ts — ce que la carte de chaleur doit dire, et ce qu'elle ne doit PAS
 * dire : une bosse là où l'on est passé, deux bosses pour deux lieux, une échelle qu'un
 * seul point extrême ne peut pas écraser, et RIEN là où personne n'a mis les pieds.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds, ReplayPoint } from '@/lib/api/types'

import {
  buildHeatmap,
  drawHeatmapLayer,
  heatIntensity,
  heatRamp,
  HEAT_RAMP_STEPS,
  type HeatGrid,
} from './heatmapLayer'
import { testReplayDoc } from '../test/testDoc'
import { count, recordingContext } from '../test/recordingContext'

const BOUNDS: ReplayBounds = { minX: 0, minY: 0, maxX: 40, maxY: 40 }

/** Une vie du document de transport : la carte de chaleur n'en lit que le slot et les points. */
interface Vie {
  slot: number
  team: number
  points: ReplayPoint[]
}

/** Un document dont UNE frame vaut 100 ms : les durées se lisent alors sans conversion. */
function docWith(tracks: Vie[], bounds = BOUNDS) {
  return testReplayDoc({ frameIntervalMs: 100, bounds, tracks })
}

/**
 * Une vie immobile en (x, y), ÉCHANTILLONNÉE À CHAQUE IMAGE pendant `frames` images —
 * comme le film le fait (100 ms). Un séjour long est une longue SUITE de mesures, jamais
 * deux mesures très espacées : cette distinction est précisément ce que le plafond de trou
 * protège, et une fixture qui l'ignorerait testerait le plafond au lieu de la durée.
 */
function still(slot: number, x: number, y: number, frames: number): Vie {
  const points: ReplayPoint[] = []
  for (let t = 0; t <= frames; t++) points.push({ t, x, y })
  return { slot, team: 0, points }
}

/** Index de la cellule qui contient la position monde (x, y). */
function cellOf(g: HeatGrid, x: number, y: number): number {
  const i = Math.floor((x - g.minX) / g.cell)
  const j = Math.floor((y - g.minY) / g.cell)
  return j * g.nx + i
}

/** Vrai si la cellule domine STRICTEMENT ses huit voisines — le sommet d'une bosse. */
function isPeak(g: HeatGrid, k: number): boolean {
  const v = g.value[k]
  for (const dj of [-1, 0, 1]) {
    for (const di of [-1, 0, 1]) {
      if (di === 0 && dj === 0) continue
      if (g.value[k + dj * g.nx + di] >= v) return false
    }
  }
  return true
}

describe('buildHeatmap — présence', () => {
  it('un lieu fréquenté donne UNE bosse, centrée là où le joueur était', () => {
    const grid = buildHeatmap(docWith([still(0, 20, 20, 10)]), BOUNDS, 'presence', [])
    expect(grid).not.toBeNull()
    const g = grid as HeatGrid
    expect(isPeak(g, cellOf(g, 20, 20))).toBe(true)
    // La chaleur décroît avec la distance — c'est ça, un lissage.
    expect(g.value[cellOf(g, 21, 20)]).toBeLessThan(g.value[cellOf(g, 20, 20)])
    expect(g.value[cellOf(g, 25, 20)]).toBeLessThan(g.value[cellOf(g, 21, 20)])
  })

  it('deux lieux éloignés donnent DEUX bosses, et rien entre les deux', () => {
    const g = buildHeatmap(
      docWith([still(0, 8, 8, 10), still(1, 32, 32, 10)]),
      BOUNDS,
      'presence',
      [],
    ) as HeatGrid
    expect(isPeak(g, cellOf(g, 8, 8))).toBe(true)
    expect(isPeak(g, cellOf(g, 32, 32))).toBe(true)
    // Le milieu est à plus de 2 sigma des deux : le noyau tronqué n'y dépose rien.
    expect(g.value[cellOf(g, 20, 20)]).toBe(0)
  })

  it('la chaleur est un TEMPS : rester deux fois plus longtemps chauffe deux fois plus', () => {
    const g = buildHeatmap(
      docWith([still(0, 8, 8, 10), still(1, 32, 32, 20)]),
      BOUNDS,
      'presence',
      [],
    ) as HeatGrid
    const court = g.value[cellOf(g, 8, 8)]
    const long = g.value[cellOf(g, 32, 32)]
    expect(long / court).toBeCloseTo(2, 5)
    // Et l'unité est bien la seconde : 10 images de 100 ms = 1 s déposée en tout.
    expect(court).toBeGreaterThan(0)
  })

  it("un TROU d'échantillonnage ne se transforme pas en présence : la durée est plafonnée", () => {
    // Deux mesures seulement, séparées de 1 s ici et de 10 s là. Sans plafond, le trou de
    // 10 s pèserait dix fois plus — alors qu'on ignore où le joueur était pendant ce trou.
    const paire = (dt: number): Vie[] => [
      { slot: 0, team: 0, points: [{ t: 0, x: 20, y: 20 }, { t: dt, x: 20, y: 20 }] },
    ]
    const mesure = buildHeatmap(docWith(paire(10)), BOUNDS, 'presence', []) as HeatGrid
    const trou = buildHeatmap(docWith(paire(100)), BOUNDS, 'presence', []) as HeatGrid
    expect(trou.value[cellOf(trou, 20, 20)]).toBeCloseTo(mesure.value[cellOf(mesure, 20, 20)], 6)
  })

  it('une vie réduite à un seul instant ne dépose rien — une durée exige deux mesures', () => {
    expect(
      buildHeatmap(docWith([{ slot: 0, team: 0, points: [{ t: 0, x: 20, y: 20 }] }]), BOUNDS, 'presence', []),
    ).toBeNull()
  })

  it('aucune trajectoire : pas de calque, jamais une grille vide', () => {
    expect(buildHeatmap(docWith([]), BOUNDS, 'presence', [])).toBeNull()
  })
})

describe('buildHeatmap — éliminations', () => {
  it('compte les morts à la position des victimes, et ignore les trajectoires', () => {
    const g = buildHeatmap(
      docWith([still(0, 8, 8, 10)]),
      BOUNDS,
      'kills',
      [{ x: 30, y: 30, frame: 5 }, { x: 30, y: 30, frame: 6 }, { x: 12, y: 30, frame: 7 }],
    ) as HeatGrid
    // Deux morts au même endroit chauffent deux fois plus qu'une seule.
    expect(g.value[cellOf(g, 30, 30)] / g.value[cellOf(g, 12, 30)]).toBeCloseTo(2, 5)
    // La trajectoire du joueur, elle, ne compte pas dans CETTE lecture.
    expect(g.value[cellOf(g, 8, 8)]).toBe(0)
  })

  it('aucune mort localisée : pas de calque', () => {
    expect(buildHeatmap(docWith([still(0, 8, 8, 10)]), BOUNDS, 'kills', [])).toBeNull()
  })
})

describe('buildHeatmap — échelle par quantiles', () => {
  /** Un match qui couvre la carte : une vie qui la parcourt en serpentant, pas à pas. */
  function parcours(slot: number, side: number): Vie {
    const points: ReplayPoint[] = []
    let t = 0
    for (let row = 0; row < side; row++) {
      for (let col = 0; col < side; col++) {
        const x = 2 + (row % 2 === 0 ? col : side - 1 - col)
        points.push({ t, x, y: 2 + row })
        t += 1
      }
    }
    return { slot, team: 0, points }
  }

  it("un seul point extrême n'écrase pas le reste de la carte (ce qu'un étalonnage sur le max ferait)", () => {
    const grande: ReplayBounds = { minX: 0, minY: 0, maxX: 64, maxY: 64 }
    // Un match plausible : tout le terrain parcouru, un lieu DISPUTÉ où l'on s'attarde
    // (5 s), et UN point extrême où une vie reste plantée 100 s — un joueur qui campe, ou
    // le corps d'un déconnecté.
    const g = buildHeatmap(
      docWith([parcours(0, 60), still(1, 20, 20, 50), still(2, 50, 50, 1_000)], grande),
      grande,
      'presence',
      [],
    ) as HeatGrid
    const ordinaire = cellOf(g, 20, 20)
    const extreme = cellOf(g, 50, 50)

    // L'extrême est bien extrême : plus de dix fois le lieu ordinaire.
    expect(g.value[extreme] / g.value[ordinaire]).toBeGreaterThan(10)
    // Étalonné sur le MAX, le lieu ordinaire tomberait sous un dixième de l'échelle...
    expect(g.value[ordinaire] / g.value[extreme]).toBeLessThan(0.1)
    // ...alors que l'échelle par quantiles le garde chaud, donc lisible.
    expect(heatIntensity(g, ordinaire)).toBeGreaterThan(0.5)
    // Et le haut d'échelle reste dans la population ordinaire, pas sur l'extrême.
    expect(g.hi).toBeLessThan(g.value[extreme])
  })
})

describe('heatIntensity', () => {
  it('rend null hors des lieux fréquentés — « rien » ne se peint pas en froid', () => {
    const g = buildHeatmap(docWith([still(0, 20, 20, 10)]), BOUNDS, 'presence', []) as HeatGrid
    expect(heatIntensity(g, cellOf(g, 0.2, 0.2))).toBeNull()
    expect(heatIntensity(g, cellOf(g, 20, 20))).not.toBeNull()
  })

  it('borne la rampe à [0, 1] : au-delà du haut d échelle, la couleur sature', () => {
    const g: HeatGrid = {
      mode: 'presence',
      cell: 1, nx: 3, ny: 1, minX: 0, minY: 0,
      value: Float32Array.from([0, 1, 100]),
      lo: 2, hi: 4, filled: 2,
    }
    expect(heatIntensity(g, 0)).toBeNull()
    expect(heatIntensity(g, 1)).toBe(0)
    expect(heatIntensity(g, 2)).toBe(1)
  })
})

describe('heatRamp', () => {
  const alphasDe = (ramp: string[]) => ramp.map((c) => Number(/,([\d.]+)\)$/.exec(c)?.[1]))
  const rgbDe = (c: string) => /rgba\((\d+),(\d+),(\d+)/.exec(c)!.slice(1).map(Number)

  it('rend une rampe de HEAT_RAMP_STEPS paliers, opacité croissante et bornée à 0,75', () => {
    const ramp = heatRamp(['#1E3A5F', '#60A5FA'])
    expect(ramp).toHaveLength(HEAT_RAMP_STEPS)
    const alphas = alphasDe(ramp)
    expect(alphas[0]).toBeCloseTo(0.12, 3)
    // A8 (2026-08-18) : le plafond monte de 0,55 à 0,75 — le levier que la mesure du lot
    // R2-V a chiffré comme cinq fois plus efficace que l'abaissement du quantile bas.
    expect(alphas[alphas.length - 1]).toBeCloseTo(0.75, 3)
    for (let i = 1; i < alphas.length; i++) expect(alphas[i]).toBeGreaterThan(alphas[i - 1])
  })

  /**
   * A8 — TROIS POINTS : bleu -> rouge -> violet, le violet AUX EXTRÊMES seulement. Le test
   * tient la règle par les couleurs des trois positions clés : début, milieu, fin.
   */
  it('à trois arrêts, la couleur change DEUX fois et le dernier ne peint que le haut', () => {
    const ramp = heatRamp(['#0000ff', '#ff0000', '#800080'])
    expect(ramp).toHaveLength(HEAT_RAMP_STEPS)
    expect(rgbDe(ramp[0])).toEqual([0, 0, 255])
    expect(rgbDe(ramp[HEAT_RAMP_STEPS - 1])).toEqual([128, 0, 128])
    // Le point milieu ne tombe sur AUCUN palier (64 paliers, donc pas de rang central) : le
    // rouge est pur entre les deux qui l'encadrent, à un cran de quantification près.
    for (const i of [31, 32]) {
      const [r, g, b] = rgbDe(ramp[i])
      expect(r).toBeGreaterThan(248)
      expect(g).toBe(0)
      expect(b).toBeLessThan(6)
    }
    // Dans la moitié BASSE, le rouge ne fait que monter et le bleu que descendre : aucun
    // retour de violet — « aux extrêmes rares » se tient.
    for (let i = 1; i < (HEAT_RAMP_STEPS - 1) / 2; i++) {
      const [r, , b] = rgbDe(ramp[i])
      const [rp, , bp] = rgbDe(ramp[i - 1])
      expect(r).toBeGreaterThanOrEqual(rp)
      expect(b).toBeLessThanOrEqual(bp)
    }
    // L'opacité, elle, ne connaît pas les segments : elle monte de bout en bout.
    const alphas = alphasDe(ramp)
    for (let i = 1; i < alphas.length; i++) expect(alphas[i]).toBeGreaterThan(alphas[i - 1])
  })

  it('une couleur illisible, ou un seul arrêt, rend une rampe VIDE — pas de couleur inventée', () => {
    expect(heatRamp(['', '#60A5FA'])).toEqual([])
    expect(heatRamp(['#1E3A5F', 'var(--absente)'])).toEqual([])
    expect(heatRamp(['#1E3A5F', '#ff0000', ''])).toEqual([])
    expect(heatRamp(['#1E3A5F'])).toEqual([])
    expect(heatRamp([])).toEqual([])
  })
})

describe('drawHeatmapLayer', () => {
  const view = { bounds: { minX: 0, minY: 0, maxX: 4, maxY: 4 }, width: 100, height: 100, pad: 0 }
  const ramp = ['rgba(0,0,0,0.12)', 'rgba(0,0,0,0.3)', 'rgba(0,0,0,0.55)']

  /** Une grille 4x2 : une plage de trois cellules identiques, une cellule chaude, du vide. */
  function grid(): HeatGrid {
    return {
      mode: 'presence',
      cell: 1, nx: 4, ny: 2, minX: 0, minY: 0,
      value: Float32Array.from([1, 1, 1, 0, 0, 0, 0, 10]),
      lo: 1, hi: 10, filled: 4,
    }
  }

  it('ne peint QUE les cellules fréquentées, et fusionne les voisines de même palier', () => {
    const { ops, ctx } = recordingContext()
    drawHeatmapLayer(ctx, grid(), view, { ramp, k: 1 })
    // Trois cellules égales = UN rectangle ; la cellule chaude = un second. Rien d'autre :
    // les quatre cellules vides ne produisent aucun trait.
    expect(count(ops, 'fillRect')).toBe(2)
  })

  it('donne à la cellule la plus chaude le dernier palier de la rampe', () => {
    const { ops, ctx } = recordingContext()
    drawHeatmapLayer(ctx, grid(), view, { ramp, k: 1 })
    const styles = ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])
    expect(styles).toEqual([ramp[0], ramp[ramp.length - 1]])
  })

  it('aligne les bords sur des pixels physiques : deux plages voisines partagent le MÊME bord', () => {
    // Une ligne de deux plages distinctes : le bord droit de la première doit tomber
    // exactement sur le bord gauche de la seconde, sinon une couture claire apparaît.
    const g: HeatGrid = {
      mode: 'presence',
      cell: 1, nx: 2, ny: 1, minX: 0, minY: 0,
      value: Float32Array.from([1, 10]),
      lo: 1, hi: 10, filled: 2,
    }
    const { ops, ctx } = recordingContext()
    drawHeatmapLayer(ctx, g, { ...view, width: 101, height: 101 }, { ramp, k: 2 })
    const rects = ops.filter((o) => o.op === 'fillRect').map((o) => o.args as number[])
    expect(rects).toHaveLength(2)
    expect(rects[0][0] + rects[0][2]).toBe(rects[1][0])
  })

  it('une rampe vide ne peint rien — le calque disparaît plutôt que de mentir', () => {
    const { ops, ctx } = recordingContext()
    drawHeatmapLayer(ctx, grid(), view, { ramp: [], k: 1 })
    expect(count(ops, 'fillRect')).toBe(0)
  })
})

/**
 * V2 (retour utilisateur du 2026-08-18) — LA PORTÉE DE TEMPS.
 *
 * MESURE PRÉALABLE, ET ELLE CORRIGE LA PRÉMISSE : la carte du 16/08 était DÉJÀ celle de tout
 * le match — `accumulatePresence` ne portait aucune borne. Ce que ce lot ajoute, c'est la
 * borne ; ces tests vérifient donc les deux choses qui comptent — sans borne, RIEN ne change ;
 * avec elle, l'avenir du film ne compte pas.
 */
describe('buildHeatmap — portée de temps (V2, 2026-08-18)', () => {
  /** Un joueur qui reste en (8, 8) pendant les 10 premières images, puis en (30, 30). */
  function deuxLieux() {
    const a: ReplayPoint[] = []
    for (let t = 0; t <= 10; t++) a.push({ t, x: 8, y: 8 })
    const b: ReplayPoint[] = []
    for (let t = 11; t <= 20; t++) b.push({ t, x: 30, y: 30 })
    return docWith([
      { slot: 0, team: 0, points: [...a, ...b] },
    ])
  }

  it('sans borne : le comportement du 16/08, tout le film compte', () => {
    const g = buildHeatmap(deuxLieux(), BOUNDS, 'presence', []) as HeatGrid
    expect(g.value[cellOf(g, 8, 8)]).toBeGreaterThan(0)
    expect(g.value[cellOf(g, 30, 30)]).toBeGreaterThan(0)
  })

  it('bornée à l image courante : le SECOND lieu n existe pas encore', () => {
    const g = buildHeatmap(deuxLieux(), BOUNDS, 'presence', [], 10) as HeatGrid
    expect(g.value[cellOf(g, 8, 8)]).toBeGreaterThan(0)
    expect(g.value[cellOf(g, 30, 30)]).toBe(0)
  })

  it('bornée avant toute mesure : aucun calque, jamais une grille vide', () => {
    expect(buildHeatmap(deuxLieux(), BOUNDS, 'presence', [], -1)).toBeNull()
  })

  it('éliminations : une mort postérieure à l image courante ne se compte pas', () => {
    const morts = [
      { x: 30, y: 30, frame: 5 },
      { x: 12, y: 30, frame: 40 },
    ]
    const g = buildHeatmap(deuxLieux(), BOUNDS, 'kills', morts, 10) as HeatGrid
    expect(g.value[cellOf(g, 30, 30)]).toBeGreaterThan(0)
    expect(g.value[cellOf(g, 12, 30)]).toBe(0)
  })
})
