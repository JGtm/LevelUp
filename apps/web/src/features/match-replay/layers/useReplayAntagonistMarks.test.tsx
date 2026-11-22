/**
 * useReplayAntagonistMarks.test.tsx : le câblage des titres au canevas : les deux PNG chargés une
 * fois, un repeint à leur arrivée, aucun calque quand le point de vue n'a aucun titre à porter, et la
 * boucle de repeinte qui mène une pulsation à son terme quand la lecture est en pause.
 *
 * L'horloge murale et les images d'animation sont simulées : `wall` avance à la main, `etape` joue
 * l'image d'animation suivante.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildAntagonists, type AntagonistKill } from '@/lib/antagonists/antagonists'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { REPLAY_TIMING_MS } from '../hooks/useReplayTiming'
import { TITLE_ICON_URL } from './antagonistMarks'
import { withLoadedImage } from './loadedImage'
import { useReplayAntagonistMarks } from './useReplayAntagonistMarks'

vi.mock('./loadedImage', () => ({ withLoadedImage: vi.fn() }))

const DOC = { frameIntervalMs: 100 } as unknown as ReplayDocumentReady
const kill = (killer: string, victim: string, t: number): AntagonistKill => ({ killer, victim, t })
const antagonists = buildAntagonists([kill('e1', 'S', 1000)], 'S', (key) => key.startsWith('e'))
const aucun = buildAntagonists([], 'S', () => false)

const ownerKeyOfSlot = (slot: number): string | null => (slot === 2 ? 'e1' : null)

const wall = { now: 0 }
const horloge = () => wall.now
let file: Array<() => void> = []
/** Joue l'image d'animation suivante, à l'instant mural donné. */
function etape(now: number) {
  wall.now = now
  const cb = file.shift()
  cb?.()
}

type Props = Parameters<typeof useReplayAntagonistMarks>[0]

function monte(over: Partial<Props> = {}) {
  const redraw = vi.fn()
  const view = renderHook((props: Props) => useReplayAntagonistMarks(props), {
    initialProps: { doc: DOC, antagonists, ownerKeyOfSlot, reducedMotion: false, redraw, now: horloge, ...over },
  })
  return { ...view, redraw }
}

beforeEach(() => {
  wall.now = 0
  file = []
  vi.stubGlobal('requestAnimationFrame', (cb: () => void) => file.push(cb))
  vi.stubGlobal('cancelAnimationFrame', () => {
    file = []
  })
  // Une image qui arrive tout de suite : la config de Vitest remet les doubles à zéro entre les tests.
  vi.mocked(withLoadedImage).mockReset()
  vi.mocked(withLoadedImage).mockImplementation((_url, ready) => {
    ready({ naturalWidth: 100, naturalHeight: 100 } as unknown as HTMLImageElement)
  })
})
afterEach(() => vi.unstubAllGlobals())

