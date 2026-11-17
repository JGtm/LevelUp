/**
 * Tests — useReplayFrame (le cadre de la page du rejeu) et le chemin des touches T, F et Échap.
 *
 * Les transitions elles-mêmes sont celles du modèle pur (`replayFrame.test.ts`) ; ici on prouve ce
 * que le hook y ajoute : l'état de départ à chaque montage, l'annonce d'un redimensionnement à
 * chaque bascule, et — avec les raccourcis et le plein écran câblés comme la page les câble — la
 * table « cadre x touche -> effet », couche ouverte comprise, et l'export en cours.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { requestExportLayout } from '../export/exportLayoutStore'
import type { ReplayFrame } from '../model/replayFrame'
import { useReplayFrame } from './useReplayFrame'
import { REPLAY_ESCAPE_LAYER_ATTR, useReplayFullscreen } from './useReplayFullscreen'
import { useReplayShortcuts } from './useReplayShortcuts'

/** Le cadre, le plein écran et le clavier, câblés comme la page et le canvas les câblent. */
function useBanc() {
  const frame = useReplayFrame()
  const fullscreen = useReplayFullscreen(frame)
  useReplayShortcuts({
    togglePlay: () => {},
    seekBy: () => {},
    stepFrames: () => {},
    restart: () => {},
    toggleSound: () => {},
    toggleFullFrame: frame.toggleFullFrame,
    toggleCinema: frame.toggleCinema,
    skipSeconds: 10,
    enabled: true,
  })
  return { frame, fullscreen }
}

function press(key: string): void {
  act(() => {
    window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  })
}

/** Les cadres de départ de la table, avec les frappes qui y mènent depuis un montage neuf. */
const DEPARTS: ReadonlyArray<{ nom: string; frappes: string[]; frame: ReplayFrame }> = [
  { nom: 'standard', frappes: [], frame: 'standard' },
  { nom: 'cinema', frappes: ['t'], frame: 'cinema' },
  { nom: 'plein-cadre depuis standard', frappes: ['f'], frame: 'plein-cadre' },
  { nom: 'plein-cadre depuis cinema', frappes: ['t', 'f'], frame: 'plein-cadre' },
]

/** Cadre de départ x touche -> cadre attendu (Échap sans couche ouverte). */
const TABLE: ReadonlyArray<{ depart: string; touche: string; attendu: ReplayFrame }> = [
  { depart: 'standard', touche: 't', attendu: 'cinema' },
  { depart: 'standard', touche: 'f', attendu: 'plein-cadre' },
  { depart: 'standard', touche: 'Escape', attendu: 'standard' },
  { depart: 'cinema', touche: 't', attendu: 'standard' },
  { depart: 'cinema', touche: 'f', attendu: 'plein-cadre' },
  { depart: 'cinema', touche: 'Escape', attendu: 'cinema' },
  { depart: 'plein-cadre depuis standard', touche: 't', attendu: 'cinema' },
  { depart: 'plein-cadre depuis standard', touche: 'f', attendu: 'standard' },
  { depart: 'plein-cadre depuis standard', touche: 'Escape', attendu: 'standard' },
  { depart: 'plein-cadre depuis cinema', touche: 't', attendu: 'cinema' },
  { depart: 'plein-cadre depuis cinema', touche: 'f', attendu: 'cinema' },
  { depart: 'plein-cadre depuis cinema', touche: 'Escape', attendu: 'cinema' },
]

function monter(nom: string) {
  const banc = renderHook(useBanc)
  const depart = DEPARTS.find((d) => d.nom === nom)
  if (!depart) throw new Error(`départ inconnu : ${nom}`)
  for (const k of depart.frappes) press(k)
  expect(banc.result.current.frame.frame).toBe(depart.frame)
  return banc
}

function exporter(): void {
  act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
}

afterEach(() => {
  act(() => requestExportLayout(null))
  document.body.innerHTML = ''
})

describe('useReplayFrame — l’état', () => {
  it('naît « standard » à chaque montage, sans mémoire d’un montage à l’autre', () => {
    const premier = renderHook(useReplayFrame)
    expect(premier.result.current.frame).toBe('standard')
    act(() => premier.result.current.toggleCinema())
    expect(premier.result.current.frame).toBe('cinema')
    premier.unmount()
    const second = renderHook(useReplayFrame)
    expect(second.result.current.frame).toBe('standard')
  })

  it('`peek` voit l’état à cet instant, avant le rendu : deux gestes rapprochés se suivent', () => {
    const { result } = renderHook(useReplayFrame)
    act(() => {
      result.current.toggleCinema()
      expect(result.current.peek().frame).toBe('cinema')
      result.current.toggleFullFrame()
      expect(result.current.peek()).toEqual({ frame: 'plein-cadre', returnTo: 'cinema' })
    })
    expect(result.current.frame).toBe('plein-cadre')
  })
})

