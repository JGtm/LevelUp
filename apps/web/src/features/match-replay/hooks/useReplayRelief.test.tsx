/**
 * Tests — useReplayRelief : le fond Relief, ses traits et la coupe par étage assemblés.
 *
 * Les fichiers (`useReplayReliefFiles`) et le calculateur (`useReliefRender`) sont remplacés par des doubles :
 * ce qui se teste ici est la DÉCISION — quel fond est posé et proposé, quand l'image des hauteurs est demandée,
 * ce qui se passe sur une `ReliefError`, comment la coupe et les traits se retiennent, et surtout que la coupe ne
 * touche à AUCUN joueur (le hook ne rend ni filtre ni masque de position).
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import { fourBlocksScene, reliefData } from '@/lib/replay/relief/reliefScene.testutil'
import type { ReliefFiles } from '@/lib/replay/reliefQueries'
import type { ReliefRender } from '@/lib/replay/useReliefRender'
import type { ReliefFrame, ReliefRenderRequest } from '@/lib/replay/relief'

import type { ReplayMapBackgroundLayer } from './useReplayView'
import { testReplayDoc } from '../test/testDoc'
import { useReplayRelief, type ReplayReliefOptions } from './useReplayRelief'

const mocks = vi.hoisted(() => ({
  files: vi.fn(),
  render: vi.fn(),
}))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReplayReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))
vi.mock('@/lib/replay/relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => 'canvas-relief'
    maskedImage = (_f: unknown, _i: unknown, plan: boolean) => (plan ? 'canvas-masque-plan' : 'canvas-masque-rendu')
  },
}))

const BASE_IMAGE = 'image-de-base' as unknown as CanvasImageSource
const CAL = { widthPx: 40, heightPx: 10, metersPerPixel: 0.1, originX: 0, originY: 1 }

function baseLayer(over: Partial<ReplayMapBackgroundLayer> = {}): ReplayMapBackgroundLayer {
  return {
    calibration: CAL as never,
    image: BASE_IMAGE,
    plan: null,
    variante: 'plan',
    planAvailable: true,
    reliefAvailable: true,
    zonesClip: null,
    ...over,
  }
}

const INK = { base: '#6b7078', outline: '#ffffff' }
const Z = { min: 0, max: 5 }
/** Le suivi du joueur ouvert, sans joueur ouvert : les tests du suivi sont dans `useReplayRelief.follow.test.tsx`. */
const NO_FOLLOW = { doc: testReplayDoc(), player: null, frameRef: { current: 0 } }

function options(over: Partial<ReplayReliefOptions> = {}): ReplayReliefOptions {
  return { base: baseLayer(), playerSlug: 'jg', matchId: 'm1', scopeKey: 'carte-a', style: 'plan', turns: 0, zRange: Z, ink: INK, follow: NO_FOLLOW, ...over }
}

function frame(over: Partial<ReliefFrame> = {}): ReliefFrame {
  const empty = { points: new Float32Array(0), offsets: new Uint32Array(1) }
  return { width: 40, height: 10, cut: null, image: new Uint8ClampedArray(4), alpha: null, planAlpha: null, lines: { steps: empty, contours: empty, walls: empty }, ...over }
}

function ready(over: Partial<ReliefFiles> = {}): ReliefFiles {
  return { data: fourBlocksScene().data, blob: new Blob(), loading: false, error: null, ...over }
}

function rendered(over: Partial<ReliefRender> = {}): ReliefRender {
  return { status: 'ready', info: { width: 40, height: 10, hasMesh: true }, frame: frame(), error: null, ...over }
}

/** La dernière demande de rendu que le hook a posée au calculateur. */
const lastRequest = (): ReliefRenderRequest | null => mocks.render.mock.calls.at(-1)?.[2] ?? null
/** Le dernier `needImage` demandé à la lecture des fichiers. */
const lastNeedImage = (): boolean => mocks.files.mock.calls.at(-1)?.[3]

