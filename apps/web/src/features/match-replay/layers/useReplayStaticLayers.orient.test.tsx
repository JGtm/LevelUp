/**
 * useReplayStaticLayers.orient.test.tsx — la carte de chaleur tourne avec la carte : la grille
 * transmise au tracé est celle que `orientHeatGrid` produit, et chaque cellule tombe au pixel que
 * `projectTo` donne pour le centre de sa cellule d'origine.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { HeatGrid } from '../../../lib/replay/heatPaint'
import type { ReplayBounds } from '@/lib/api/types'
import { orientBounds, QUARTER_TURNS, viewOrientation, type QuarterTurns } from '../model/orientation'
import { projectTo, type CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { useReplayStaticLayers } from './useReplayStaticLayers'

const drawHeat = vi.hoisted(() => vi.fn())
vi.mock('../../../lib/replay/heatPaint', () => ({ drawHeatmapLayer: drawHeat }))

const SCENE: ReplayBounds = { minX: 0, maxX: 24, minY: 0, maxY: 16 }
const GRID: HeatGrid = {
  cell: 4,
  nx: 6,
  ny: 4,
  minX: 0,
  minY: 0,
  mode: 'presence',
  // Des valeurs toutes distinctes : on retrouve chaque cellule par sa valeur.
  value: Float32Array.from({ length: 24 }, (_, i) => i + 1),
  lo: 1,
  hi: 24,
  filled: 24,
}

function viewAt(turns: QuarterTurns): CanvasView {
  return { bounds: orientBounds(SCENE, turns), width: 600, height: 600, pad: 0, orient: viewOrientation(turns, SCENE) }
}

beforeEach(() => {
  drawHeat.mockReset()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
})
afterEach(() => vi.restoreAllMocks())

describe('la carte de chaleur suit l’orientation', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour, chaque cellule est au pixel de son centre projeté', (turns) => {
    const view = viewAt(turns)
    renderHook(() =>
      useReplayStaticLayers({
        view,
        redraw: () => {},
        zones: { zones: [], bigColors: [], fineInk: 'i', locale: 'fr' } as never,
        heat: { grid: GRID, ramp: ['#000000', '#ffffff'] },
        wallPlan: { plan: null, ink: 'w' },
        objectives: { elements: [], colorOfTeam: () => 'c', neutralOutline: 'o', z: { min: 0, max: 1 } },
      }),
    )
    expect(drawHeat).toHaveBeenCalledTimes(1)
    const [, grid, heatView] = drawHeat.mock.calls[0] as [unknown, HeatGrid, { topLeft: { x: number; y: number }; step: number }]
    expect(grid.nx * grid.ny).toBe(GRID.nx * GRID.ny)
    for (let row = 0; row < GRID.ny; row++) {
      for (let col = 0; col < GRID.nx; col++) {
        const attendu = projectTo(view, { x: (col + 0.5) * GRID.cell, y: (row + 0.5) * GRID.cell })
        // La cellule portant la même valeur dans la grille tournée.
        const idx = grid.value.indexOf(GRID.value[row * GRID.nx + col])
        const c2 = idx % grid.nx
        const r2 = Math.floor(idx / grid.nx)
        // La ligne 0 est la plus basse en monde, donc la plus basse à l'écran.
        const x = heatView.topLeft.x + (c2 + 0.5) * heatView.step
        const y = heatView.topLeft.y + (grid.ny - 1 - r2 + 0.5) * heatView.step
        expect(x).toBeCloseTo(attendu.x, 6)
        expect(y).toBeCloseTo(attendu.y, 6)
      }
    }
  })
})
