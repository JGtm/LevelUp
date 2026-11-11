/**
 * Tests — les couleurs de série du canvas, extraites telles quelles de `ReplayCanvas.tsx` : une par
 * joueur quand l'option est active (rien sinon), une par GRANDE zone nommée.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { CalloutZoneReady } from '../../../lib/replay/calloutsPaint'
import { useDistinctPlayerColors, useZoneSeriesColors } from './useSeriesColors'

vi.mock('@/lib/accessibility/plotlyColorscale', () => ({
  getSeriesColors: (n: number) => Array.from({ length: n }, (_, i) => `serie-${i}`),
}))

const zone = (big: boolean) => ({ big }) as CalloutZoneReady

describe('useDistinctPlayerColors', () => {
  it('option éteinte : aucune série', () => {
    expect(renderHook(() => useDistinctPlayerColors(null, 1)).result.current).toBeNull()
  })

  it('option active : une couleur par joueur du roster', () => {
    expect(renderHook(() => useDistinctPlayerColors(3, 1)).result.current).toEqual(['serie-0', 'serie-1', 'serie-2'])
  })

  it('se relit à chaque version de palette', () => {
    const { result, rerender } = renderHook(({ v }) => useDistinctPlayerColors(2, v), { initialProps: { v: 1 } })
    const first = result.current
    rerender({ v: 2 })
    expect(result.current).not.toBe(first)
  })
})

describe('useZoneSeriesColors', () => {
  it('une couleur par GRANDE zone seulement', () => {
    const zones = [zone(true), zone(false), zone(true)]
    expect(renderHook(() => useZoneSeriesColors(zones, 1)).result.current).toEqual(['serie-0', 'serie-1'])
  })

  it('aucune grande zone : liste vide', () => {
    expect(renderHook(() => useZoneSeriesColors([zone(false)], 1)).result.current).toEqual([])
  })
})
