/**
 * Tests — `useReplayPlayback.seekToFrame` : LE SAUT QUI NE MET PAS EN PAUSE.
 *
 * # CE QU'ILS PROTÈGENT (2026-09-07, revue F5)
 *
 * `seekToFrame` est le geste du clic sur une porte de présence, exposé par le lot L4 : aller à
 * l'instant où quelqu'un est arrivé ou parti. Son contrat le distingue de `stepFrames`, qui
 * partage pourtant le même `seekTo` : un saut vers un instant NOMMÉ est un déplacement, pas un
 * arrêt sur image — la lecture reprend de là, et c'est ce qu'on attend en cliquant sur un repère
 * pendant qu'on regarde. Rien ne le tenait : la seule différence entre les deux commandes est un
 * `setPlaying(false)` qu'un futur passage pouvait recopier sans qu'aucun test ne rougisse.
 *
 * # POURQUOI UN FICHIER À CÔTÉ, ET PAS TROIS CAS DE PLUS DANS L'AUTRE
 *
 * `useReplayPlayback.test.tsx` est à moins de vingt lignes de CODE du plafond du dépôt
 * (`max-lines`, 500, ratchet : on extrait, on ne relève pas). Trois cas de plus l'y faisaient
 * passer. Le harnais est donc refait ici, court et local — deuxième et dernière copie tolérée
 * (règle n° 6) : une troisième imposerait de sortir le harnais dans un module partagé.
 *
 * LA BOUCLE D'ANIMATION EST NEUTRALISÉE, pas pilotée : ces cas n'ont besoin d'aucun pas de
 * boucle, seulement de savoir que la lecture est TOUJOURS en marche après le saut. La file de
 * rappels ne sert donc qu'à empêcher jsdom d'en exécuter un vrai.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type RefObject } from 'react'

import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { useReplayPlayback } from './useReplayPlayback'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'

/** Un document de 51 images (`endFrame` = 50) à la cadence par défaut. */
const DOC = testReplayDoc({ frameCount: 51 })

/** La fenêtre de gameplay du fichier voisin : le match court de l'image 10 à la 40, préambule 9. */
const FENETRE: ReplayWindowBounds = {
  startFrame: 10,
  leadInFrame: 9,
  endFrame: 40,
  startMs: 10_000,
  endMs: 40_000,
}

