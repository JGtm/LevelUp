/**
 * useReplayStaticLayers.plan.test.tsx — les FORMES des zones nommées se rognent à l'alpha de l'image plan
 * quand c'est elle qui est posée (`lib/replay/planMask`) ; les libellés restent entiers ; sans plan posé, le
 * calque est celui d'avant (aucune composition, aucune image).
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import type { CanvasView } from '../model/replayView'
import { useReplayStaticLayers } from './useReplayStaticLayers'

const SCENE: ReplayBounds = { minX: 0, maxX: 100, minY: 0, maxY: 100 }
const VIEW: CanvasView = { bounds: SCENE, width: 400, height: 400, pad: 0 }
const ZONE = normalizeCalloutZones([
  {
    name: 'a',
    fr: 'Fosse',
    en: 'Pit',
    volume_index: 1,
    big: true,
    polygon: [
      [10, 10],
      [60, 10],
      [60, 60],
      [10, 60],
    ],
    x: 35,
    y: 35,
    z: 0,
    z_bottom: -1,
    z_top: 1,
  },
])
const PLAN = {
  calibration: { metersPerPixel: 1, originX: 0, originY: 100, widthPx: 100, heightPx: 100, convention: 'xy' },
  image: {} as CanvasImageSource,
}

let ops: CanvasOp[] = []

beforeEach(() => {
  ops = []
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => {
    const enregistre = recordingContext()
    ops = enregistre.ops
    return enregistre.ctx as never
  })
})
afterEach(() => vi.restoreAllMocks())

function cuire(plan: typeof PLAN | null | undefined) {
  renderHook(() =>
    useReplayStaticLayers({
      view: VIEW,
      redraw: () => {},
      zones: { zones: ZONE, bigColors: ['c'], fineInk: 'i', locale: 'fr', ...(plan === undefined ? {} : { plan }) },
      heat: { grid: null, ramp: [] },
      wallPlan: { plan: null, ink: 'w' },
      objectives: { elements: [], colorOfTeam: () => 'c', neutralOutline: 'o', z: { min: 0, max: 1 } },
    }),
  )
  return ops.map((o) => o.op)
}

describe('calque des zones nommées — rognage au plan', () => {
  it('avec le plan posé : formes, puis plan en destination-in, puis libellés', () => {
    const noms = cuire(PLAN)
    const composition = noms.indexOf('set globalCompositeOperation')
    const image = noms.indexOf('drawImage')
    const forme = noms.indexOf('fill')
    const libelle = noms.indexOf('fillText')
    expect(forme).toBeGreaterThanOrEqual(0)
    expect(composition).toBeGreaterThan(forme)
    expect(image).toBeGreaterThan(composition)
    expect(libelle).toBeGreaterThan(image)
    expect(ops.find((o) => o.op === 'set globalCompositeOperation')?.args[0]).toBe('destination-in')
    expect(noms.slice(image)).toContain('restore')
  })

  it.each([
    ['plan absent du réglage', undefined],
    ['plan à null (rendu posé, ou carte sans vue plan)', null],
  ])('%s : zones entières, ni composition ni image', (_nom, plan) => {
    const noms = cuire(plan)
    expect(noms).toContain('fillText')
    expect(noms).not.toContain('set globalCompositeOperation')
    expect(noms).not.toContain('drawImage')
  })
})
