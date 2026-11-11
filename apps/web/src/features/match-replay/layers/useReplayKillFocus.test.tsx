/**
 * Tests — useReplayKillFocus : la mise en valeur d'une élimination, câblée au canvas.
 *
 * Ce qu'ils protègent : (1) la demande du magasin devient un dessin — équerres sur le tueur et la
 * victime, à leur position du film —, et rien sans demande ; (2) le nom du pion cède sa place à
 * l'étiquette (slots masqués) ; (3) l'animation suit le temps RÉEL et la boucle de repeinte S'ARRÊTE
 * à la fin de l'animation (jamais de boucle infinie sur une carte en pause) ; (4) `prefers-reduced-
 * motion` : pas de boucle, poses d'arrivée ; (5) un autre document ou le démontage efface ;
 * (6) l'icône de l'arme se charge et repeint.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { KillFocusTarget } from '../model/killFocusLogic'
import { FOCUS_MOTION_MS } from '../model/killFocusMotion'
import { createPlaybackStore } from '../model/playbackStore'
import type { CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { useReplayKillFocus } from './useReplayKillFocus'

const loaded = vi.hoisted(() => ({ withLoadedImage: vi.fn() }))
vi.mock('./loadedImage', () => loaded)
vi.mock('./replayDraw', () => ({ tintedIconCanvas: () => ({ width: 44, height: 20, tinted: true }) }))

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
  tracks: [
    { slot: 1, team: 0, xuid: 'k', points: [{ t: 0, x: 25, y: 75 }, { t: 100, x: 25, y: 75 }], startFrame: 0, endFrame: 100 },
    { slot: 2, team: 1, xuid: 'v', points: [{ t: 0, x: 75, y: 25 }, { t: 100, x: 75, y: 25 }], startFrame: 0, endFrame: 100 },
  ] as never,
})

const VIEW: CanvasView = { bounds: DOC.bounds as CanvasView['bounds'], width: 400, height: 400, pad: 0 }

const CIBLE: KillFocusTarget = {
  entryKey: 'e1',
  frame: 10,
  killerXuid: 'k',
  victimXuid: 'v',
  weapon: { imageUrl: '/w.png', tinted: true },
}

let maintenant = 0
let rafs: Array<() => void> = []

beforeEach(() => {
  maintenant = 1_000
  rafs = []
  loaded.withLoadedImage.mockReset()
  vi.spyOn(performance, 'now').mockImplementation(() => maintenant)
  vi.stubGlobal('requestAnimationFrame', (cb: () => void) => {
    rafs.push(cb)
    return rafs.length
  })
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

function monter(over: { reducedMotion?: boolean; doc?: typeof DOC } = {}) {
  const store = createPlaybackStore()
  const redraw = vi.fn()
  const view = renderHook(
    (props: { doc: typeof DOC }) =>
      useReplayKillFocus({
        doc: props.doc,
        view: VIEW,
        store,
        scrimInk: 'scrim',
        labelInk: 'label-ink',
        seatedOfXuid: () => true,
        colorOfXuid: (x) => (x === 'k' ? 'kc' : 'vc'),
        nameOfXuid: (x) => (x === 'k' ? 'Tueur' : 'Victime'),
        reducedMotion: over.reducedMotion ?? false,
        redraw,
      }),
    { initialProps: { doc: over.doc ?? DOC } },
  )
  return { ...view, store, redraw }
}

/** Joue un tour de la boucle d'images d'écran : exécute les rappels en attente. */
function tourDEcran() {
  const attente = rafs
  rafs = []
  attente.forEach((cb) => cb())
}

function ops(paint: (ctx: CanvasRenderingContext2D, f: number, k: number) => void) {
  const { ctx, ops: all } = recordingContext()
  Reflect.set(ctx, 'measureText', (t: string) => ({ width: t.length * 8 }))
  paint(ctx, 10, 1)
  return all
}

describe('les deux calques', () => {
  it('portent leur id, dans l’ordre de la scène', () => {
    const { result } = monter()
    expect(result.current.veil.id).toBe('mise-en-valeur')
    expect(result.current.labels.id).toBe('etiquettes-mise-en-valeur')
  })

  it('sans demande, rien ne se peint et aucun nom n’est masqué', () => {
    const { result } = monter()
    expect(ops(result.current.veil.paint).filter((o) => !o.op.startsWith('set measureText'))).toHaveLength(0)
    expect(ops(result.current.labels.paint).filter((o) => !o.op.startsWith('set measureText'))).toHaveLength(0)
    expect(result.current.nameHiddenAtSlot(1)).toBe(false)
  })

  it('un clic pose les équerres sur le tueur et la victime, à leur position du film', () => {
    const { result, store } = monter()
    act(() => store.focusKill(CIBLE))
    const veil = ops(result.current.veil.paint)
    // Deux trous au voile, deux tracés d'équerres.
    expect(veil.filter((o) => o.op === 'arc')).toHaveLength(2)
    expect(veil.filter((o) => o.op === 'stroke')).toHaveLength(2)
    const traits = veil.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(traits).toEqual(expect.arrayContaining(['kc', 'vc']))
    // Tueur en (25, 75) du monde = (100, 100) sur la toile.
    expect(veil.filter((o) => o.op === 'translate').map((o) => o.args)).toContainEqual([100, 100])
  })

  it('les étiquettes portent les deux noms', () => {
    const { result, store } = monter()
    act(() => store.focusKill(CIBLE))
    const noms = ops(result.current.labels.paint).filter((o) => o.op === 'fillText').map((o) => o.args[0])
    expect(noms).toEqual(['Tueur', 'Victime'])
  })

  it('le nom du pion cède la place à l’étiquette : les slots du tueur et de la victime sont masqués', () => {
    const { result, store } = monter()
    act(() => store.focusKill(CIBLE))
    expect(result.current.nameHiddenAtSlot(1)).toBe(true)
    expect(result.current.nameHiddenAtSlot(2)).toBe(true)
    expect(result.current.nameHiddenAtSlot(3)).toBe(false)
  })

  it('l’effacement (Lecture, saut manuel) retire tout', () => {
    const { result, store } = monter()
    act(() => store.focusKill(CIBLE))
    act(() => store.clearFocus())
    expect(ops(result.current.veil.paint).filter((o) => o.op === 'stroke')).toHaveLength(0)
    expect(result.current.nameHiddenAtSlot(1)).toBe(false)
  })

  it('une victime sans position : seul le tueur est mis en valeur', () => {
    const { result, store } = monter()
    act(() => store.focusKill({ ...CIBLE, victimXuid: 'inconnu' }))
    expect(ops(result.current.veil.paint).filter((o) => o.op === 'stroke')).toHaveLength(1)
  })
})

