/**
 * Tests — useReliefView : le fond Relief, ses traits et la coupe par étage d'une vue 2D sans panneau « Vue ».
 *
 * Les fichiers (`useReliefFiles`) et le calculateur (`useReliefRender`) sont remplacés par des doubles : ce qui se
 * teste est la DÉCISION — quels fonds sont proposés, quand l'image des hauteurs est demandée, ce qui est posé à la
 * place du fond de base, et le repli sur le plan, journalisé, sur toute erreur du relief.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReliefFrame, ReliefRenderRequest } from './relief'
import { fourBlocksScene, reliefData } from './relief/reliefScene.testutil'
import type { ReliefFiles, ReliefFilesSource } from './reliefQueries'
import type { ReliefRender } from './useReliefRender'
import { useReliefView, type ReliefViewOptions } from './useReliefView'

const mocks = vi.hoisted(() => ({ files: vi.fn(), render: vi.fn() }))
vi.mock('./reliefQueries', () => ({ useReliefFiles: mocks.files }))
vi.mock('./useReliefRender', () => ({ useReliefRender: mocks.render }))
vi.mock('./relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => 'canvas-relief'
    maskedImage = (_f: unknown, _i: unknown, plan: boolean) => (plan ? 'canvas-masque-plan' : 'canvas-masque-rendu')
  },
}))

const SOURCE: ReliefFilesSource = { dataKey: ['d'], dataPath: '/d', imageKey: ['i'], imagePath: '/i' }
const BASE = 'image-de-base' as unknown as CanvasImageSource
const INK = { base: '#6b7078', outline: '#ffffff' }

function frame(over: Partial<ReliefFrame> = {}): ReliefFrame {
  const empty = { points: new Float32Array(0), offsets: new Uint32Array(1) }
  return { width: 40, height: 10, cut: null, image: new Uint8ClampedArray(4), alpha: null, planAlpha: null, lines: { steps: empty, contours: empty, walls: empty }, ...over }
}

function files(over: Partial<ReliefFiles> = {}): ReliefFiles {
  return { data: fourBlocksScene().data, blob: new Blob(), loading: false, error: null, ...over }
}

function options(over: Partial<ReliefViewOptions> = {}): ReliefViewOptions {
  return {
    files: SOURCE,
    cut: null,
    onCut: vi.fn(),
    style: 'plan',
    wanted: 'plan',
    planAvailable: true,
    linesOn: false,
    turns: 0,
    ink: INK,
    baseImage: BASE,
    baseIsPlan: true,
    ...over,
  }
}

const lastRequest = (): ReliefRenderRequest | null => mocks.render.mock.calls.at(-1)?.[2] ?? null
const lastNeedImage = (): boolean => mocks.files.mock.calls.at(-1)?.[1]

beforeEach(() => {
  mocks.files.mockReset().mockReturnValue(files())
  mocks.render
    .mockReset()
    .mockReturnValue({ status: 'ready', info: { width: 40, height: 10, hasMesh: true }, frame: frame(), error: null } satisfies ReliefRender)
})

afterEach(() => vi.restoreAllMocks())

describe('les fonds proposés', () => {
  it('Plan, Rendu, Relief quand la carte offre un plan et un relief lisible', () => {
    const { result } = renderHook(() => useReliefView(options()))
    expect(result.current.options).toEqual(['plan', 'rendu', 'relief'])
    expect(result.current.selected).toBe('plan')
  })

  it('une carte sans relief offert (files nul) ne propose que Plan, Rendu, et ne demande aucune image', () => {
    const { result } = renderHook(() => useReliefView(options({ files: null })))
    expect(result.current.options).toEqual(['plan', 'rendu'])
    expect(lastNeedImage()).toBe(false)
    expect(lastRequest()).toBeNull()
  })

  it('le relief voulu est celui qui est retenu quand il est lisible', () => {
    const { result } = renderHook(() => useReliefView(options({ style: 'relief', wanted: 'relief' })))
    expect(result.current.selected).toBe('relief')
    expect(result.current.surface?.image).toBe('canvas-relief')
    expect(result.current.reliefShown).toBe(true)
  })
})

describe('l’image des hauteurs n’est demandée que quand une commande s’en sert', () => {
  it('fond Plan, pas de traits, pas de coupe : non', () => {
    renderHook(() => useReliefView(options()))
    expect(lastNeedImage()).toBe(false)
  })

  it('fond Relief : oui, avec la base de teinte', () => {
    renderHook(() => useReliefView(options({ style: 'relief', wanted: 'relief' })))
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()?.base).toBe(INK.base)
  })

  it('traits allumés : oui, sans teinte', () => {
    renderHook(() => useReliefView(options({ linesOn: true })))
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()).toMatchObject({ base: null, lines: true, masks: false })
  })

  it('coupe choisie sur un fond Plan : oui, et les masques sont demandés', () => {
    renderHook(() => useReliefView(options({ cut: 2.2 })))
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()).toMatchObject({ cut: 2.2, masks: true })
  })
})

describe('ce qui prend la place du fond de base', () => {
  it('sous coupe, le plan est modulé par le masque du plan', () => {
    const { result } = renderHook(() => useReliefView(options({ cut: 2.2 })))
    expect(result.current.surface?.image).toBe('canvas-masque-plan')
    expect(result.current.reliefShown).toBe(false)
  })

  it('sous coupe, le rendu est modulé par le masque du rendu', () => {
    const { result } = renderHook(() => useReliefView(options({ cut: 2.2, style: 'rendu', wanted: 'rendu', baseIsPlan: false })))
    expect(result.current.surface?.image).toBe('canvas-masque-rendu')
  })

  it('sans coupe, le fond de base reste seul posé', () => {
    const { result } = renderHook(() => useReliefView(options()))
    expect(result.current.surface).toBeNull()
  })

  it('sous coupe mais le fond de base pas encore décodé : le fond de base reste posé', () => {
    const { result } = renderHook(() => useReliefView(options({ cut: 2.2, baseImage: null })))
    expect(result.current.surface).toBeNull()
  })

  it('une nouvelle frame change la surface, même quand le canvas réutilisé est le même', () => {
    const { result, rerender } = renderHook(() => useReliefView(options({ cut: 2.2 })))
    const first = result.current.surface
    mocks.render.mockReturnValue({ status: 'ready', info: null, frame: frame({ cut: 2.2 }), error: null })
    rerender()
    expect(result.current.surface?.image).toBe(first?.image)
    expect(result.current.surface).not.toBe(first)
  })
})

describe('repli sur le plan, journalisé', () => {
  it('une erreur de lecture des fichiers : le plan est retenu, le relief n’est plus proposé, un journal est écrit', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    mocks.files.mockReturnValue(files({ data: null, blob: null, error: new Error('version inconnue') }))
    const { result } = renderHook(() => useReliefView(options({ style: 'relief', wanted: 'relief' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.options).toEqual(['plan', 'rendu'])
    expect(result.current.surface).toBeNull()
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('[relief-view]'))
    expect(warn.mock.calls[0][0]).toContain('version inconnue')
  })

  it('une panne du calculateur : même repli', () => {
    mocks.render.mockReturnValue({ status: 'failed', info: null, frame: null, error: new Error('worker') } satisfies ReliefRender)
    const { result } = renderHook(() => useReliefView(options({ style: 'relief', wanted: 'relief' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.options).not.toContain('relief')
    expect(result.current.surface).toBeNull()
  })

  it('sans image plan non plus, le repli est le rendu', () => {
    mocks.files.mockReturnValue(files({ data: null, blob: null, error: new Error('x') }))
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { result } = renderHook(() => useReliefView(options({ style: 'relief', wanted: 'relief', planAvailable: false })))
    expect(result.current.selected).toBe('rendu')
    expect(result.current.options).toEqual([])
  })
})

describe('la commande d’étage', () => {
  it('des étages : les rangs et le maillage pour juger des positions', () => {
    const { result } = renderHook(() => useReliefView(options()))
    expect(result.current.floor.state.state).toBe('ready')
    expect(result.current.mesh).not.toBeNull()
    expect(result.current.meshPercent).toBe(25)
  })

  it('carte sans maillage : aucune commande d’étage, ni maillage ; le relief et ses traits restent', () => {
    mocks.files.mockReturnValue(files({ data: reliefData({ width: 4, height: 4, degradations: ['sans-maillage'] }) }))
    const { result } = renderHook(() => useReliefView(options()))
    expect(result.current.floor.state).toEqual({ state: 'noMesh' })
    expect(result.current.floor.cut).toBeNull()
    expect(result.current.mesh).toBeNull()
    expect(result.current.options).toContain('relief')
    expect(result.current.linesAvailable).toBe(true)
  })

  it('les gestes passent par onCut : étage exact, hauteur libre arrondie au quart d’unité, « Tout »', () => {
    const onCut = vi.fn()
    const { result } = renderHook(() => useReliefView(options({ onCut })))
    result.current.floor.selectRank(2.2)
    result.current.floor.setHeight(1.13)
    result.current.floor.selectAll()
    expect(onCut.mock.calls).toEqual([[2.2], [1.25], [null]])
  })

  it('relief pas encore lu : aucune commande, aucun trait proposé', () => {
    mocks.files.mockReturnValue(files({ data: null, blob: null, loading: true }))
    const { result } = renderHook(() => useReliefView(options()))
    expect(result.current.floor.state).toEqual({ state: 'none' })
    expect(result.current.linesAvailable).toBe(false)
  })
})

describe('les traits', () => {
  it('éteints : aucun calque de traits', () => {
    const { result } = renderHook(() => useReliefView(options({ linesOn: false })))
    expect(result.current.linesLayer).toBeNull()
  })

  it('allumés, une fois la frame rendue : le calque porte le calage, le liseré et les encres', () => {
    const { result } = renderHook(() => useReliefView(options({ linesOn: true })))
    expect(result.current.linesLayer?.calibration.widthPx).toBe(40)
    expect(result.current.linesLayer?.ink.outline).toBe(INK.outline)
  })
})
