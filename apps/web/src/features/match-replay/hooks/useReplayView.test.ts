/**
 * useReplayView.test.ts — LE CADRAGE PENDANT UN EXPORT (2026-09-16).
 *
 * Ce qui se verrouille : pendant un export, la taille de DESSIN et la projection prennent le
 * cadre 16:9 du format, quelle que soit la place que l'ecran offre — mais la BOITE D'ECRAN, elle,
 * ne bouge pas (la page ne saute pas). Et tout revient a l'ecran quand l'export se retire.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { canvasPixelRatio, isExportLayoutApplied, requestExportLayout } from '../export/exportLayoutStore'
import { canvasToWorld, worldToCanvas } from '../../../lib/replay/replayLogic'
import { orientWorld, QUARTER_TURNS, unorientWorld, type QuarterTurns } from '../model/orientation'
import { projectTo } from '../model/replayView'
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

describe('useReplayView — la main reprend le cadrage (Suivre, lot 4)', () => {
  it('un geste manuel prévient l’écouteur ; le cadrage programmé, non', () => {
    const onManualMove = vi.fn()
    const { result } = renderHook(() => useReplayView({ doc: DOC, width: 1200, freeHeight: 600, onManualMove }))
    act(() => result.current.zoom.focusOn({ x: 5, y: 5 }, 2))
    expect(result.current.zoom.level).toBe(2)
    expect(onManualMove).not.toHaveBeenCalled()
    act(() => result.current.zoom.panStep(1, 0))
    expect(onManualMove).toHaveBeenCalledTimes(1)
    act(() => result.current.zoom.zoomIn())
    expect(onManualMove).toHaveBeenCalledTimes(2)
  })

  it('pendant un export, ni geste ni cadrage programmé ne bougent la carte', () => {
    const { result } = renderHook(() => useReplayView({ doc: DOC, width: 1200, freeHeight: 600 }))
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('1080p'), 'current')))
    act(() => result.current.zoom.focusOn({ x: 5, y: 5 }, 2))
    expect(result.current.zoom.level).toBe(1)
  })
})

describe('useReplayView — l’orientation de la carte', () => {
  // Une carte allongée : la rotation d'un quart de tour échange sa largeur et sa hauteur.
  const LONG = testReplayDoc({
    frameIntervalMs: 50,
    frameCount: 10,
    bounds: { minX: 0, minY: 0, maxX: 40, maxY: 10 },
  })
  const SCENE = LONG.bounds
  const POINTS = [
    { x: 0, y: 0 },
    { x: 40, y: 10 },
    { x: 12, y: 7.5 },
    { x: 33, y: 2 },
  ]

  it.each(QUARTER_TURNS)('à %i quart(s) de tour, un point monde tombe au pixel que worldToCanvas donne, et l’inverse le rend', (turns) => {
    const { result } = renderHook(() => useReplayView({ doc: LONG, turns, width: 1200, freeHeight: 600 }))
    const v = result.current.canvasView
    for (const p of POINTS) {
      const turned = orientWorld(p, turns, SCENE)
      const attendu = worldToCanvas(turned, v.bounds, v.width, v.height, v.pad)
      const got = projectTo(v, p)
      expect(got.x).toBeCloseTo(attendu.x, 9)
      expect(got.y).toBeCloseTo(attendu.y, 9)
      // L'inverse : du pixel au monde, puis défaire la rotation.
      const back = unorientWorld(canvasToWorld(got, v.bounds, v.width, v.height, v.pad), turns, SCENE)
      expect(back.x).toBeCloseTo(p.x, 9)
      expect(back.y).toBeCloseTo(p.y, 9)
    }
  })

  it('un quart de tour échange la largeur et la hauteur de la scène cadrée, autour du même centre', () => {
    const droit = renderHook(() => useReplayView({ doc: LONG, turns: 0, width: 1200, freeHeight: 600 }))
    const tourne = renderHook(() => useReplayView({ doc: LONG, turns: 1, width: 1200, freeHeight: 600 }))
    const a = droit.result.current.canvasView.bounds
    const b = tourne.result.current.canvasView.bounds
    expect((a.minX + a.maxX) / 2).toBeCloseTo((b.minX + b.maxX) / 2, 9)
    expect((a.minY + a.maxY) / 2).toBeCloseTo((b.minY + b.maxY) / 2, 9)
    // La carte tournée est plus haute que large : la hauteur utile suit, elle n'est plus celle d'avant.
    expect(tourne.result.current.renderHeight).toBeGreaterThan(droit.result.current.renderHeight)
    // `bounds` reste le repère MONDE des données du film.
    expect(tourne.result.current.bounds).toEqual(droit.result.current.bounds)
  })

  it('l’export garde son cadre 16:9 quelle que soit l’orientation', () => {
    for (const turns of QUARTER_TURNS) {
      const { result } = renderHook(() => useReplayView({ doc: LONG, turns, width: 1200, freeHeight: 600 }))
      act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
      expect(result.current.renderWidth).toBe(960)
      expect(result.current.renderHeight).toBe(540)
      const v = result.current.canvasView
      expect(v).toMatchObject({ width: 960, height: 540 })
      expect((v.bounds.maxX - v.bounds.minX) / (v.bounds.maxY - v.bounds.minY)).toBeCloseTo(
        (960 - 2 * v.pad) / (540 - 2 * v.pad),
        6,
      )
      // La carte entière tient dans le cadre tourné, et la rotation est bien celle de la page.
      const coin = projectTo(v, { x: SCENE.minX, y: SCENE.minY })
      expect(coin.x).toBeGreaterThanOrEqual(0)
      expect(coin.x).toBeLessThanOrEqual(960)
      expect(coin.y).toBeGreaterThanOrEqual(0)
      expect(coin.y).toBeLessThanOrEqual(540)
      act(() => requestExportLayout(null))
    }
  })

  it('changer d’orientation revient à la carte entière', () => {
    const { result, rerender } = renderHook(
      ({ turns }: { turns: QuarterTurns }) => useReplayView({ doc: LONG, turns, width: 1200, freeHeight: 600 }),
      { initialProps: { turns: 0 as QuarterTurns } },
    )
    act(() => result.current.zoom.zoomIn())
    act(() => result.current.zoom.zoomIn())
    expect(result.current.zoom.level).toBe(2)
    rerender({ turns: 1 })
    expect(result.current.zoom.level).toBe(1)
  })

  it('la molette garde le point visé immobile à l’écran, sous toute orientation', () => {
    for (const turns of QUARTER_TURNS) {
      const { result } = renderHook(() => useReplayView({ doc: LONG, turns, width: 1200, freeHeight: 600 }))
      const v = result.current.canvasView
      const visee = { x: v.width * 0.7, y: v.height * 0.4 }
      // Ce que fait `useReplayWheelZoom` : le point de la fenêtre sous le pointeur, dans le repère de la carte tournée.
      const sous = canvasToWorld(visee, v.bounds, v.width, v.height, v.pad)
      const monde = unorientWorld(sous, turns, SCENE)
      act(() => result.current.zoom.zoomAt(1, sous))
      const apres = projectTo(result.current.canvasView, monde)
      expect(apres.x).toBeCloseTo(visee.x, 6)
      expect(apres.y).toBeCloseTo(visee.y, 6)
    }
  })

  it('le cadrage programmé (Suivre) parle monde : le joueur suivi tombe au centre de la toile', () => {
    for (const turns of QUARTER_TURNS) {
      const { result } = renderHook(() => useReplayView({ doc: LONG, turns, width: 1200, freeHeight: 600 }))
      const joueur = { x: 21, y: 6 }
      act(() => result.current.zoom.focusOn(joueur, 2))
      const v = result.current.canvasView
      const px = projectTo(v, joueur)
      expect(px.x).toBeCloseTo(v.width / 2, 6)
      expect(px.y).toBeCloseTo(v.height / 2, 6)
    }
  })
})
