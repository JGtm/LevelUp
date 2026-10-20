/**
 * useReplayView.test.ts — LE CADRAGE PENDANT UN EXPORT (2026-09-16).
 *
 * Ce qui se verrouille : pendant un export, la taille de DESSIN et la projection prennent le
 * cadre 16:9 du format, quelle que soit la place que l'ecran offre — mais la BOITE D'ECRAN, elle,
 * ne bouge pas (la page ne saute pas). Et tout revient a l'ecran quand l'export se retire.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { canvasPixelRatio, isExportLayoutApplied, requestExportLayout } from '../export/exportLayoutStore'
import { testReplayDoc } from '../test/testDoc'
import { useReplayView } from './useReplayView'

const DOC = testReplayDoc({ frameIntervalMs: 50, frameCount: 10 })

afterEach(() => {
  act(() => requestExportLayout(null))
})

describe('useReplayView — la mise en page d’export', () => {
  it('dessine au cadre du format, garde la boite d’ecran, puis revient', () => {
    const { result } = renderHook(() => useReplayView({ doc: DOC, width: 1200, freeHeight: 600 }))
    const ecran = { width: result.current.renderWidth, height: result.current.renderHeight }
    expect(result.current.screen).toEqual(ecran)

    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    expect(result.current.renderWidth).toBe(960)
    expect(result.current.renderHeight).toBe(540)
    expect(result.current.canvasView).toMatchObject({ width: 960, height: 540 })
    // LA PROJECTION EST ELARGIE AU 16:9, PAS ETIREE : le cadre du monde a les proportions de la
    // zone de dessin utile (hors marge), comme a l'ecran.
    const b = result.current.canvasView.bounds
    const pad = result.current.canvasView.pad
    expect((b.maxX - b.minX) / (b.maxY - b.minY)).toBeCloseTo((960 - 2 * pad) / (540 - 2 * pad), 6)
    expect(result.current.screen).toEqual(ecran)
    expect(isExportLayoutApplied()).toBe(true)
    expect(canvasPixelRatio()).toBeCloseTo(4 / 3, 12)

    act(() => requestExportLayout(null))
    expect({ width: result.current.renderWidth, height: result.current.renderHeight }).toEqual(ecran)
    expect(canvasPixelRatio()).toBe(window.devicePixelRatio || 1)
  })

  it('recree le cadrage meme si les tailles ne changent pas : les calques cuits changent de densite', () => {
    // Une toile d'ecran qui ferait deja 960x540 : seule la densite differe a l'export.
    const { result } = renderHook(() => useReplayView({ doc: DOC, width: 960, freeHeight: 540 }))
    const avant = result.current.canvasView
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('1080p'))))
    expect(result.current.canvasView).not.toBe(avant)
  })
})

describe('useReplayView — le cadrage et les gestes pendant l’export (D6/D7)', () => {
  const span = (b: { minX: number; maxX: number }) => b.maxX - b.minX

  function zoomeADeux() {
    const hook = renderHook(() => useReplayView({ doc: DOC, width: 1200, freeHeight: 600 }))
    act(() => hook.result.current.zoom.zoomIn()) // 1.5x
    act(() => hook.result.current.zoom.zoomIn()) // 2x
    act(() => hook.result.current.zoom.panStep(1, 1))
    expect(hook.result.current.zoom.level).toBe(2)
    return hook
  }

  it('« carte entiere » rend a 1x SANS toucher au zoom, qui revient intact', () => {
    const { result } = zoomeADeux()
    const centreAvant = result.current.zoom.center
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('1080p'), 'current')))
    const cadrageActuel = result.current.canvasView.bounds
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('1080p'), 'whole')))
    // La fenetre est DEUX fois plus large : la carte entiere, pas le palier de l'ecran.
    expect(span(result.current.canvasView.bounds)).toBeCloseTo(2 * span(cadrageActuel), 6)
    expect(result.current.zoom.level).toBe(2)
    act(() => requestExportLayout(null))
    expect(result.current.zoom.level).toBe(2)
    expect(result.current.zoom.center).toEqual(centreAvant)
  })

  it('« cadrage actuel » garde palier et centre dans le cadre 16:9', () => {
    const { result } = zoomeADeux()
    const { center } = result.current.zoom
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'), 'current')))
    const b = result.current.canvasView.bounds
    expect((b.minX + b.maxX) / 2).toBeCloseTo(center.x, 6)
    expect((b.minY + b.maxY) / 2).toBeCloseTo(center.y, 6)
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'), 'whole')))
    expect(span(b) * 2).toBeCloseTo(span(result.current.canvasView.bounds), 6)
  })

  it('les gestes de cadrage sont SANS EFFET pendant l’export, et reviennent apres', () => {
    const { result } = zoomeADeux()
    const centre = result.current.zoom.center
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('1080p'), 'current')))
    const z = result.current.zoom
    expect([z.canZoomIn, z.canZoomOut, z.canPan]).toEqual([false, false, false])
    act(() => {
      z.zoomIn()
      z.zoomOut()
      z.reset()
      z.panStep(1, 0)
      z.panBy(100, 100)
      z.zoomAt(-1, { x: 0, y: 0 })
    })
    expect(result.current.zoom.level).toBe(2)
    expect(result.current.zoom.center).toEqual(centre)
    act(() => requestExportLayout(null))
    expect(result.current.zoom.canZoomOut).toBe(true)
    act(() => result.current.zoom.zoomOut())
    expect(result.current.zoom.level).toBe(1.5)
  })
})