describe('l’animation — le temps réel, une boucle qui s’arrête', () => {
  it('repeint à chaque image d’écran PENDANT l’animation, puis s’arrête d’elle-même', () => {
    const { store, redraw } = monter()
    act(() => store.focusKill(CIBLE))
    redraw.mockClear()
    expect(rafs).toHaveLength(1)

    maintenant += 100
    tourDEcran()
    expect(redraw).toHaveBeenCalledTimes(1)
    expect(rafs).toHaveLength(1) // elle se rappelle : 100 ms < 340 ms

    maintenant += FOCUS_MOTION_MS
    tourDEcran()
    expect(redraw).toHaveBeenCalledTimes(2)
    expect(rafs).toHaveLength(0) // fini : plus rien ne tourne
  })

  it('chaque clic rejoue : le même clic répété relance une boucle', () => {
    const { store, redraw } = monter()
    act(() => store.focusKill(CIBLE))
    maintenant += FOCUS_MOTION_MS * 2
    tourDEcran()
    expect(rafs).toHaveLength(0)
    redraw.mockClear()
    act(() => store.focusKill(CIBLE))
    expect(rafs).toHaveLength(1)
  })

  it('au départ la pose est celle du départ, à la fin celle d’arrivée', () => {
    const { result, store } = monter()
    act(() => store.focusKill(CIBLE))
    const debut = ops(result.current.veil.paint)
    expect((debut.filter((o) => o.op === 'set globalAlpha').pop() as { args: number[] }).args[0]).toBe(0)
    maintenant += FOCUS_MOTION_MS
    const fin = ops(result.current.veil.paint)
    expect((fin.filter((o) => o.op === 'set globalAlpha').pop() as { args: number[] }).args[0]).toBe(1)
  })

  it('prefers-reduced-motion : aucune boucle, équerres posées tout de suite', () => {
    const { result, store } = monter({ reducedMotion: true })
    act(() => store.focusKill(CIBLE))
    expect(rafs).toHaveLength(0)
    const veil = ops(result.current.veil.paint)
    expect((veil.filter((o) => o.op === 'set globalAlpha').pop() as { args: number[] }).args[0]).toBe(1)
    expect(veil.filter((o) => o.op === 'scale').every((o) => o.args[0] === 1)).toBe(true)
  })
})

describe('le nettoyage', () => {
  it('un autre document efface la mise en valeur', () => {
    const { store, rerender } = monter()
    act(() => store.focusKill(CIBLE))
    expect(store.focus()).not.toBeNull()
    rerender({ doc: testReplayDoc({ frameIntervalMs: 100, tracks: [] }) })
    expect(store.focus()).toBeNull()
  })

  it('le démontage l’efface', () => {
    const { store, unmount } = monter()
    act(() => store.focusKill(CIBLE))
    unmount()
    expect(store.focus()).toBeNull()
  })
})

describe('l’icône de l’arme', () => {
  it('est demandée à l’image chargée, teinte, et repeint la scène à son arrivée', () => {
    const { store, redraw, result } = monter()
    act(() => store.focusKill(CIBLE))
    expect(loaded.withLoadedImage).toHaveBeenCalledTimes(1)
    expect(loaded.withLoadedImage.mock.calls[0][0]).toBe('/w.png')
    redraw.mockClear()
    const prete = loaded.withLoadedImage.mock.calls[0][1] as (im: unknown) => void
    act(() => prete({ width: 40, height: 20 }))
    expect(redraw).toHaveBeenCalledTimes(1)
    const images = ops(result.current.labels.paint).filter((o) => o.op === 'drawImage')
    expect(images).toHaveLength(1)
  })

  it('une arme non identifiée n’en charge aucune', () => {
    const { store } = monter()
    act(() => store.focusKill({ ...CIBLE, weapon: { imageUrl: '', tinted: false } }))
    expect(loaded.withLoadedImage).not.toHaveBeenCalled()
  })

  it('une image qui arrive après un autre clic est ignorée', () => {
    const { store, redraw } = monter()
    act(() => store.focusKill(CIBLE))
    const tardive = loaded.withLoadedImage.mock.calls[0][1] as (im: unknown) => void
    act(() => store.focusKill({ ...CIBLE, weapon: { imageUrl: '/autre.png', tinted: true } }))
    redraw.mockClear()
    act(() => tardive({ width: 40, height: 20 }))
    expect(redraw).not.toHaveBeenCalled()
  })
})