beforeEach(() => {
  window.localStorage.clear()
  mocks.files.mockReset()
  mocks.render.mockReset()
  mocks.files.mockReturnValue(ready())
  mocks.render.mockReturnValue(rendered())
  vi.spyOn(console, 'warn').mockImplementation(() => {})
})

describe('useReplayRelief — le fond posé et proposé', () => {
  it('le Relief voulu et lisible devient le fond : variante relief, image teintée, les trois fonds proposés', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.selected).toBe('relief')
    expect(result.current.options).toEqual(['plan', 'rendu', 'relief'])
    expect(result.current.background).toMatchObject({ image: 'canvas-relief', variante: 'relief' })
  })

  it('le Relief voulu mais pas encore rendu : le fond de base reste posé', () => {
    mocks.render.mockReturnValue(rendered({ status: 'loading', frame: null, info: null }))
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.selected).toBe('relief')
    expect(result.current.background?.image).toBe(BASE_IMAGE)
    expect(result.current.background?.variante).toBe('plan')
  })

  it('le Plan voulu reste le Plan, même quand le relief est lisible ; le fond de base est rendu tel quel', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.background?.image).toBe(BASE_IMAGE)
  })

  it('un fond sans relief ne propose pas le Relief et retombe sur le plan', () => {
    mocks.files.mockReturnValue({ data: null, blob: null, loading: false, error: null })
    mocks.render.mockReturnValue({ status: 'idle', info: null, frame: null, error: null })
    const { result } = renderHook(() => useReplayRelief(options({ base: baseLayer({ reliefAvailable: false }), style: 'relief' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.options).toEqual(['plan', 'rendu'])
    expect(mocks.files.mock.calls.at(-1)?.[2]).toBe(false)
    expect(result.current.floor.state).toEqual({ state: 'none' })
    expect(result.current.lines.available).toBe(false)
  })

  it('sans plan ni relief, rien n’est proposé et le rendu est servi', () => {
    mocks.files.mockReturnValue({ data: null, blob: null, loading: false, error: null })
    mocks.render.mockReturnValue({ status: 'idle', info: null, frame: null, error: null })
    const base = baseLayer({ planAvailable: false, reliefAvailable: false, variante: 'rendu' })
    const { result } = renderHook(() => useReplayRelief(options({ base, style: 'relief' })))
    expect(result.current.selected).toBe('rendu')
    expect(result.current.options).toEqual([])
  })

  it('pas de fond de base : pas de fond', () => {
    const { result } = renderHook(() => useReplayRelief(options({ base: null })))
    expect(result.current.background).toBeNull()
  })
})

describe('useReplayRelief — une ReliefError replie sur le Plan', () => {
  it('un calculateur en échec : le Plan est posé, le Relief n’est plus proposé, rien de relief ne reste', () => {
    mocks.render.mockReturnValue({ status: 'failed', info: null, frame: null, error: new Error('x') })
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.options).toEqual(['plan', 'rendu'])
    expect(result.current.background?.image).toBe(BASE_IMAGE)
    expect(result.current.floor.state).toEqual({ state: 'none' })
    expect(result.current.lines.available).toBe(false)
    expect(result.current.linesLayer).toBeNull()
    expect(result.current.meshPercent).toBeNull()
  })

  it('une lecture de fichiers en erreur (fichier absent, version inconnue) : même repli, et un journal', () => {
    mocks.files.mockReturnValue({ data: null, blob: null, loading: false, error: new Error('data-version') })
    mocks.render.mockReturnValue({ status: 'idle', info: null, frame: null, error: null })
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.selected).toBe('plan')
    expect(result.current.options).toEqual(['plan', 'rendu'])
    expect(console.warn).toHaveBeenCalledWith(expect.stringContaining('[replay-relief]'))
  })

  it('un échec sans plan retombe sur le rendu', () => {
    mocks.render.mockReturnValue({ status: 'failed', info: null, frame: null, error: new Error('x') })
    const base = baseLayer({ planAvailable: false, variante: 'rendu' })
    const { result } = renderHook(() => useReplayRelief(options({ base, style: 'relief' })))
    expect(result.current.selected).toBe('rendu')
    expect(result.current.options).toEqual([])
  })
})

