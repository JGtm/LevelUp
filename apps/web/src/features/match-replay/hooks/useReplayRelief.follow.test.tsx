/**
 * Tests — « Suivre le joueur ouvert » à travers `useReplayRelief` (item D.6 du plan du fond Relief).
 *
 * La logique pure du pas (sol porteur, seuil de 0,5 u, délai de 300 ms) est testée dans `follow.test.ts` ; ici
 * se teste le BRANCHEMENT : la lecture du film (`reliefFollowSample`), le temps du rejeu, l'état partagé avec les
 * choix manuels (`useReplayFloorChoice`) et ce que le panneau en voit. Les fichiers du relief et le calculateur
 * sont des doubles, comme dans `useReplayRelief.test.tsx` ; la carte est un sol bas (x de 0 à 5) et un étage à
 * 2,2 u (x de 5 à 10). Une image du film dure 100 ms.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { reliefData } from '@/lib/replay/relief/reliefScene.testutil'
import type { ReliefFiles } from '@/lib/replay/reliefQueries'
import type { ReliefRender } from '@/lib/replay/useReliefRender'
import type { ReliefFrame } from '@/lib/replay/relief'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { testReplayDoc } from '../test/testDoc'
import type { ReplayMapBackgroundLayer } from './useReplayView'
import { useReplayRelief, type ReplayReliefOptions } from './useReplayRelief'

const mocks = vi.hoisted(() => ({ files: vi.fn(), render: vi.fn() }))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReplayReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))
vi.mock('@/lib/replay/relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => 'canvas-relief'
    maskedImage = () => 'canvas-masque'
  },
}))

const FRAME_MS = 100
/** Le délai du suivi (300 ms de temps de rejeu) en images du film. */
const DELAY_FRAMES = 3
const LOW = 0
const HIGH = 2.2

const CARTE = reliefData({
  width: 100,
  height: 100,
  levels: [LOW, HIGH],
  rects: [
    { i0: 0, j0: 0, i1: 99, j1: 99, z: LOW },
    { i0: 50, j0: 0, i1: 99, j1: 99, z: HIGH },
  ],
})

const BASE: ReplayMapBackgroundLayer = {
  calibration: CARTE.calibration as never,
  image: 'image-de-base' as unknown as CanvasImageSource,
  plan: null,
  variante: 'plan',
  planAvailable: true,
  reliefAvailable: true,
  zonesClip: null,
}

interface Point {
  x: number
  z: number
}

/** Une vie d'un joueur, une position par image de `from` à `to` (bornes comprises). */
function life(slot: number, xuid: string, from: number, to: number, at: (frame: number) => Point) {
  const points = []
  for (let t = from; t <= to; t++) points.push({ t, x: at(t).x, y: 5, z: at(t).z })
  return { slot, team: 0, xuid, startFrame: from, endFrame: to, points }
}

/** Le document et les joueurs de test : les vies données, regroupées par joueur. */
function film(...lives: ReturnType<typeof life>[]) {
  const doc = testReplayDoc({ frameIntervalMs: FRAME_MS, tracks: lives as never })
  const player = (xuid: string) => ({ xuid, lives: doc.tracks.filter((t) => t.xuid === xuid) }) as unknown as ReplayPlayer
  return { doc, player }
}

const frameRef = { current: 0 }

function options(doc: ReturnType<typeof film>['doc'], player: ReplayPlayer | null, over: Partial<ReplayReliefOptions> = {}): ReplayReliefOptions {
  return {
    base: BASE, playerSlug: 'jg', matchId: 'm1', scopeKey: 'carte-a', style: 'plan', turns: 0, zRange: { min: 0, max: 5 },
    ink: { base: '#6b7078', outline: '#ffffff' }, follow: { doc, player, frameRef }, ...over,
  }
}

