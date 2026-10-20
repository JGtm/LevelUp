/**
 * Tests — useReplayFullscreen (le mode plein écran du rejeu).
 *
 * jsdom n'implémente pas l'API Fullscreen : elle est SIMULÉE ici au plus près du navigateur —
 * `requestFullscreen` pose l'élément puis émet `fullscreenchange`, `exitFullscreen` le retire puis
 * émet de même. Ce que ces cas protègent :
 *  - le natif est demandé sur la PAGE entière, et sa chute (Échap du navigateur) ferme le mode ;
 *  - un refus ou une API absente laissent la superposition seule — c'est déjà le mode ;
 *  - un export en cours fige le mode ;
 *  - en superposition seule, Échap ferme d'abord la couche la plus intérieure, le mode ensuite —
 *    même quand la couche s'est abonnée au clavier AVANT le mode et se retire du DOM aussitôt.
 */
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { requestExportLayout } from '../export/exportLayoutStore'
import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { ReplaySpeedMenu } from '../ui/ReplaySpeedMenu'
import {
  REPLAY_ESCAPE_LAYER_ATTR,
  useReplayFullscreen,
  type ReplayFullscreen,
} from './useReplayFullscreen'

/** L'élément en plein écran natif, tel que le navigateur simulé le tient. */
let fullscreenElement: Element | null = null

function emitChange(): void {
  document.dispatchEvent(new Event('fullscreenchange'))
}

/** Le navigateur ACCEPTE la demande : la page passe en plein écran natif. */
function acceptingRequest() {
  return vi.fn(async function (this: Element) {
    fullscreenElement = document.documentElement
    emitChange()
  })
}

function installFullscreenApi(request: () => Promise<void>, enabled = true): void {
  Object.defineProperty(document, 'fullscreenEnabled', { configurable: true, value: enabled })
  Object.defineProperty(document, 'fullscreenElement', {
    configurable: true,
    get: () => fullscreenElement,
  })
  Object.defineProperty(document.documentElement, 'requestFullscreen', {
    configurable: true,
    value: request,
  })
  Object.defineProperty(document, 'exitFullscreen', {
    configurable: true,
    value: vi.fn(async () => {
      fullscreenElement = null
      emitChange()
    }),
  })
}

/** Le navigateur SORT de lui-même (Échap en plein écran natif) : la page ne voit que l'événement. */
function browserLeavesFullscreen(): void {
  act(() => {
    fullscreenElement = null
    emitChange()
  })
}

function pressEscape(): void {
  act(() => {
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
  })
}

/** Laisse aboutir les promesses de l'API simulée (et leurs `catch`). */
async function settle(): Promise<void> {
  await act(async () => {
    await Promise.resolve()
  })
}

let warn: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  fullscreenElement = null
  warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
})

afterEach(() => {
  act(() => requestExportLayout(null))
  warn.mockRestore()
  document.body.innerHTML = ''
  for (const key of ['fullscreenEnabled', 'fullscreenElement', 'exitFullscreen'] as const) {
    Reflect.deleteProperty(document, key)
  }
  Reflect.deleteProperty(document.documentElement, 'requestFullscreen')
})

describe('useReplayFullscreen — plein écran natif', () => {
  it('ENTRÉE : ouvre le mode et demande le plein écran de TOUTE la page', async () => {
    const request = acceptingRequest()
    installFullscreenApi(request)
    const { result } = renderHook(() => useReplayFullscreen())
    expect(result.current.active).toBe(false)
    act(() => result.current.toggle())
    await settle()
    expect(result.current.active).toBe(true)
    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.contexts[0]).toBe(document.documentElement)
  })

  it('SORTIE AU BOUTON : ferme le mode et rend le plein écran natif', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    await settle()
    act(() => result.current.toggle())
    await settle()
    expect(result.current.active).toBe(false)
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
    expect(fullscreenElement).toBeNull()
  })

  it('ÉCHAP DU NAVIGATEUR : la chute du natif (`fullscreenchange`) ferme le mode', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    await settle()
    browserLeavesFullscreen()
    expect(result.current.active).toBe(false)
    // Le navigateur est déjà sorti : rien à lui rendre.
    expect(document.exitFullscreen).not.toHaveBeenCalled()
  })

  it('un `fullscreenchange` étranger (une vidéo en plein écran) ne touche pas au mode', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    await settle()
    const video = document.createElement('video')
    act(() => {
      fullscreenElement = video
      emitChange()
    })
    expect(result.current.active).toBe(true)
  })

  it('F DEUX FOIS AVANT LA TRANSITION : la demande arrivée après la sortie est défaite', async () => {
    let grant: () => void = () => {}
    const request = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          grant = () => {
            fullscreenElement = document.documentElement
            emitChange()
            resolve()
          }
        }),
    )
    installFullscreenApi(request)
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    act(() => result.current.toggle())
    expect(result.current.active).toBe(false)
    await act(async () => grant())
    await settle()
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
    expect(fullscreenElement).toBeNull()
    expect(result.current.active).toBe(false)
  })

  it('QUITTER LA PAGE rend le plein écran natif', async () => {
    installFullscreenApi(acceptingRequest())
    const { result, unmount } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    await settle()
    unmount()
    await settle()
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
  })
})