describe('useReplayRelief — quand l’image des hauteurs est demandée', () => {
  it('jamais sur un fond Plan sans traits ni coupe : les données seules, aucune demande de rendu', () => {
    renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(lastNeedImage()).toBe(false)
    expect(lastRequest()).toBeNull()
  })

  it('sur le fond Relief : oui, avec la teinte, les traits allumés d’office et aucun masque', () => {
    renderHook(() => useReplayRelief(options({ style: 'relief', turns: 3 })))
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()).toEqual({ turns: 3, cut: null, base: '#6b7078', lines: true, masks: false })
  })

  it('sur un fond Plan, allumer les traits la demande, sans teinte ni masque', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    act(() => result.current.lines.onToggle())
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()).toEqual({ turns: 0, cut: null, base: null, lines: true, masks: false })
  })

  it('sur un fond Plan, choisir un étage la demande, avec les masques de la coupe', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    act(() => result.current.floor.selectRank(2.2))
    expect(lastNeedImage()).toBe(true)
    expect(lastRequest()).toEqual({ turns: 0, cut: 2.2, base: null, lines: false, masks: true })
  })

  it('sans relief au sidecar : jamais demandée', () => {
    renderHook(() => useReplayRelief(options({ base: baseLayer({ reliefAvailable: false }), style: 'relief' })))
    expect(lastNeedImage()).toBe(false)
  })
})

describe('useReplayRelief — les traits du relief, retenus par fond', () => {
  it('allumés sur le Relief, éteints sur le Plan et le Rendu', () => {
    const relief = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(relief.result.current.lines.show).toBe(true)
    const plan = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(plan.result.current.lines.show).toBe(false)
    const rendu = renderHook(() => useReplayRelief(options({ style: 'rendu' })))
    expect(rendu.result.current.lines.show).toBe(false)
  })

  it('un geste sur le Plan ne touche pas le Relief', () => {
    const plan = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    act(() => plan.result.current.lines.onToggle())
    expect(plan.result.current.lines.show).toBe(true)
    const relief = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(relief.result.current.lines.show).toBe(true)
    act(() => relief.result.current.lines.onToggle())
    expect(relief.result.current.lines.show).toBe(false)
    expect(renderHook(() => useReplayRelief(options({ style: 'plan' }))).result.current.lines.show).toBe(true)
  })

  it('le calque des traits porte les traits de la frame, le liseré du fichier et les encres tirées de la base', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    const layer = result.current.linesLayer
    expect(layer).not.toBeNull()
    expect(layer?.calibration).toEqual(fourBlocksScene().data.calibration)
    expect(layer?.outline).toEqual(fourBlocksScene().data.lines.outline)
    expect(layer?.ink.outline).toBe('#ffffff')
    expect(layer?.ink.edge).toMatch(/^rgb\(/)
  })

  it('éteints : pas de calque de traits', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(result.current.linesLayer).toBeNull()
  })

  it('pas de calque tant que le calculateur n’a rien rendu', () => {
    mocks.render.mockReturnValue(rendered({ status: 'loading', frame: null, info: null }))
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.linesLayer).toBeNull()
  })

  it('une base de teinte illisible (jeton non résolu) : pas de calque de traits, sans lever', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief', ink: { base: '', outline: '#fff' } })))
    expect(result.current.linesLayer).toBeNull()
  })
})