const emptyLines = { points: new Float32Array(0), offsets: new Uint32Array(1) }
const FRAME: ReliefFrame = {
  width: 100, height: 100, cut: null, image: new Uint8ClampedArray(4), alpha: null, planAlpha: null,
  lines: { steps: emptyLines, contours: emptyLines, walls: emptyLines },
}

beforeEach(() => {
  window.localStorage.clear()
  frameRef.current = 0
  mocks.files.mockReset()
  mocks.render.mockReset()
  mocks.files.mockReturnValue({ data: CARTE, blob: new Blob(), loading: false, error: null } satisfies ReliefFiles)
  mocks.render.mockReturnValue({ status: 'ready', info: { width: 100, height: 100, hasMesh: true }, frame: FRAME, error: null } satisfies ReliefRender)
})

type Hook = ReturnType<typeof renderHook<ReturnType<typeof useReplayRelief>, ReplayReliefOptions>>

/** Monte le hook, allume le suivi à l'image 0 et rend le hook. */
function followed(doc: ReturnType<typeof film>['doc'], player: ReplayPlayer): Hook {
  const hook = renderHook((o: ReplayReliefOptions) => useReplayRelief(o), { initialProps: options(doc, player) })
  act(() => hook.result.current.floor.follow.onToggle())
  return hook
}

/** Joue les images de `from` à `to` (bornes comprises), une par une, et rend la coupe après chacune. */
function play(hook: Hook, from: number, to: number): Array<number | null> {
  const cuts: Array<number | null> = []
  for (let f = from; f <= to; f++) {
    frameRef.current = f
    act(() => hook.result.current.followStep(f))
    cuts.push(hook.result.current.floor.cut)
  }
  return cuts
}

describe('« Suivre le joueur ouvert » — marche et saut', () => {
  it('allumé, la coupe prend l’étage du joueur à l’image courante, sans délai', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, () => ({ x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    expect(hook.result.current.floor.follow).toMatchObject({ available: true, on: true })
    expect(hook.result.current.floor.cut).toBe(HIGH)
  })

  it('marche sur un étage : la coupe ne bouge plus', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, (f) => ({ x: 6 + f / 100, z: HIGH })))
    const hook = followed(doc, player('A'))
    expect(new Set(play(hook, 1, 100))).toEqual(new Set([HIGH]))
  })

  it('saut sur place de près d’une unité : la coupe ne bouge pas', () => {
    // Un saut dure une seconde (dix images) et monte de 0,9 u : plus haut que la bande, jamais un changement d’étage.
    const jump = (f: number) => ({ x: 7, z: HIGH + 0.9 * Math.sin((Math.PI * (f % 10)) / 10) })
    const { doc, player } = film(life(1, 'A', 0, 100, jump))
    const hook = followed(doc, player('A'))
    expect(new Set(play(hook, 1, 100))).toEqual(new Set([HIGH]))
  })
})

