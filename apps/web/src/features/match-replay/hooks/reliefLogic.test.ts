/**
 * Tests — reliefLogic : les décisions pures du fond Relief et de la coupe par étage.
 *
 * Ce qu'ils protègent : (1) l'ordre des fonds Plan, Rendu, Relief et l'absence de commande sans vrai
 * choix ; (2) les étages en RANGS, avec le nombre d'anneaux d'un pion à chaque étage ; (3) les trois états
 * de la commande d'étage (rien, carte sans maillage, étages) ; (4) la demande de rendu : teinte pour le Relief,
 * masques seuls pour le Plan et le Rendu ; (5) le fond posé : relief teinté, fond de base modulé, ou fond de base.
 */
import { describe, expect, it } from 'vitest'

import { fourBlocksScene, reliefData } from '@/lib/replay/relief/reliefScene.testutil'

import type { ReplayMapBackgroundLayer } from './useReplayView'
import {
  backgroundOptions,
  composeReliefBackground,
  floorChips,
  floorStateOf,
  isChipSelected,
  reliefRequestOf,
} from './reliefLogic'

describe('backgroundOptions', () => {
  it('propose Plan, Rendu, Relief dans cet ordre quand le fond a les trois', () => {
    expect(backgroundOptions(true, true)).toEqual(['plan', 'rendu', 'relief'])
  })

  it('omet ce que le fond n’a pas', () => {
    expect(backgroundOptions(true, false)).toEqual(['plan', 'rendu'])
    expect(backgroundOptions(false, true)).toEqual(['rendu', 'relief'])
  })

  it('ne propose rien sans vrai choix : le rendu seul est servi sans commande', () => {
    expect(backgroundOptions(false, false)).toEqual([])
  })
})

describe('floorChips', () => {
  it('numérote les étages en rangs du plus bas au plus haut, sans unité', () => {
    const chips = floorChips([4.4, 0, 2.2], { min: 0, max: 5 })
    expect(chips.map((c) => [c.rank, c.z])).toEqual([[0, 0], [1, 2.2], [2, 4.4]])
  })

  it('dit combien d’anneaux porte un pion posé à la hauteur de chaque étage (langage d’étage des marqueurs)', () => {
    const chips = floorChips([0, 2.2, 4.4], { min: 0, max: 5 })
    const rings = chips.map((c) => c.rings)
    expect(rings[0]).toBe(0)
    expect(rings[1]).toBeGreaterThanOrEqual(1)
    expect(rings[2]).toBeGreaterThan(rings[1])
  })

  it('un terrain plat ne porte aucun anneau, à aucun étage', () => {
    expect(floorChips([0, 2.2], { min: 1, max: 1 }).map((c) => c.rings)).toEqual([0, 0])
  })
})

describe('floorStateOf', () => {
  const RANGE = { min: 0, max: 5 }

  it('rien tant que les données ne sont pas lues (ou que le relief est inutilisable)', () => {
    expect(floorStateOf(null, RANGE)).toEqual({ state: 'none' })
  })

  it('carte sans maillage : l’état dit pourquoi, aucune commande d’étage', () => {
    const data = reliefData({ width: 4, height: 4, degradations: ['sans-maillage'] })
    expect(floorStateOf(data, RANGE)).toEqual({ state: 'noMesh' })
  })

  it('carte sans étage détecté : même état que sans maillage', () => {
    expect(floorStateOf(reliefData({ width: 4, height: 4, levels: [] }), RANGE)).toEqual({ state: 'noMesh' })
  })

  it('carte avec étages : pastilles, bornes du curseur', () => {
    const { data } = fourBlocksScene()
    const state = floorStateOf(data, RANGE)
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
    const state = floorStateOf(data, { min: 0, max: 12 })
    expect(state.state === 'ready' && state.chips.length).toBe(8)
  })
})

describe('isChipSelected', () => {
  const chip = { rank: 1, z: 2.2, rings: 1 }

  it('seule la hauteur exacte de l’étage retient la pastille ; « Tout » n’en retient aucune', () => {
    expect(isChipSelected(2.2, chip)).toBe(true)
    expect(isChipSelected(null, chip)).toBe(false)
  })

  it('une hauteur libre du curseur ne retient aucune pastille', () => {
    expect(isChipSelected(2.25, chip)).toBe(false)
  })
})

describe('reliefRequestOf', () => {
  it('aucune demande quand aucune commande ne se sert du relief', () => {
    expect(reliefRequestOf({ needed: false, turns: 0, cut: null, tintBase: null, lines: false })).toBeNull()
  })

  it('le fond Relief demande l’image teintée (et l’orientation, qui règle la lumière), jamais les masques', () => {
    expect(reliefRequestOf({ needed: true, turns: 2, cut: 2.2, tintBase: '#6b7078', lines: true })).toEqual({
      turns: 2,
      cut: 2.2,
      base: '#6b7078',
      lines: true,
      masks: false,
    })
  })

  it('un fond Plan ou Rendu avec coupe ne demande que les masques, sans teinte, et pas l’orientation', () => {
    expect(reliefRequestOf({ needed: true, turns: 3, cut: 2.2, tintBase: null, lines: false })).toEqual({
      turns: 0,
      cut: 2.2,
      base: null,
      lines: false,
      masks: true,
    })
  })

  it('sans coupe, un fond Plan ou Rendu n’a pas de masque à demander', () => {
    expect(reliefRequestOf({ needed: true, turns: 0, cut: null, tintBase: null, lines: true })?.masks).toBe(false)
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
  const RELIEF = 'relief' as unknown as CanvasImageSource
  const MASKED = 'masque' as unknown as CanvasImageSource

  it('pas de fond de base : pas de fond', () => {
    expect(composeReliefBackground(null, 'relief', { relief: RELIEF, masked: null })).toBeNull()
  })

  it('le relief choisi et rendu devient le fond, variante relief, au même calage', () => {
    const out = composeReliefBackground(BASE, 'relief', { relief: RELIEF, masked: null })
    expect(out).toMatchObject({ image: RELIEF, variante: 'relief', calibration: BASE.calibration })
  })

  it('le relief choisi mais pas encore rendu : le fond de base reste posé, sans saut', () => {
    expect(composeReliefBackground(BASE, 'relief', { relief: null, masked: null })).toBe(BASE)
  })

  it('un fond Plan ou Rendu modulé par la coupe garde sa variante, donc le sort du calque des murs', () => {
    const out = composeReliefBackground(BASE, 'plan', { relief: null, masked: MASKED })
    expect(out).toMatchObject({ image: MASKED, variante: 'plan' })
  })

  it('sans coupe, le fond de base est rendu tel quel', () => {
    expect(composeReliefBackground(BASE, 'rendu', { relief: null, masked: null })).toBe(BASE)
  })

  it('une image de relief obsolète ne se pose pas sur un fond Plan ou Rendu', () => {
    expect(composeReliefBackground(BASE, 'plan', { relief: RELIEF, masked: null })).toBe(BASE)
  })
})
