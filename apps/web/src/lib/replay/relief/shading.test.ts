import { describe, expect, it } from 'vitest'

import type { QuarterTurns } from '../orientation'
import { QUARTER_TURNS } from '../orientation'
import { heightGrey, lightForTurns, shadeRelief } from './shading'
import { reliefImage } from './reliefScene.testutil'
import type { ReliefHeights } from './types'

const RAMP = { low: 0, high: 10 }
const MPP = 0.1

describe('lightForTurns : la lumière reste en haut à gauche de l’écran', () => {
  it('donne la direction dans le repère de l’image pour chaque quart de tour horaire', () => {
    expect(lightForTurns(0)).toEqual({ x: -1, y: -1 })
    expect(lightForTurns(1)).toEqual({ x: -1, y: 1 })
    expect(lightForTurns(2)).toEqual({ x: 1, y: 1 })
    expect(lightForTurns(3)).toEqual({ x: 1, y: -1 })
  })
})

describe('shadeRelief : la clarté suit la hauteur', () => {
  const flat = (z: number) => reliefImage(20, 20, () => ({ z }))

  it('un sol plat prend la clarté de sa hauteur sur la rampe, sans ombrage', () => {
    const at = (z: number) => shadeRelief(flat(z), RAMP, MPP, 0)[10 * 20 + 10]
    expect(at(0)).toBe(Math.round(0.36 * 255))
    expect(at(5)).toBe(Math.round((0.36 + 0.64 * 0.5) * 255))
    expect(at(10)).toBe(255)
    expect(at(-5)).toBe(at(0))
    expect(at(50)).toBe(255)
  })

  it('plus haut est plus clair', () => {
    const grey = shadeRelief(reliefImage(20, 20, (i) => ({ z: i * 0.5 })), RAMP, MPP, 0)
    expect(grey[10 * 20 + 2]).toBeLessThanOrEqual(grey[10 * 20 + 9])
    expect(heightGrey(10, RAMP)).toBeGreaterThan(heightGrey(0, RAMP))
  })

  it('une rampe de moins d’une unité ne divise pas par zéro', () => {
    expect(Number.isFinite(heightGrey(3, { low: 3, high: 3 }))).toBe(true)
  })

  it('un pixel de hauteur inconnue reste à zéro et ne contamine pas ses voisins', () => {
    const img = reliefImage(20, 20, (i, j) => (i === 10 && j === 10 ? null : { z: 5 }))
    const grey = shadeRelief(img, RAMP, MPP, 0)
    expect(grey[10 * 20 + 10]).toBe(0)
    expect(grey[10 * 20 + 11]).toBe(Math.round((0.36 + 0.64 * 0.5) * 255))
  })
})

/** Un dôme : le même éloignement du centre donne la même hauteur (donc la même clarté de rampe). */
function dome(): ReliefHeights {
  return reliefImage(41, 41, (i, j) => ({ z: Math.max(0, 6 - 0.15 * Math.hypot(i - 20, j - 20)) }))
}

/** Le gris au point `(dx, dy)` du centre du dôme, à 8 pixels de lui sur la diagonale. */
const flank = (grey: Uint8Array, dx: number, dy: number): number => grey[(20 + dy) * 41 + (20 + dx)]

describe('shadeRelief : la lumière vient toujours du haut à gauche de l’ÉCRAN', () => {
  it('carte non tournée : le flanc haut-gauche est plus clair que le flanc bas-droit', () => {
    const grey = shadeRelief(dome(), RAMP, MPP, 0)
    expect(flank(grey, -8, -8)).toBeGreaterThan(flank(grey, 8, 8))
  })

  it('carte tournée d’un quart de tour : c’est le flanc bas-gauche de l’image qui est éclairé', () => {
    const grey = shadeRelief(dome(), RAMP, MPP, 1)
    expect(flank(grey, -8, 8)).toBeGreaterThan(flank(grey, 8, -8))
  })

  it('carte tournée de 180° : le flanc bas-droit de l’image est éclairé (haut-gauche à l’écran)', () => {
    const grey = shadeRelief(dome(), RAMP, MPP, 2)
    expect(flank(grey, 8, 8)).toBeGreaterThan(flank(grey, -8, -8))
  })

  it('carte tournée de trois quarts de tour : c’est le flanc haut-droit de l’image qui est éclairé', () => {
    const grey = shadeRelief(dome(), RAMP, MPP, 3)
    expect(flank(grey, 8, -8)).toBeGreaterThan(flank(grey, -8, 8))
  })

  it('l’ombrage d’une orientation, vu à l’écran (tourné), est celui de la carte tournée éclairée en haut à gauche', () => {
    // Relief quelconque et non symétrique, sur une image non carrée.
    const w = 30
    const h = 20
    const hill = (i: number, j: number): number => 3 + 2 * Math.sin(i * 0.31) + 1.5 * Math.cos(j * 0.23) + i * 0.05
    const base = reliefImage(w, h, (i, j) => ({ z: hill(i, j) }))
    let screen: ReliefHeights = base
    for (const q of QUARTER_TURNS.slice(1) as QuarterTurns[]) {
      screen = rotateCw(screen)
      // L'ombrage de la carte pour q quarts de tour, tourné q fois comme la carte l'est à l'écran.
      let shaded: Uint8Array = shadeRelief(base, RAMP, MPP, q)
      let dims = { w, h }
      for (let k = 0; k < q; k++) {
        shaded = rotateGridCw(shaded, dims.w, dims.h)
        dims = { w: dims.h, h: dims.w }
      }
      const reference = shadeRelief(screen, RAMP, MPP, 0)
      expect(shaded.length).toBe(reference.length)
      let worst = 0
      for (let p = 0; p < reference.length; p++) worst = Math.max(worst, Math.abs(shaded[p] - reference[p]))
      expect(worst, `orientation ${q}`).toBeLessThanOrEqual(1)
    }
  })
})

/** Quart de tour horaire d'une grille de `w × h` : le pixel `(x, y)` va en `(h − 1 − y, x)`. */
function rotateGridCw<T extends Uint8Array | Float32Array>(src: T, w: number, h: number): T {
  const out = new (src.constructor as new (n: number) => T)(src.length)
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) out[x * h + (h - 1 - y)] = src[y * w + x]
  return out
}

function rotateCw(img: ReliefHeights): ReliefHeights {
  return {
    width: img.height,
    height: img.width,
    heights: rotateGridCw(img.heights, img.width, img.height),
    flags: rotateGridCw(img.flags, img.width, img.height),
  }
}
