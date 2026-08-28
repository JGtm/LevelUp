/**
 * Tests — useReplayShortcuts (le clavier du lecteur).
 *
 * CE QU'ILS PROTÈGENT EN PREMIER, ce n'est pas qu'un raccourci marche : c'est qu'il NE PARTE
 * PAS quand il ne doit pas. Espace depuis un champ de recherche mettrait le rejeu en pause au
 * lieu d'écrire une espace ; Cmd+R rechargerait la page ET rembobinerait le film. Ces deux cas
 * sont les seuls qu'on ne peut pas rattraper d'un second geste.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  TIMELINE_SHORTCUT_ATTR,
  useReplayShortcuts,
  type ReplayShortcutHandlers,
} from './useReplayShortcuts'

function mount(over: Partial<ReplayShortcutHandlers> = {}) {
  const handlers: ReplayShortcutHandlers = {
    togglePlay: vi.fn(),
    seekBy: vi.fn(),
    stepFrames: vi.fn(),
    restart: vi.fn(),
    toggleSound: vi.fn(),
    skipSeconds: 10,
    enabled: true,
    ...over,
  }
  renderHook(() => useReplayShortcuts(handlers))
  return handlers
}

/** Frappe une touche au niveau de la fenêtre, éventuellement depuis une cible donnée. */
function press(key: string, init: KeyboardEventInit & { target?: Element } = {}) {
  const { target, ...rest } = init
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...rest })
  ;(target ?? window).dispatchEvent(event)
  return event
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('useReplayShortcuts — les commandes', () => {
  it('Espace et K mettent en lecture/pause', () => {
    const h = mount()
    press(' ')
    press('k')
    expect(h.togglePlay).toHaveBeenCalledTimes(2)
  })

  it('les flèches et J/L sautent de ±10 s', () => {
    const h = mount()
    press('ArrowLeft')
    expect(h.seekBy).toHaveBeenCalledWith(-10)
    press('ArrowRight')
    expect(h.seekBy).toHaveBeenCalledWith(10)
    press('j')
    press('l')
    expect(h.seekBy).toHaveBeenCalledTimes(4)
  })

  it('« , » et « . » avancent d’une image, dans les deux sens', () => {
    const h = mount()
    press(',')
    expect(h.stepFrames).toHaveBeenCalledWith(-1)
    press('.')
    expect(h.stepFrames).toHaveBeenCalledWith(1)
  })

  it('M commande le son, R recommence — majuscules comprises', () => {
    const h = mount()
    press('m')
    press('M')
    expect(h.toggleSound).toHaveBeenCalledTimes(2)
    press('r')
    press('R')
    expect(h.restart).toHaveBeenCalledTimes(2)
  })

  it('une touche non traitée ne fait rien, et laisse le navigateur tranquille', () => {
    const h = mount()
    const event = press('a')
    expect(event.defaultPrevented).toBe(false)
    expect(h.togglePlay).not.toHaveBeenCalled()
    expect(h.seekBy).not.toHaveBeenCalled()
  })
})

describe('useReplayShortcuts — ce qu’il refuse de capter', () => {
  it('Espace DEPUIS UN CHAMP DE SAISIE écrit une espace, il ne met rien en pause', () => {
    const h = mount()
    const input = document.createElement('input')
    document.body.appendChild(input)
    press(' ', { target: input })
    expect(h.togglePlay).not.toHaveBeenCalled()
  })

  it('la même règle vaut pour une zone de texte et un contenu éditable', () => {
    const h = mount()
    const textarea = document.createElement('textarea')
    const editable = document.createElement('div')
    editable.setAttribute('contenteditable', 'true')
    // jsdom ne dérive pas `isContentEditable` de l'attribut : on pose le fait que le hook lit.
    Object.defineProperty(editable, 'isContentEditable', { value: true })
    document.body.append(textarea, editable)
    press(' ', { target: textarea })
    press('k', { target: editable })
    expect(h.togglePlay).not.toHaveBeenCalled()
  })

  it('Cmd+R et Ctrl+R rechargent la page — ils ne rembobinent pas le film', () => {
    const h = mount()
    press('r', { metaKey: true })
    press('r', { ctrlKey: true })
    expect(h.restart).not.toHaveBeenCalled()
  })

  it('un raccourci avec Alt n’est pas capté non plus', () => {
    const h = mount()
    press('ArrowRight', { altKey: true })
    expect(h.seekBy).not.toHaveBeenCalled()
  })

  it('DÉSACTIVÉ, il n’écoute rien du tout (pas de rejeu à l’écran)', () => {
    const h = mount({ enabled: false })
    press(' ')
    press('ArrowRight')
    expect(h.togglePlay).not.toHaveBeenCalled()
    expect(h.seekBy).not.toHaveBeenCalled()
  })
})