describe('« Suivre le joueur ouvert » — monter, tomber', () => {
  it('monte d’un étage : la coupe suit après le délai de 300 ms de rejeu, pas avant', () => {
    // Au sol jusqu’à l’image 20, puis sur l’étage (x = 7, z = 2,2).
    const { doc, player } = film(life(1, 'A', 0, 100, (f) => (f < 20 ? { x: 2, z: LOW } : { x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    expect(hook.result.current.floor.cut).toBe(LOW)
    const cuts = play(hook, 1, 40)
    const firstHigh = cuts.indexOf(HIGH) + 1
    expect(firstHigh).toBe(20 + DELAY_FRAMES)
    expect(cuts.slice(firstHigh - 1)).not.toContain(LOW)
  })

  it('chute d’un étage : la coupe descend après le délai', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, (f) => (f < 20 ? { x: 7, z: HIGH } : { x: 2, z: HIGH - 0.5 * (f - 20) })))
    const hook = followed(doc, player('A'))
    expect(hook.result.current.floor.cut).toBe(HIGH)
    const cuts = play(hook, 1, 60)
    // L’image n se lit à cuts[n - 1] : encore l’étage à l’image 22, le sol bas à l’image 23 (20 + 3).
    expect(cuts[22 - 1]).toBe(HIGH)
    expect(cuts[20 + DELAY_FRAMES - 1]).toBe(LOW)
    expect(cuts.slice(20 + DELAY_FRAMES - 1).every((c) => c === LOW)).toBe(true)
  })

  it('le délai se compte en temps de rejeu : une image répétée (pause) ne l’écoule pas', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, (f) => (f < 20 ? { x: 2, z: LOW } : { x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    play(hook, 1, 20)
    for (let i = 0; i < 500; i++) act(() => hook.result.current.followStep(20))
    expect(hook.result.current.floor.cut).toBe(LOW)
  })

  it('un saut de lecture de plus d’une seconde pose l’étage sans délai', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, (f) => (f < 50 ? { x: 2, z: LOW } : { x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    play(hook, 1, 10)
    frameRef.current = 60
    act(() => hook.result.current.followStep(60))
    expect(hook.result.current.floor.cut).toBe(HIGH)
  })
})

describe('« Suivre le joueur ouvert » — mort et réapparition', () => {
  const lives = () => film(life(1, 'A', 0, 20, () => ({ x: 2, z: LOW })), life(1, 'A', 50, 100, () => ({ x: 7, z: HIGH })))

  it('à la mort l’étage reste, le suivi aussi ; à la réapparition à un autre étage, la coupe le prend aussitôt', () => {
    const { doc, player } = lives()
    const hook = followed(doc, player('A'))
    play(hook, 1, 20)
    const mort = play(hook, 21, 49)
    expect(new Set(mort)).toEqual(new Set([LOW]))
    expect(hook.result.current.floor.follow.on).toBe(true)
    expect(play(hook, 50, 50)).toEqual([HIGH])
  })

  it('allumé pendant la mort : rien n’est posé avant la réapparition', () => {
    const { doc, player } = lives()
    frameRef.current = 30
    const hook = followed(doc, player('A'))
    expect(hook.result.current.floor.cut).toBeNull()
    expect(play(hook, 31, 50).pop()).toBe(HIGH)
  })
})

describe('« Suivre le joueur ouvert » — le choix manuel coupe le suivi', () => {
  const marche = () => film(life(1, 'A', 0, 100, (f) => (f < 10 ? { x: 2, z: LOW } : { x: 7, z: HIGH })))

  it.each([
    ['une pastille', (h: Hook) => h.result.current.floor.selectRank(LOW)],
    ['« Tout »', (h: Hook) => h.result.current.floor.selectAll()],
    ['le curseur', (h: Hook) => h.result.current.floor.setHeight(1.3)],
  ])('%s éteint le suivi et la coupe ne bouge plus', (_nom, geste) => {
    const { doc, player } = marche()
    const hook = followed(doc, player('A'))
    act(() => geste(hook))
    const coupe = hook.result.current.floor.cut
    expect(hook.result.current.floor.follow).toMatchObject({ available: true, on: false })
    expect(new Set(play(hook, 1, 60))).toEqual(new Set([coupe]))
  })

  it('rallumer le suivi reprend l’étage du joueur à l’image courante', () => {
    const { doc, player } = marche()
    const hook = followed(doc, player('A'))
    act(() => hook.result.current.floor.selectRank(LOW))
    frameRef.current = 30
    act(() => hook.result.current.floor.follow.onToggle())
    expect(hook.result.current.floor.cut).toBe(HIGH)
    expect(hook.result.current.floor.follow.on).toBe(true)
  })

  it('éteint par l’interrupteur, la coupe reste où elle est', () => {
    const { doc, player } = marche()
    const hook = followed(doc, player('A'))
    play(hook, 1, 40)
    act(() => hook.result.current.floor.follow.onToggle())
    expect(hook.result.current.floor.follow.on).toBe(false)
    expect(hook.result.current.floor.cut).toBe(HIGH)
  })
})

describe('« Suivre le joueur ouvert » — le joueur ouvert change ou disparaît', () => {
  const duo = () => film(life(1, 'A', 0, 100, () => ({ x: 2, z: LOW })), life(2, 'B', 0, 100, () => ({ x: 7, z: HIGH })))

  it('un autre joueur ouvert : le suivi reste allumé et repart de l’état initial (étage pris sans délai)', () => {
    const { doc, player } = duo()
    const hook = followed(doc, player('A'))
    play(hook, 1, 10)
    expect(hook.result.current.floor.cut).toBe(LOW)
    hook.rerender(options(doc, player('B')))
    expect(hook.result.current.floor.follow.on).toBe(true)
    expect(hook.result.current.floor.cut).toBe(HIGH)
  })

  it('aucun joueur ouvert : l’interrupteur est indisponible et ne s’allume pas', () => {
    const { doc } = duo()
    const hook = renderHook((o: ReplayReliefOptions) => useReplayRelief(o), { initialProps: options(doc, null) })
    expect(hook.result.current.floor.follow).toMatchObject({ available: false, on: false })
    act(() => hook.result.current.floor.follow.onToggle())
    expect(hook.result.current.floor.follow.on).toBe(false)
    expect(hook.result.current.floor.cut).toBeNull()
  })

  it('la fiche se ferme : le suivi s’éteint, la coupe reste, et rouvrir une fiche ne le rallume pas', () => {
    const { doc, player } = duo()
    const hook = followed(doc, player('B'))
    expect(hook.result.current.floor.cut).toBe(HIGH)
    hook.rerender(options(doc, null))
    expect(hook.result.current.floor.follow).toMatchObject({ available: false, on: false })
    expect(hook.result.current.floor.cut).toBe(HIGH)
    hook.rerender(options(doc, player('A')))
    expect(hook.result.current.floor.follow).toMatchObject({ available: true, on: false })
    expect(play(hook, 1, 20).every((c) => c === HIGH)).toBe(true)
  })

  it('une autre carte repart de « Tout », suivi éteint', () => {
    const { doc, player } = duo()
    const hook = followed(doc, player('B'))
    hook.rerender(options(doc, player('B'), { scopeKey: 'carte-b' }))
    expect(hook.result.current.floor.cut).toBeNull()
    expect(hook.result.current.floor.follow.on).toBe(false)
  })

  it('sans suivi, le pas ne fait rien', () => {
    const { doc, player } = duo()
    const hook = renderHook((o: ReplayReliefOptions) => useReplayRelief(o), { initialProps: options(doc, player('B')) })
    expect(play(hook, 1, 30).every((c) => c === null)).toBe(true)
  })
})

describe('« Suivre le joueur ouvert » — carte sans maillage', () => {
  it('pas de commande, pas de suivi', () => {
    mocks.files.mockReturnValue({
      data: reliefData({ width: 100, height: 100, degradations: ['sans-maillage'] }), blob: new Blob(), loading: false, error: null,
    })
    const { doc, player } = film(life(1, 'A', 0, 100, () => ({ x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    expect(hook.result.current.floor.state).toEqual({ state: 'noMesh' })
    expect(hook.result.current.floor.follow.on).toBe(false)
    expect(play(hook, 1, 10).every((c) => c === null)).toBe(true)
  })
})

describe('« Suivre le joueur ouvert » — la coupe ne masque aucun joueur', () => {
  it('suivi allumé, le hook ne rend toujours ni filtre ni masque de joueur', () => {
    const { doc, player } = film(life(1, 'A', 0, 100, () => ({ x: 7, z: HIGH })))
    const hook = followed(doc, player('A'))
    expect(Object.keys(hook.result.current).sort()).toEqual(
      ['background', 'floor', 'followStep', 'linesLayer', 'lines', 'meshPercent', 'options', 'selected'].sort(),
    )
    expect(Object.keys(hook.result.current.floor).sort()).toEqual(['cut', 'follow', 'selectAll', 'selectRank', 'setHeight', 'state'])
  })
})