describe('useReplayFrame — chaque bascule annonce un redimensionnement', () => {
  it('une seule annonce par changement de cadre, aucune au premier rendu ni sans effet', () => {
    const resize = vi.fn()
    window.addEventListener('resize', resize)
    const { result } = renderHook(useReplayFrame)
    expect(resize).not.toHaveBeenCalled()
    act(() => result.current.exit())
    expect(resize).not.toHaveBeenCalled()
    act(() => result.current.toggleCinema())
    expect(resize).toHaveBeenCalledTimes(1)
    act(() => result.current.toggleFullFrame())
    expect(resize).toHaveBeenCalledTimes(2)
    act(() => result.current.exit())
    expect(resize).toHaveBeenCalledTimes(3)
    act(() => result.current.toggleCinema())
    expect(resize).toHaveBeenCalledTimes(4)
    window.removeEventListener('resize', resize)
  })

  it('une bascule ignorée pendant un export n’annonce rien', () => {
    const resize = vi.fn()
    window.addEventListener('resize', resize)
    const { result } = renderHook(useReplayFrame)
    exporter()
    act(() => result.current.toggleCinema())
    act(() => result.current.toggleFullFrame())
    expect(resize).not.toHaveBeenCalled()
    window.removeEventListener('resize', resize)
  })
})

describe('les touches T, F et Échap, cadre par cadre', () => {
  it.each(TABLE)('$depart + « $touche » -> $attendu', ({ depart, touche, attendu }) => {
    const { result } = monter(depart)
    press(touche)
    expect(result.current.frame.frame).toBe(attendu)
  })

  it('T et F gardent la main de leur frappe (défaut du navigateur retenu)', () => {
    renderHook(useBanc)
    for (const key of ['t', 'T', 'f', 'F']) {
      const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true })
      act(() => {
        window.dispatchEvent(event)
      })
      expect(event.defaultPrevented).toBe(true)
    }
  })
})

describe('Échap : une couche ouverte garde la priorité', () => {
  function ouvrirUneCouche(): HTMLElement {
    const couche = document.createElement('div')
    couche.setAttribute(REPLAY_ESCAPE_LAYER_ATTR, '')
    document.body.appendChild(couche)
    return couche
  }

  it('en Plein cadre : la couche ouverte empêche la sortie, qui suit dès qu’elle est fermée', () => {
    const { result } = monter('plein-cadre depuis cinema')
    const couche = ouvrirUneCouche()
    press('Escape')
    expect(result.current.frame.frame).toBe('plein-cadre')
    couche.remove()
    press('Escape')
    expect(result.current.frame.frame).toBe('cinema')
  })

  it('en Cinéma et en Standard : Échap ne fait rien, couche ouverte ou non', () => {
    for (const depart of ['cinema', 'standard']) {
      const { result, unmount } = monter(depart)
      const couche = ouvrirUneCouche()
      press('Escape')
      couche.remove()
      press('Escape')
      expect(result.current.frame.frame).toBe(depart)
      unmount()
    }
  })

  it('T et F ne tiennent pas compte des couches : ils basculent le cadre', () => {
    const { result } = monter('standard')
    ouvrirUneCouche()
    press('t')
    expect(result.current.frame.frame).toBe('cinema')
  })
})

describe('pendant un export vidéo : aucune bascule de cadre', () => {
  it.each(DEPARTS.map((d) => d.nom))('%s : T, F, le geste du système et les boutons sont sans effet', (nom) => {
    const { result } = monter(nom)
    const avant = result.current.frame.peek()
    const resize = vi.fn()
    window.addEventListener('resize', resize)
    exporter()
    expect(result.current.frame.disabled).toBe(true)
    press('t')
    press('f')
    press('F')
    press('T')
    act(() => result.current.frame.toggleCinema())
    act(() => result.current.frame.toggleFullFrame())
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.peek()).toEqual(avant)
    expect(resize).not.toHaveBeenCalled()
    window.removeEventListener('resize', resize)
  })

  it('la sortie du Plein cadre par Échap reste admise, comme le modèle la décrit', () => {
    const { result } = monter('plein-cadre depuis standard')
    exporter()
    press('Escape')
    expect(result.current.frame.frame).toBe('standard')
  })
})
