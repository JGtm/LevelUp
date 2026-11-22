/**
 * useReplayAntagonistMarks.test.tsx : le câblage des titres au canevas : les deux PNG chargés une
 * fois, un repeint à leur arrivée, aucun calque quand le point de vue n'a aucun titre à porter.
 */
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

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

function monte(over: Partial<Parameters<typeof useReplayAntagonistMarks>[0]> = {}) {
  const redraw = vi.fn()
  const view = renderHook(
    (props: Parameters<typeof useReplayAntagonistMarks>[0]) => useReplayAntagonistMarks(props),
    { initialProps: { doc: DOC, antagonists, ownerKeyOfSlot, reducedMotion: false, redraw, ...over } },
  )
  return { ...view, redraw }
}

// Une image qui arrive tout de suite : la config de Vitest remet les doubles à zéro entre les tests.
beforeEach(() => {
  vi.mocked(withLoadedImage).mockReset()
  vi.mocked(withLoadedImage).mockImplementation((_url, ready) => {
    ready({ naturalWidth: 100, naturalHeight: 100 } as unknown as HTMLImageElement)
  })
})

describe('useReplayAntagonistMarks', () => {
  it('charge les deux PNG noirs et repeint à leur arrivée', () => {
    const { redraw } = monte()
    expect(vi.mocked(withLoadedImage).mock.calls.map((c) => c[0]).sort()).toEqual(
      [TITLE_ICON_URL.nemesis, TITLE_ICON_URL.victim].sort(),
    )
    expect(redraw).toHaveBeenCalled()
  })

  it('lit les titres à l’image, sur l’axe du rejeu (image fois pas, en millisecondes)', () => {
    const { result } = monte({ reducedMotion: true })
    expect(result.current?.titlesOfSlot(2, 9)).toEqual([])
    expect(result.current?.titlesOfSlot(2, 10)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(result.current?.titlesOfSlot(1, 10)).toEqual([])
  })

  it('pulse pendant la durée de rémanence d’un événement, puis se repose', () => {
    const { result } = monte()
    const imageDeLaPrise = 10
    const apres = imageDeLaPrise + REPLAY_TIMING_MS.event / DOC.frameIntervalMs!
    expect(result.current?.titlesOfSlot(2, imageDeLaPrise + 1)[0].motion?.kind).toBe('gain')
    expect(result.current?.titlesOfSlot(2, apres)[0].motion).toBeUndefined()
  })

  it('en mouvement réduit, aucun mouvement', () => {
    const { result } = monte({ reducedMotion: true })
    expect(result.current?.titlesOfSlot(2, 11)[0].motion).toBeUndefined()
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
    const { result, rerender } = monte()
    const first = result.current
    rerender({ doc: DOC, antagonists, ownerKeyOfSlot, reducedMotion: false, redraw: vi.fn() })
    expect(result.current).toBe(first)
  })
})
