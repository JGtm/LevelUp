/**
 * Tests — reliefLogic : ce que le rejeu ajoute à la logique commune du fond Relief.
 *
 * Ce qu'ils protègent : (1) les étages en RANGS, avec le nombre d'anneaux d'un pion à chaque étage ; (2) l'état de la
 * commande d'étage du rejeu = celui de la vue commune, ses pastilles portant les anneaux ; (3) le fond posé : relief
 * teinté, fond de base modulé, ou fond de base. L'ordre des fonds, la demande de rendu et la pastille retenue sont
 * ceux de `lib/replay/reliefViewLogic` (testés là).
 */
import { describe, expect, it } from 'vitest'

import { floorRanks } from '@/lib/replay/relief'
import { fourBlocksScene, reliefData } from '@/lib/replay/relief/reliefScene.testutil'
import { viewFloorState } from '@/lib/replay/reliefViewLogic'
import type { ReliefSurface } from '@/lib/replay/useReliefView'

import type { ReplayMapBackgroundLayer } from './useReplayView'
import { composeReliefBackground, floorChips, floorStateWithRings } from './reliefLogic'

describe('floorChips', () => {
  it('numérote les étages en rangs du plus bas au plus haut, sans unité', () => {
    const chips = floorChips(floorRanks([4.4, 0, 2.2]), { min: 0, max: 5 })
    expect(chips.map((c) => [c.rank, c.z])).toEqual([[0, 0], [1, 2.2], [2, 4.4]])
  })

  it('dit combien d’anneaux porte un pion posé à la hauteur de chaque étage (langage d’étage des marqueurs)', () => {
    const chips = floorChips(floorRanks([0, 2.2, 4.4]), { min: 0, max: 5 })
    const rings = chips.map((c) => c.rings)
    expect(rings[0]).toBe(0)
    expect(rings[1]).toBeGreaterThanOrEqual(1)
    expect(rings[2]).toBeGreaterThan(rings[1])
  })

  it('un terrain plat ne porte aucun anneau, à aucun étage', () => {
    expect(floorChips(floorRanks([0, 2.2]), { min: 1, max: 1 }).map((c) => c.rings)).toEqual([0, 0])
  })
})

describe('floorStateWithRings', () => {
  const RANGE = { min: 0, max: 5 }

  it('rien tant que les données ne sont pas lues (ou que le relief est inutilisable)', () => {
    expect(floorStateWithRings(viewFloorState(null), RANGE)).toEqual({ state: 'none' })
  })

  it('carte sans maillage : l’état dit pourquoi, aucune commande d’étage', () => {
    const data = reliefData({ width: 4, height: 4, degradations: ['sans-maillage'] })
    expect(floorStateWithRings(viewFloorState(data), RANGE)).toEqual({ state: 'noMesh' })
  })

  it('carte sans étage détecté : même état que sans maillage', () => {
    expect(floorStateWithRings(viewFloorState(reliefData({ width: 4, height: 4, levels: [] })), RANGE)).toEqual({ state: 'noMesh' })
  })

  it('carte avec étages : pastilles, bornes du curseur', () => {
    const { data } = fourBlocksScene()
    const state = floorStateWithRings(viewFloorState(data), RANGE)
    expect(state.state).toBe('ready')
    if (state.state !== 'ready') return
    expect(state.chips.map((c) => c.rank)).toEqual([0, 1])
    expect(state.min).toBe(0)
    expect(state.max).toBe(2.2)
  })

  it('beaucoup d’étages : une pastille chacun, aucune n’est écartée', () => {
    const levels = [0, 1.5, 3, 4.5, 6, 7.5, 9, 10.5]
    const rects = levels.map((z, i) => ({ i0: i % 4, j0: Math.floor(i / 4), i1: i % 4, j1: Math.floor(i / 4), z }))
    const data = reliefData({ width: 4, height: 4, levels, rects })
    const state = floorStateWithRings(viewFloorState(data), { min: 0, max: 12 })
    expect(state.state === 'ready' && state.chips.length).toBe(8)
  })
})

describe('composeReliefBackground', () => {
  const BASE = {
    calibration: { widthPx: 4, heightPx: 4, metersPerPixel: 1, originX: 0, originY: 4 },
    image: 'base' as unknown as CanvasImageSource,
    plan: null,
    variante: 'plan',
    planAvailable: true,
    reliefAvailable: true,
    zonesClip: null,
  } as unknown as ReplayMapBackgroundLayer
  const SURFACE = { image: 'canvas-du-relief' as unknown as HTMLCanvasElement, frame: {} } as unknown as ReliefSurface

  it('pas de fond de base : pas de fond', () => {
    expect(composeReliefBackground(null, SURFACE, true)).toBeNull()
  })

  it('le relief retenu et rendu devient le fond, variante relief, au même calage', () => {
    const out = composeReliefBackground(BASE, SURFACE, true)
    expect(out).toMatchObject({ image: SURFACE.image, variante: 'relief', calibration: BASE.calibration })
  })

  it('le relief retenu mais pas encore rendu : le fond de base reste posé, sans saut', () => {
    expect(composeReliefBackground(BASE, null, false)).toBe(BASE)
  })

  it('un fond Plan ou Rendu modulé par la coupe garde sa variante, donc le sort du calque des murs', () => {
    const out = composeReliefBackground(BASE, SURFACE, false)
    expect(out).toMatchObject({ image: SURFACE.image, variante: 'plan' })
  })

  it('sans coupe ni relief, le fond de base est rendu tel quel', () => {
    expect(composeReliefBackground(BASE, null, false)).toBe(BASE)
  })
})
