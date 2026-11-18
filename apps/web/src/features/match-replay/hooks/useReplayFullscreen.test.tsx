/**
 * Tests — useReplayFullscreen (le plein écran du système, et la sortie du Plein cadre par Échap).
 *
 * jsdom n'implémente pas l'API Fullscreen : elle est SIMULÉE ici au plus près du navigateur —
 * `requestFullscreen` pose l'élément puis émet `fullscreenchange`, `exitFullscreen` le retire puis
 * émet de même. Ce que ces cas protègent :
 *  - le natif est demandé sur la PAGE entière, seul, et s'ajoute à n'importe quel cadre ;
 *  - depuis le Standard, il entre d'abord en Plein cadre, qui se referme avec lui ; un Plein cadre
 *    ouvert autrement (F) ne se referme jamais avec le natif ;
 *  - `fullscreenchange` est la vérité du natif : sa chute (Échap du navigateur) est suivie ;
 *  - un refus ou une API absente ne laissent que le cadre ;
 *  - un export en cours fige le geste ;
 *  - en Plein cadre, Échap ferme d'abord la couche la plus intérieure, le cadre ensuite — même
 *    quand la couche s'est abonnée au clavier AVANT le hook et se retire du DOM aussitôt.
 */
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { requestExportLayout } from '../export/exportLayoutStore'
import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { ReplaySpeedMenu } from '../ui/ReplaySpeedMenu'
import { useReplayFrame } from './useReplayFrame'
import { REPLAY_ESCAPE_LAYER_ATTR, useReplayFullscreen } from './useReplayFullscreen'

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

/** Le cadre et le plein écran du système, câblés comme la page les câble. */
function useBanc() {
  const frame = useReplayFrame('match-test')
  const fullscreen = useReplayFullscreen(frame)
  return { frame, fullscreen }
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

describe('useReplayFullscreen — le natif seul', () => {
  it('ENTRÉE DEPUIS LE STANDARD : entre en Plein cadre et demande le plein écran de TOUTE la page', async () => {
    const request = acceptingRequest()
    installFullscreenApi(request)
    const { result } = renderHook(useBanc)
    expect(result.current.fullscreen.active).toBe(false)
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.frame.frame).toBe('plein-cadre')
    expect(result.current.fullscreen.active).toBe(true)
    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.contexts[0]).toBe(document.documentElement)
  })

  it('ENTRÉE DEPUIS LE CINÉMA : le natif s’ajoute, le cadre ne change pas', async () => {
    const request = acceptingRequest()
    installFullscreenApi(request)
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleCinema())
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.frame.frame).toBe('cinema')
    expect(result.current.fullscreen.active).toBe(true)
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('ENTRÉE DEPUIS LE PLEIN CADRE : le natif s’ajoute, le cadre ne change pas', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleFullFrame())
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.frame.frame).toBe('plein-cadre')
    expect(result.current.fullscreen.active).toBe(true)
  })

  it('SORTIE AU BOUTON, ouverte depuis le Standard : rend le natif ET referme le Plein cadre', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('standard')
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
    expect(fullscreenElement).toBeNull()
  })

  it('SORTIE AU BOUTON, ouverte depuis le Cinéma : rend le natif, le cadre reste', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleCinema())
    act(() => result.current.fullscreen.toggle())
    await settle()
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('cinema')
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
  })

  it('SORTIE AU BOUTON, Plein cadre ouvert par F : le natif part, le Plein cadre reste', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleFullFrame())
    act(() => result.current.fullscreen.toggle())
    await settle()
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('plein-cadre')
  })

  it('F depuis le Plein cadre + natif : le cadre se referme, le natif reste', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    act(() => result.current.frame.toggleFullFrame())
    expect(result.current.frame.frame).toBe('standard')
    expect(result.current.fullscreen.active).toBe(true)
    expect(document.exitFullscreen).not.toHaveBeenCalled()
  })
})

describe('useReplayFullscreen — `fullscreenchange` reste la vérité du natif', () => {
  it('ÉCHAP DU NAVIGATEUR, natif ouvert depuis le Standard : la chute referme aussi le Plein cadre', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    browserLeavesFullscreen()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('standard')
    // Le navigateur est déjà sorti : rien à lui rendre.
    expect(document.exitFullscreen).not.toHaveBeenCalled()
  })

  it('ÉCHAP DU NAVIGATEUR, natif ajouté au Cinéma : la chute est suivie, le cadre reste', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleCinema())
    act(() => result.current.fullscreen.toggle())
    await settle()
    browserLeavesFullscreen()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('cinema')
  })

  it('ÉCHAP DU NAVIGATEUR, natif ajouté au Plein cadre de F : la chute est suivie, le Plein cadre reste', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleFullFrame())
    act(() => result.current.fullscreen.toggle())
    await settle()
    browserLeavesFullscreen()
    expect(result.current.fullscreen.active).toBe(false)
    expect(result.current.frame.frame).toBe('plein-cadre')
  })

  it('un `fullscreenchange` étranger (une vidéo en plein écran) ne touche à rien', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    const video = document.createElement('video')
    act(() => {
      fullscreenElement = video
      emitChange()
    })
    expect(result.current.fullscreen.active).toBe(true)
    expect(result.current.frame.frame).toBe('plein-cadre')
  })

  it('GESTE DEUX FOIS AVANT LA TRANSITION : la demande arrivée après la sortie est défaite', async () => {
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
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('standard')
    await act(async () => grant())
    await settle()
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
    expect(fullscreenElement).toBeNull()
    expect(result.current.fullscreen.active).toBe(false)
  })

  it('QUITTER LA PAGE rend le plein écran natif', async () => {
    installFullscreenApi(acceptingRequest())
    const { result, unmount } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    unmount()
    await settle()
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
  })
})