describe('useReplayShortcuts — preventDefault, mais seulement là où il faut', () => {
  it('Espace ne fait pas défiler la page, la flèche ne double pas le pas du curseur', () => {
    mount()
    expect(press(' ').defaultPrevented).toBe(true)
    expect(press('ArrowRight').defaultPrevented).toBe(true)
  })
})

/**
 * LA FRISE GARDE SA FRAPPE (décision utilisateur du 2026-08-28, gate de la planche 2a).
 *
 * CE QUE CES CAS FIXENT : un `input[type=range]` est un champ de saisie pour le navigateur, donc
 * la garde anti-frappe l'attrapait — et les raccourcis mouraient dès le premier clic sur la
 * frise, c'est-à-dire au moment même où l'on analyse un match. L'exemption est NOMINATIVE (un
 * attribut posé sur ce champ-là), et les deux moitiés de la règle comptent autant l'une que
 * l'autre : la frise répond, le VOLUME — le même élément HTML — continue de ne pas répondre.
 */
describe('useReplayShortcuts — la frise du lecteur est exemptée, elle seule', () => {
  /** Un `input[type=range]` porteur de l'attribut : la frise du lecteur. */
  function timelineInput(): HTMLInputElement {
    const el = document.createElement('input')
    el.type = 'range'
    el.setAttribute(TIMELINE_SHORTCUT_ATTR, '')
    document.body.appendChild(el)
    return el
  }

  /** Un `input[type=range]` ORDINAIRE : le curseur de volume. */
  function volumeInput(): HTMLInputElement {
    const el = document.createElement('input')
    el.type = 'range'
    document.body.appendChild(el)
    return el
  }

  it('FRISE FOCALISÉE : la flèche saute de 10 s, et le pas natif est supprimé', () => {
    const h = mount()
    const event = press('ArrowRight', { target: timelineInput() })
    expect(h.seekBy).toHaveBeenCalledWith(10)
    // Sans `preventDefault`, le champ avancerait AUSSI d'une image : un double pas.
    expect(event.defaultPrevented).toBe(true)
  })

  it('FRISE FOCALISÉE : Espace met en pause, « , » et « . » avancent image par image', () => {
    const h = mount()
    const frise = timelineInput()
    press(' ', { target: frise })
    expect(h.togglePlay).toHaveBeenCalledTimes(1)
    press(',', { target: frise })
    press('.', { target: frise })
    expect(h.stepFrames).toHaveBeenNthCalledWith(1, -1)
    expect(h.stepFrames).toHaveBeenNthCalledWith(2, 1)
  })

  // L'AUTRE MOITIÉ DE LA RÈGLE, et elle n'est pas accessoire : qui vient de cliquer sur le
  // volume et presse ← attend que le volume baisse, pas que le film saute de dix secondes.
  it('VOLUME FOCALISÉ : rien n’est capté, la flèche reste au navigateur', () => {
    const h = mount()
    const event = press('ArrowLeft', { target: volumeInput() })
    expect(h.seekBy).not.toHaveBeenCalled()
    expect(h.togglePlay).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })

  it('un CHAMP TEXTE reste protégé, attribut ou pas', () => {
    const h = mount()
    const input = document.createElement('input')
    document.body.appendChild(input)
    press(' ', { target: input })
    expect(h.togglePlay).not.toHaveBeenCalled()
  })

  it('depuis la frise, un raccourci avec modificateur n’est toujours pas capté', () => {
    const h = mount()
    press('r', { target: timelineInput(), metaKey: true })
    expect(h.restart).not.toHaveBeenCalled()
  })
})