describe('useReplayAntagonistMarks : les images', () => {
  it('charge les deux PNG noirs et repeint à leur arrivée', () => {
    const { redraw } = monte()
    expect(vi.mocked(withLoadedImage).mock.calls.map((c) => c[0]).sort()).toEqual(
      [TITLE_ICON_URL.nemesis, TITLE_ICON_URL.victim].sort(),
    )
    expect(redraw).toHaveBeenCalled()
  })

  it('sans titre à porter dans tout le match : aucun calque, aucune image chargée', () => {
    const { result } = monte({ antagonists: aucun })
    expect(result.current).toBeUndefined()
    expect(withLoadedImage).not.toHaveBeenCalled()
  })

  it('sans index (appelant sans rejeu) : rien non plus', () => {
    const { result } = monte({ antagonists: undefined })
    expect(result.current).toBeUndefined()
    expect(withLoadedImage).not.toHaveBeenCalled()
  })

  it('reste le même objet tant que l’index et le document ne changent pas', () => {
    const { result, rerender, redraw } = monte()
    const first = result.current
    // Le repeint de la page est stable (`useCallback`) : le même rendu rend le même calque.
    rerender({ doc: DOC, antagonists, ownerKeyOfSlot, reducedMotion: false, redraw, now: horloge })
    expect(result.current).toBe(first)
  })

  it('lit les titres à l’image, sur l’axe du rejeu (image fois pas, en millisecondes)', () => {
    const { result } = monte()
    expect(result.current?.titlesOfSlot(2, 9)).toEqual([])
    expect(result.current?.titlesOfSlot(2, 10)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(result.current?.titlesOfSlot(1, 10)).toEqual([])
  })
})

describe('useReplayAntagonistMarks : la pulsation', () => {
  it('démarre au franchissement par la lecture, d’une durée de rémanence d’événement', () => {
    const { result } = monte()
    // Image 9 -> 11 : l'instant 1000 ms (image 10) est franchi en avançant.
    result.current?.advance(9, 11)
    expect(result.current?.titlesOfSlot(2, 11)[0].motion?.kind).toBe('gain')
    wall.now = REPLAY_TIMING_MS.event - 1
    expect(result.current?.titlesOfSlot(2, 11)[0].motion).toBeDefined()
    wall.now = REPLAY_TIMING_MS.event
    expect(result.current?.titlesOfSlot(2, 11)[0].motion).toBeUndefined()
  })

  it('un saut ne déclenche rien : ni pulsation, ni boucle de repeinte', () => {
    const { result } = monte()
    expect(result.current?.titlesOfSlot(2, 11)[0].motion).toBeUndefined()
    expect(file).toHaveLength(0)
  })

  it('en pause, la boucle repeint jusqu’au terme de la pulsation puis s’arrête sur le repos', () => {
    const { result, redraw } = monte()
    redraw.mockClear()
    result.current?.advance(9, 11)
    expect(file).toHaveLength(1)
    etape(300)
    etape(700)
    etape(1399)
    expect(redraw).toHaveBeenCalledTimes(3)
    expect(file).toHaveLength(1)
    etape(1400)
    // Un dernier repeint pose l'état de repos, puis plus rien ne tourne.
    expect(redraw).toHaveBeenCalledTimes(4)
    expect(file).toHaveLength(0)
  })

  it('un repeint qui lit les titres, comme le vrai : une image par tick, et un tracé de repos garanti à la fin', () => {
    const { result, redraw } = monte()
    const vus: Array<string | undefined> = []
    redraw.mockImplementation(() => {
      vus.push(result.current?.titlesOfSlot(2, 11)[0].motion?.kind)
    })
    redraw.mockClear()
    result.current?.advance(9, 11)
    // La lecture est en pause : seules les images d'animation (16 ms) font avancer la pulsation.
    let ticks = 0
    for (let t = 16; file.length > 0 && ticks < 200; t += 16) {
      etape(t)
      ticks++
    }
    expect(file).toHaveLength(0)
    // Une image repeinte à chaque tick : le repeint de la boucle ne compte pas comme un tracé de la lecture.
    expect(redraw).toHaveBeenCalledTimes(ticks)
    // Le dernier tracé est celui du repos : la pulsation est finie quand il lit les titres.
    expect(vus[vus.length - 1]).toBeUndefined()
    expect(vus[vus.length - 2]).toBe('gain')
  })

  it('ne double pas le tracé de la lecture : pas de repeinte quand une image vient d’être peinte', () => {
    const { result, redraw } = monte()
    redraw.mockClear()
    result.current?.advance(9, 11)
    wall.now = 100
    result.current?.titlesOfSlot(2, 11) // le tracé de la lecture lit les titres
    etape(110)
    expect(redraw).not.toHaveBeenCalled()
    etape(300)
    expect(redraw).toHaveBeenCalledTimes(1)
  })

  it('en mouvement réduit : aucune pulsation, aucune boucle', () => {
    const { result } = monte({ reducedMotion: true })
    result.current?.advance(9, 11)
    expect(file).toHaveLength(0)
    expect(result.current?.titlesOfSlot(2, 11)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('la boucle s’arrête au démontage', () => {
    const { result, unmount } = monte()
    result.current?.advance(9, 11)
    expect(file).toHaveLength(1)
    unmount()
    expect(file).toHaveLength(0)
  })
})