describe('useReplayFullscreen — refus et API absente', () => {
  it('DEMANDE REFUSÉE : le Plein cadre du geste reste, et le refus est journalisé', async () => {
    installFullscreenApi(vi.fn(() => Promise.reject(new Error('refus'))))
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    expect(result.current.frame.frame).toBe('plein-cadre')
    expect(result.current.fullscreen.active).toBe(false)
    expect(warn).toHaveBeenCalledTimes(1)
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('standard')
  })

  it('API ABSENTE OU INTERDITE : aucune demande, le geste ouvre et ferme le Plein cadre seul', () => {
    const request = acceptingRequest()
    installFullscreenApi(request, false)
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('plein-cadre')
    expect(request).not.toHaveBeenCalled()
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('standard')
  })

  it('API ABSENTE depuis le Cinéma : le geste ne change rien, et le suivant non plus', () => {
    installFullscreenApi(acceptingRequest(), false)
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleCinema())
    act(() => result.current.fullscreen.toggle())
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('cinema')
  })
})

describe('useReplayFullscreen — Échap en Plein cadre', () => {
  it('Échap rend le cadre d’où l’on vient, et Échap hors Plein cadre ne fait rien', () => {
    installFullscreenApi(acceptingRequest(), false)
    const { result } = renderHook(useBanc)
    pressEscape()
    expect(result.current.frame.frame).toBe('standard')
    act(() => result.current.frame.toggleFullFrame())
    pressEscape()
    expect(result.current.frame.frame).toBe('standard')
    act(() => result.current.frame.toggleCinema())
    pressEscape()
    expect(result.current.frame.frame).toBe('cinema')
    act(() => result.current.frame.toggleFullFrame())
    pressEscape()
    expect(result.current.frame.frame).toBe('cinema')
  })

  it('Plein cadre ouvert par le geste du système : Échap rend aussi le natif', async () => {
    installFullscreenApi(acceptingRequest())
    const { result } = renderHook(useBanc)
    act(() => result.current.fullscreen.toggle())
    await settle()
    pressEscape()
    await settle()
    expect(result.current.frame.frame).toBe('standard')
    expect(result.current.fullscreen.active).toBe(false)
    expect(document.exitFullscreen).toHaveBeenCalledTimes(1)
  })

  it('une couche ouverte AVANT le hook, qui se retire aussitôt : seule elle se ferme', () => {
    installFullscreenApi(acceptingRequest(), false)
    // LE PIRE ORDRE : la couche s'abonne au clavier AVANT le hook, et se retire du DOM pendant
    // la frappe même (comme React vidant sa mise à jour entre deux écouteurs).
    const couche = document.createElement('div')
    couche.setAttribute(REPLAY_ESCAPE_LAYER_ATTR, '')
    document.body.appendChild(couche)
    const fermerLaCouche = (e: KeyboardEvent) => {
      if (e.key === 'Escape') couche.remove()
    }
    window.addEventListener('keydown', fermerLaCouche)
    const { result } = renderHook(useBanc)
    act(() => result.current.frame.toggleFullFrame())

    pressEscape()
    expect(couche.isConnected).toBe(false)
    expect(result.current.frame.frame).toBe('plein-cadre')

    pressEscape()
    expect(result.current.frame.frame).toBe('standard')
    window.removeEventListener('keydown', fermerLaCouche)
  })

  it('avec le vrai menu de vitesse : Échap le referme, puis rend le cadre', () => {
    installFullscreenApi(acceptingRequest(), false)
    const { result } = renderHook(useBanc)
    render(<ReplaySpeedMenu speed={1} onSetSpeed={() => {}} locale="fr" />)
    act(() => result.current.frame.toggleFullFrame())
    fireEvent.click(screen.getByRole('button', { name: 'Vitesse' }))
    expect(screen.getByRole('group', { name: 'Vitesse' })).toHaveAttribute(REPLAY_ESCAPE_LAYER_ATTR)

    pressEscape()
    expect(screen.queryByRole('group', { name: 'Vitesse' })).toBeNull()
    expect(result.current.frame.frame).toBe('plein-cadre')

    pressEscape()
    expect(result.current.frame.frame).toBe('standard')
  })
})

describe('useReplayFullscreen — pendant un export vidéo', () => {
  it('le geste du système est désactivé, depuis chaque cadre : ni cadre ni natif ne bougent', () => {
    const request = acceptingRequest()
    installFullscreenApi(request)
    const { result } = renderHook(useBanc)
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    expect(result.current.fullscreen.disabled).toBe(true)
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('standard')
    expect(request).not.toHaveBeenCalled()
    act(() => requestExportLayout(null))
    expect(result.current.fullscreen.disabled).toBe(false)
    act(() => result.current.frame.toggleCinema())
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    act(() => result.current.fullscreen.toggle())
    expect(result.current.frame.frame).toBe('cinema')
    expect(request).not.toHaveBeenCalled()
  })
})
