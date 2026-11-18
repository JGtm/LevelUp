/**
 * useResolvedTurns.test.ts — L'ORIENTATION EFFECTIVE : le choix retenu l'emporte toujours, sans choix
 * c'est la règle Auto (`autoTurns`) qui décide, avec son hystérésis.
 *
 * Les cotes sont celles d'une carte de 100 x 120 dans une boîte de hauteur 100 : le gain du plan tourné
 * y vaut ~21 % à 108 de large (sous le seuil de retour), ~23 % à 111 (entre les deux seuils) et ~25,4 % à 112
 * (au-dessus du seuil pour tourner). Le seuil lui-même n'est pas réécrit ici : seul son effet est observé.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { requestExportLayout } from '../export/exportLayoutStore'
import { useResolvedTurns } from './useResolvedTurns'

const PORTRAIT: ReplayBounds = { minX: 0, maxX: 100, minY: 0, maxY: 120 }
const CARRE: ReplayBounds = { minX: 0, maxX: 100, minY: 0, maxY: 100 }
/** Une carte haute (Ronin : 659 x 1288) dans une boîte large. */
const HAUTE: ReplayBounds = { minX: 0, maxX: 659, minY: 0, maxY: 1288 }

interface Props {
  scopeKey?: string
  choice: QuarterTurns | null
  bounds: ReplayBounds
  width: number
  freeHeight: number
}

function render(initial: Props) {
  return renderHook((p: Props) => useResolvedTurns(p.scopeKey ?? 'm1', p.choice, p.bounds, p.width, p.freeHeight), { initialProps: initial })
}

describe('useResolvedTurns — Auto', () => {
  it('tourne une carte haute dans une boîte large', () => {
    const { result } = render({ choice: null, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current).toEqual({ turns: 1, autoTurned: true })
  })

  it('laisse droite une carte carrée', () => {
    const { result } = render({ choice: null, bounds: CARRE, width: 1390, freeHeight: 700 })
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })

  it('laisse droit le plan tant que la boîte n’est pas mesurée', () => {
    const { result } = render({ choice: null, bounds: HAUTE, width: 0, freeHeight: 480 })
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })

  it('HYSTÉRÉSIS : la boîte qui varie entre les deux seuils ne fait pas basculer le plan', () => {
    const { result, rerender } = render({ choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    // Droit, à ~23 % : sous le seuil pour tourner.
    expect(result.current.turns).toBe(0)
    // Le seuil pour tourner est atteint : le plan tourne.
    rerender({ choice: null, bounds: PORTRAIT, width: 112, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    // Retour à ~23 % : entre 20 % et 25 %, le plan reste tourné.
    rerender({ choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    rerender({ choice: null, bounds: PORTRAIT, width: 110, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    // Sous le seuil de retour (~16,6 %) : le plan revient droit.
    rerender({ choice: null, bounds: PORTRAIT, width: 108, freeHeight: 100 })
    expect(result.current.turns).toBe(0)
    // Et il y reste tant que le seuil pour tourner n'est pas atteint.
    rerender({ choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    expect(result.current.turns).toBe(0)
  })
})

describe('useResolvedTurns — un choix retenu', () => {
  it.each([0, 1, 2, 3] as const)('%i quart(s) de tour retenu(s) : respecté, jamais écrasé par l’Auto', (choice) => {
    const { result } = render({ choice, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current).toEqual({ turns: choice, autoTurned: false })
  })

  it('un 0 retenu garde droite une carte que l’Auto tournerait', () => {
    const { result, rerender } = render({ choice: null, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current.turns).toBe(1)
    rerender({ choice: 0, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })

  it('le retour en Auto repart du plan droit : il ne garde pas l’état d’avant le choix', () => {
    const { result, rerender } = render({ choice: null, bounds: PORTRAIT, width: 112, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    rerender({ choice: 2, bounds: PORTRAIT, width: 112, freeHeight: 100 })
    rerender({ choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    // Entre les deux seuils, sans état antérieur « tourné » : le plan reste droit.
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })
})

describe('useResolvedTurns — la portée de l’état retenu', () => {
  it('un plan tourné à +30 % dans un match ne tourne pas le match suivant à +23 % (entre les deux seuils)', () => {
    const { result, rerender } = render({ scopeKey: 'm1', choice: null, bounds: PORTRAIT, width: 114, freeHeight: 100 })
    expect(result.current).toEqual({ turns: 1, autoTurned: true })
    // Même match, +23 % : l'hystérésis garde le plan tourné.
    rerender({ scopeKey: 'm1', choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    // Autre match, mêmes mesures : il repart du plan droit, +23 % ne suffit pas à le tourner.
    rerender({ scopeKey: 'm2', choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })

  it('revenir au premier match ne rend pas l’état qu’il avait : la portée courante seule compte', () => {
    const { result, rerender } = render({ scopeKey: 'm1', choice: null, bounds: PORTRAIT, width: 114, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    rerender({ scopeKey: 'm2', choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    rerender({ scopeKey: 'm1', choice: null, bounds: PORTRAIT, width: 111, freeHeight: 100 })
    expect(result.current.turns).toBe(0)
  })
})

describe('useResolvedTurns — gelée pendant un export', () => {
  afterEach(() => {
    act(() => requestExportLayout(null))
  })
  const exporter = () => act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
  const finir = () => act(() => requestExportLayout(null))

  it('un plan tourné reste tourné quand la fenêtre se redimensionne pendant l’export, puis la règle reprend', () => {
    const { result, rerender } = render({ choice: null, bounds: PORTRAIT, width: 114, freeHeight: 100 })
    expect(result.current.turns).toBe(1)
    exporter()
    rerender({ choice: null, bounds: PORTRAIT, width: 100, freeHeight: 100 })
    expect(result.current).toEqual({ turns: 1, autoTurned: true })
    finir()
    // La mesure d'après l'export (0 % de gain, sous le seuil de retour) s'applique.
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
  })

  it('un plan droit reste droit quand la fenêtre s’élargit pendant l’export', () => {
    const { result, rerender } = render({ choice: null, bounds: PORTRAIT, width: 100, freeHeight: 100 })
    expect(result.current.turns).toBe(0)
    exporter()
    rerender({ choice: null, bounds: PORTRAIT, width: 130, freeHeight: 100 })
    expect(result.current).toEqual({ turns: 0, autoTurned: false })
    finir()
    expect(result.current).toEqual({ turns: 1, autoTurned: true })
  })

  it('le gel garde l’état d’avant l’export : la hauteur et les bornes qui changent n’y font rien non plus', () => {
    const { result, rerender } = render({ choice: null, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current.turns).toBe(1)
    exporter()
    rerender({ choice: null, bounds: CARRE, width: 1390, freeHeight: 700 })
    expect(result.current.turns).toBe(1)
  })

  it('un choix retenu l’emporte aussi pendant l’export', () => {
    const { result, rerender } = render({ choice: null, bounds: HAUTE, width: 1390, freeHeight: 700 })
    exporter()
    rerender({ choice: 2, bounds: HAUTE, width: 1390, freeHeight: 700 })
    expect(result.current).toEqual({ turns: 2, autoTurned: false })
  })
})
