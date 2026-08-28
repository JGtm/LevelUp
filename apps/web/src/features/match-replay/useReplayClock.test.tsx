/**
 * Tests — useReplayClock (la publication de l'image courante hors du canvas).
 *
 * CE QU'ILS PROTÈGENT : le bridage à 150 ms, qui existe pour ne pas re-rendre le DOM soixante
 * fois par seconde, ET SON EXCEPTION — la DERNIÈRE image de la fenêtre de gameplay part sans
 * délai. Sans elle, l'écran de fin de match ne se rendait pas : la boucle de lecture peint cette
 * image puis s'arrête, personne ne repasse derrière pour rattraper une publication sautée
 * (régression constatée par l'utilisateur le 2026-08-28, « je n'ai plus le message qui indique
 * la défaite ou victoire »).
 */
import { describe, expect, it, vi } from 'vitest'
import { renderHook } from '@testing-library/react'

import { useReplayClock } from './useReplayClock'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'

/** Un document réduit à ce que l'horloge lui demande : la cadence et le nombre d'images. */
const DOC = { frameIntervalMs: 100, frameCount: 600, durationMs: 60_000 } as unknown as ReplayDocumentReady

/** La fenêtre de gameplay du témoin : la fin tombe au frame 500. */
const WINDOW: ReplayWindowBounds = { startFrame: 10, endFrame: 500, startMs: 1_000, endMs: 50_000 }

function mount(playWindow: ReplayWindowBounds | null, onFrameChange: (f: number) => void) {
  return renderHook(() => useReplayClock({ doc: DOC, playWindow, onFrameChange })).result.current
}

describe('useReplayClock — la publication de l’image', () => {
  it('bride le flux : deux images coup sur coup ne publient qu’une fois', () => {
    const onFrameChange = vi.fn()
    const { tick } = mount(WINDOW, onFrameChange)
    tick(100)
    tick(101)
    expect(onFrameChange).toHaveBeenCalledTimes(1)
    expect(onFrameChange).toHaveBeenCalledWith(100)
  })

  it('publie la BORNE DE FIN sans attendre le bridage (l’écran de fin en dépend)', () => {
    const onFrameChange = vi.fn()
    const { tick } = mount(WINDOW, onFrameChange)
    tick(100)
    tick(WINDOW.endFrame)
    expect(onFrameChange).toHaveBeenLastCalledWith(WINDOW.endFrame)
  })

  it('publie aussi AU-DELÀ de la borne (frise tirée au bout)', () => {
    const onFrameChange = vi.fn()
    const { tick } = mount(WINDOW, onFrameChange)
    tick(100)
    tick(WINDOW.endFrame + 20)
    expect(onFrameChange).toHaveBeenLastCalledWith(WINDOW.endFrame + 20)
  })

  it('sans fenêtre, aucune image n’échappe au bridage : il n’y a pas de borne à annoncer', () => {
    const onFrameChange = vi.fn()
    const { tick } = mount(null, onFrameChange)
    tick(100)
    tick(599)
    expect(onFrameChange).toHaveBeenCalledTimes(1)
  })
})