describe('useReplayFullscreen — le repli en superposition seule', () => {
  it('DEMANDE REFUSÉE : le mode reste ouvert en superposition, et le refus est journalisé', async () => {
    installFullscreenApi(vi.fn(() => Promise.reject(new Error('refus'))))
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    await settle()
    expect(result.current.active).toBe(true)
    expect(warn).toHaveBeenCalledTimes(1)
  })

  it('API ABSENTE OU INTERDITE : aucune demande, la superposition seule fait le mode', () => {
    const request = acceptingRequest()
    installFullscreenApi(request, false)
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    expect(result.current.active).toBe(true)
    expect(request).not.toHaveBeenCalled()
  })

  it('Échap ferme le mode, et le bouton aussi', () => {
    installFullscreenApi(acceptingRequest(), false)
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())
    pressEscape()
    expect(result.current.active).toBe(false)
    act(() => result.current.toggle())
    act(() => result.current.toggle())
    expect(result.current.active).toBe(false)
  })

  it('le basculement est annoncé comme un redimensionnement (hauteur du terrain, tiroir)', () => {
    installFullscreenApi(acceptingRequest(), false)
    const resize = vi.fn()
    window.addEventListener('resize', resize)
    const { result } = renderHook(() => useReplayFullscreen())
    expect(resize).not.toHaveBeenCalled()
    act(() => result.current.toggle())
    expect(resize).toHaveBeenCalledTimes(1)
    act(() => result.current.toggle())
    expect(resize).toHaveBeenCalledTimes(2)
    window.removeEventListener('resize', resize)
  })
})

describe('useReplayFullscreen — pendant un export vidéo', () => {
  it('le mode est désactivé : ni ouverture, ni fermeture au bouton ou à F', () => {
    const request = acceptingRequest()
    installFullscreenApi(request)
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    expect(result.current.disabled).toBe(true)
    act(() => result.current.toggle())
    expect(result.current.active).toBe(false)
    expect(request).not.toHaveBeenCalled()
    act(() => requestExportLayout(null))
    expect(result.current.disabled).toBe(false)
  })
})

describe('useReplayFullscreen — Échap ferme la couche la plus intérieure d’abord', () => {
  it('une couche ouverte AVANT le mode, qui se retire aussitôt : seule elle se ferme', () => {
    installFullscreenApi(acceptingRequest(), false)
    // LE PIRE ORDRE : la couche s'abonne au clavier AVANT le mode, et se retire du DOM pendant
    // la frappe même (comme React vidant sa mise à jour entre deux écouteurs).
    const couche = document.createElement('div')
    couche.setAttribute(REPLAY_ESCAPE_LAYER_ATTR, '')
    document.body.appendChild(couche)
    const fermerLaCouche = (e: KeyboardEvent) => {
      if (e.key === 'Escape') couche.remove()
    }
    window.addEventListener('keydown', fermerLaCouche)
    const { result } = renderHook(() => useReplayFullscreen())
    act(() => result.current.toggle())

    pressEscape()
    expect(couche.isConnected).toBe(false)
    expect(result.current.active).toBe(true)

    pressEscape()
    expect(result.current.active).toBe(false)
    window.removeEventListener('keydown', fermerLaCouche)
  })

  it('avec le vrai menu de vitesse : Échap le referme, puis ferme le mode', () => {
    installFullscreenApi(acceptingRequest(), false)
    const mode: { current: ReplayFullscreen | null } = { current: null }
    function Banc() {
      mode.current = useReplayFullscreen()
      return <ReplaySpeedMenu speed={1} onSetSpeed={() => {}} locale="fr" />
    }
    render(<Banc />)
    act(() => mode.current?.toggle())
    fireEvent.click(screen.getByRole('button', { name: 'Vitesse' }))
    expect(screen.getByRole('group', { name: 'Vitesse' })).toHaveAttribute(REPLAY_ESCAPE_LAYER_ATTR)

    pressEscape()
    expect(screen.queryByRole('group', { name: 'Vitesse' })).toBeNull()
    expect(mode.current?.active).toBe(true)

    pressEscape()
    expect(mode.current?.active).toBe(false)
  })
})