beforeEach(() => {
  // La lecture automatique est ÉTEINTE par défaut depuis le 2026-08-29 : ces cas éprouvent le
  // saut PENDANT une lecture, ils l'allument donc explicitement.
  localStorage.clear()
  localStorage.setItem(AUTOPLAY_KEY, 'true')
  vi.stubGlobal('requestAnimationFrame', () => 1)
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function monter(frame: number, extra: { playWindow?: ReplayWindowBounds | null; openAtFrame?: number | null } = {}) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  const draw = vi.fn()
  const soundTick = vi.fn()
  // Défaut par DÉSTRUCTURATION, pas `??` : un `playWindow: null` explicite (« pas de fenêtre »)
  // ne doit PAS retomber sur FENETRE — seul `undefined` (absence de la clé) le doit.
  const { playWindow = FENETRE, openAtFrame } = extra
  const view = renderHook(
    (props: { playWindow: ReplayWindowBounds | null; openAtFrame?: number | null }) =>
      useReplayPlayback({
        doc: DOC,
        playWindow: props.playWindow,
        baseFps: 10,
        speed: 1,
        renderWidth: 480,
        frameRef,
        draw,
        soundTick,
        openAtFrame: props.openAtFrame,
        onEnded: vi.fn(),
        onTransportGesture: vi.fn(),
      }),
    { initialProps: { playWindow, openAtFrame } },
  )
  return { ...view, frameRef, draw, soundTick }
}

describe('useReplayPlayback — `seekToFrame`, le saut vers un instant nommé', () => {
  it('pose le curseur à l’image demandée SANS mettre en pause', () => {
    const { result, frameRef } = monter(20)
    expect(result.current.playing).toBe(true)
    act(() => {
      result.current.seekToFrame(33)
    })
    expect(frameRef.current).toBe(33)
    expect(result.current.playing).toBe(true)
  })

  it('peint et fait battre le son, comme un pas de boucle', () => {
    // Un curseur déplacé sans repeindre montrerait la scène de l'instant précédent, et la piste
    // sonore reprendrait au mauvais endroit.
    const { result, draw, soundTick } = monter(20)
    draw.mockClear()
    soundTick.mockClear()
    act(() => {
      result.current.seekToFrame(33)
    })
    expect(draw).toHaveBeenCalledTimes(1)
    expect(soundTick).toHaveBeenCalledTimes(1)
  })

  it('reste borné à la fenêtre, aux deux extrémités', () => {
    const { result, frameRef } = monter(20)
    act(() => {
      result.current.seekToFrame(9_999)
    })
    expect(frameRef.current).toBe(FENETRE.endFrame)
    act(() => {
      result.current.seekToFrame(-5)
    })
    // La borne basse est le PRÉAMBULE, pas le coup d'envoi (décision D3 du 2026-09-02).
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
  })

  it('écrit le remplissage de la frise, comme tout déplacement', () => {
    const { result } = monter(10)
    const el = document.createElement('input')
    el.type = 'range'
    result.current.sliderRef.current = el
    act(() => {
      result.current.seekToFrame(25)
    })
    expect(el.value).toBe('25')
    expect(el.style.getPropertyValue('--played')).toBe('50%')
  })

  it('CONTRASTE AVEC `stepFrames`, qui lui met en pause — les deux partagent `seekTo`', () => {
    const { result } = monter(20)
    act(() => {
      result.current.stepFrames(1)
    })
    expect(result.current.playing).toBe(false)
  })
})

// ─── openAtFrame — le lien tactique (`?t=&clock=`, lot M1b, 2026-09-08) ────────────────────

describe('useReplayPlayback — `openAtFrame`, le lien tactique ouvert à une frame précise', () => {
  it('pose le curseur à la frame demandée au montage, via le MÊME `seekTo` (peint, fait battre le son)', () => {
    const { frameRef, draw, soundTick } = monter(0, { openAtFrame: 25 })
    // `seekTo` ne borne QUE par la fenêtre de gameplay — [leadInFrame=9, endFrame=40] ici.
    expect(frameRef.current).toBe(25)
    expect(draw).toHaveBeenCalled()
    expect(soundTick).toHaveBeenCalled()
  })

  it('`openAtFrame` absent (null/undefined) : comportement d’avant ce lot, cadrage au préambule', () => {
    const { frameRef } = monter(0, { openAtFrame: null })
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
  })

  it('ne se répète jamais : un `openAtFrame` appliqué ne reprend pas la main sur une frise déplacée depuis', () => {
    const { result, frameRef, rerender } = monter(0, { openAtFrame: 12 })
    expect(frameRef.current).toBe(12)
    act(() => {
      result.current.seekToFrame(30) // l'utilisateur déplace la frise depuis.
    })
    expect(frameRef.current).toBe(30)
    // Un rendu qui repropose LA MÊME valeur ne doit rien reposer.
    rerender({ playWindow: FENETRE, openAtFrame: 12 })
    expect(frameRef.current).toBe(30)
  })

  it('GAGNE SUR LE CADRAGE AU COUP D’ENVOI même si la fenêtre de gameplay arrive APRÈS (Match View asynchrone)', () => {
    // Sans fenêtre au montage (comportement « avant ce lot » : image zéro) puis la fenêtre
    // arrive — le cadrage au préambule ne doit PAS écraser le lien déjà posé, même à une
    // frame antérieure au préambule (2 < leadInFrame=9).
    const { frameRef, rerender } = monter(0, { playWindow: null, openAtFrame: 2 })
    expect(frameRef.current).toBe(2)
    rerender({ playWindow: FENETRE, openAtFrame: 2 })
    expect(frameRef.current).toBe(2)
  })
})