describe('useReplayRelief — la coupe par étage', () => {
  it('« Tout » par défaut ; les pastilles viennent des étages détectés, en rangs', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(result.current.floor.cut).toBeNull()
    const state = result.current.floor.state
    expect(state.state === 'ready' && state.chips.map((c) => c.rank)).toEqual([0, 1])
  })

  it('une pastille retient la hauteur exacte de l’étage, un curseur une hauteur arrondie au quart d’unité, « Tout » la rend', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    act(() => result.current.floor.selectRank(2.2))
    expect(result.current.floor.cut).toBe(2.2)
    act(() => result.current.floor.setHeight(1.3))
    expect(result.current.floor.cut).toBe(1.25)
    act(() => result.current.floor.selectAll())
    expect(result.current.floor.cut).toBeNull()
  })

  it('une autre carte repart de « Tout »', () => {
    const { result, rerender } = renderHook((o: ReplayReliefOptions) => useReplayRelief(o), { initialProps: options({ style: 'plan' }) })
    act(() => result.current.floor.selectRank(2.2))
    expect(result.current.floor.cut).toBe(2.2)
    rerender(options({ style: 'plan', scopeKey: 'carte-b' }))
    expect(result.current.floor.cut).toBeNull()
  })

  it('un fond Plan coupé est modulé par le masque du plan, un fond Rendu par celui du rendu', () => {
    mocks.render.mockReturnValue(rendered({ frame: frame({ cut: 2.2, alpha: new Uint8Array(1), planAlpha: new Uint8Array(1) }) }))
    const plan = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    act(() => plan.result.current.floor.selectRank(2.2))
    expect(plan.result.current.background).toMatchObject({ image: 'canvas-masque-plan', variante: 'plan' })
    const rendu = renderHook(() => useReplayRelief(options({ base: baseLayer({ variante: 'rendu' }), style: 'rendu' })))
    act(() => rendu.result.current.floor.selectRank(2.2))
    expect(rendu.result.current.background).toMatchObject({ image: 'canvas-masque-rendu', variante: 'rendu' })
  })

  it('sans coupe, le fond de base n’est pas touché', () => {
    mocks.render.mockReturnValue(rendered({ frame: frame({ alpha: new Uint8Array(1), planAlpha: new Uint8Array(1) }) }))
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(result.current.background?.image).toBe(BASE_IMAGE)
  })

  it('carte sans maillage : l’état le dit, aucune coupe ne s’applique, les traits restent proposés', () => {
    mocks.files.mockReturnValue(ready({ data: reliefData({ width: 40, height: 10, degradations: ['sans-maillage'] }) }))
    const { result } = renderHook(() => useReplayRelief(options({ style: 'plan' })))
    expect(result.current.floor.state).toEqual({ state: 'noMesh' })
    expect(result.current.floor.cut).toBeNull()
    expect(result.current.lines.available).toBe(true)
  })

  it('la part du relief tirée du maillage vient du fichier (pourcentage entier)', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    expect(result.current.meshPercent).toBe(25)
  })
})

describe('useReplayRelief — la coupe ne masque AUCUN joueur', () => {
  it('le hook ne rend que des fonds, des traits et des commandes : aucun filtre ni masque de position ou de joueur', () => {
    const { result } = renderHook(() => useReplayRelief(options({ style: 'relief' })))
    act(() => result.current.floor.selectRank(2.2))
    expect(Object.keys(result.current).sort()).toEqual(['background', 'floor', 'followStep', 'linesLayer', 'lines', 'meshPercent', 'options', 'selected'].sort())
    expect(Object.keys(result.current.floor).sort()).toEqual(['cut', 'follow', 'selectAll', 'selectRank', 'setHeight', 'state'])
  })
})

describe('useReplayRelief — les fonds sont ceux de la préférence', () => {
  it.each<MapBackgroundVariant>(['plan', 'rendu', 'relief'])('le fond voulu %s est celui qui est retenu quand tout est lisible', (style) => {
    const base = baseLayer({ variante: style === 'rendu' ? 'rendu' : 'plan' })
    const { result } = renderHook(() => useReplayRelief(options({ base, style })))
    expect(result.current.selected).toBe(style)
  })
})
