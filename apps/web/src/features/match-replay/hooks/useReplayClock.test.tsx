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
import type { ReplayDocumentReady } from '../model/replayNormalize'
import type { ReplayWindowBounds } from '../model/replayWindow'

/** Un document réduit à ce que l'horloge lui demande : la cadence et le nombre d'images. */
const DOC = { frameIntervalMs: 100, frameCount: 600, durationMs: 60_000 } as unknown as ReplayDocumentReady

/** La fenêtre de gameplay du témoin : la fin tombe au frame 500. */
// `leadInFrame` est là pour le TYPE : l'horloge ne lit que `startMs` et `endFrame` — c'est
// précisément ce que la décision D3 (préambule de lecture, 2026-09-02) garantit.
const WINDOW: ReplayWindowBounds = {
  startFrame: 10,
  leadInFrame: 0,
  endFrame: 500,
  startMs: 1_000,
  endMs: 50_000,
}

function mount(playWindow: ReplayWindowBounds | null, publish: (f: number) => void) {
  return renderHook(() => useReplayClock({ doc: DOC, playWindow, publish })).result.current
}

describe('useReplayClock — la publication de l’image', () => {
  it('bride le flux : deux images coup sur coup ne publient qu’une fois', () => {
    const publish = vi.fn()
    const { tick } = mount(WINDOW, publish)
    tick(100)
    tick(101)
    expect(publish).toHaveBeenCalledTimes(1)
    expect(publish).toHaveBeenCalledWith(100)
  })

  it('publie la BORNE DE FIN sans attendre le bridage (l’écran de fin en dépend)', () => {
    const publish = vi.fn()
    const { tick } = mount(WINDOW, publish)
    tick(100)
    tick(WINDOW.endFrame)
    expect(publish).toHaveBeenLastCalledWith(WINDOW.endFrame)
  })

  it('publie aussi AU-DELÀ de la borne (frise tirée au bout)', () => {
    const publish = vi.fn()
    const { tick } = mount(WINDOW, publish)
    tick(100)
    tick(WINDOW.endFrame + 20)
    expect(publish).toHaveBeenLastCalledWith(WINDOW.endFrame + 20)
  })

  it('sans fenêtre, aucune image n’échappe au bridage : il n’y a pas de borne à annoncer', () => {
    const publish = vi.fn()
    const { tick } = mount(null, publish)
    tick(100)
    tick(599)
    expect(publish).toHaveBeenCalledTimes(1)
  })
})
