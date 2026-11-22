/**
 * Tests — la barre de lecture d'une ligne, sous ses deux décisions : sur combien de lignes elle se range (la frise
 * passe seule en tête quand elle n'aurait plus sa largeur minimale) et sous quelle forme elle se montre au Plein cadre
 * (fine en lecture, pleine à la pause, au survol, au clavier et juste après une touche ; préférence éteinte et autres
 * cadres : toujours pleine ; mouvement réduit : sans transition).
 */
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import type { ReplayFrame } from '../model/replayFrame'
import { MARGE_BASCULE_PX } from '../model/rowLayout'
import { requiredWidthOneLine } from '../model/transportBarLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { makeFrameControl } from '../test/frameControlStub'
import { makeCommands } from '../test/transportStubs'
import { ReplayTransportCompact } from './ReplayTransportCompact'

const T = REPLAY_TEXT.fr

function barOf(container: HTMLElement): HTMLElement {
  return container.querySelector('[data-replay-transport="compact"]') as HTMLElement
}

function renderBar(over: { frame?: ReplayFrame; thinBar?: boolean; playing?: boolean } = {}) {
  const { frame = 'plein-cadre', thinBar = true, playing = true } = over
  const commands = makeCommands({ frameControl: makeFrameControl(frame), thinBar, playing })
  const view = render(<ReplayTransportCompact {...commands} />)
  const rerender = () => view.rerender(<ReplayTransportCompact {...commands} />)
  return { ...view, rerender, root: barOf(view.container) }
}

function mockReducedMotion(matches: boolean) {
  const matchMedia = vi.fn().mockReturnValue({ matches })
  Object.defineProperty(window, 'matchMedia', { value: matchMedia, configurable: true, writable: true })
}

/** Fait répondre `:focus-visible` à tous les éléments (le moteur de test ne sait pas ce qu'est un focus clavier). */
function mockFocusVisible(answer: boolean) {
  const original = Element.prototype.matches
  vi.spyOn(Element.prototype, 'matches').mockImplementation(function (this: Element, selector: string) {
    return selector === ':focus-visible' ? answer : original.call(this, selector)
  })
}

beforeEach(() => mockReducedMotion(false))
afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('La forme de la barre au Plein cadre', () => {
  it('en lecture, au repos : fine — plus basse, réduite, translucide', () => {
    const { root } = renderBar()
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
    expect(root.style.height).toBe(`${C.fineHauteurPx}px`)
    expect((root.firstElementChild as HTMLElement).style.transform).toBe(`scale(${C.fineEchelle})`)
    expect(root.className).toContain('backdrop-blur')
  })

  it('à la pause : pleine', () => {
    const { root } = renderBar({ playing: false })
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    expect(root.style.height).toBe(`${C.barreHauteurPx}px`)
    expect((root.firstElementChild as HTMLElement).style.transform).toBe('')
  })

  it('au survol : pleine ; le pointeur parti, elle le reste le temps du geste récent, puis redevient fine', () => {
    vi.useFakeTimers()
    const { root } = renderBar()
    fireEvent.pointerEnter(root)
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    fireEvent.pointerLeave(root)
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    act(() => void vi.advanceTimersByTime(C.fineRetenuMs - 1))
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    act(() => void vi.advanceTimersByTime(1))
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
  })

  it('une touche du clavier la rend pleine pour la durée du geste récent', () => {
    vi.useFakeTimers()
    const { root } = renderBar()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    act(() => void vi.advanceTimersByTime(C.fineRetenuMs))
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
  })

  it("le focus du clavier sur une commande la rend pleine et opaque, tant qu'il y est", () => {
    vi.useFakeTimers()
    mockFocusVisible(true)
    const { root } = renderBar()
    const speed = screen.getByRole('button', { name: T.speed })
    act(() => speed.focus())
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    expect(root.style.opacity).toBe('')
    act(() => void vi.advanceTimersByTime(10 * C.fineRetenuMs))
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    act(() => speed.blur())
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
  })

  it('un focus donné à la souris (pas un focus clavier) ne garde pas la barre pleine', () => {
    mockFocusVisible(false)
    const { root } = renderBar()
    act(() => screen.getByRole('button', { name: T.speed }).focus())
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
  })

  it('préférence éteinte : toujours pleine, sous le bloc du plan, sans fond propre et à marge étroite', () => {
    const { root } = renderBar({ thinBar: false })
    expect(root).toHaveAttribute('data-replay-bar-form', 'pleine')
    expect(root.className).not.toContain('bg-background')
    expect(root.className).not.toContain('backdrop-blur')
    expect((root.firstElementChild as HTMLElement).style.padding).toBe(`0px ${C.margeXHorsBlocPx}px`)
  })

  it('sur le plan (préférence allumée) : la marge du socle ; dans le bloc : la marge du socle et le fond léger', () => {
    const posee = renderBar({ playing: false })
    expect((posee.root.firstElementChild as HTMLElement).style.padding).toBe(`0px ${C.margeXPx}px`)
    expect(posee.root.className).toContain('bg-background/70')
    cleanup()
    const dansLeBloc = renderBar({ frame: 'standard' })
    expect((dansLeBloc.root.firstElementChild as HTMLElement).style.padding).toBe(`0px ${C.margeXPx}px`)
    expect(dansLeBloc.root.className).toContain('bg-background/40')
  })

  it('Cinéma : sous le bloc, sans fond et à marge étroite, quelle que soit la préférence', () => {
    for (const thinBar of [true, false]) {
      const { root } = renderBar({ frame: 'cinema', thinBar })
      expect(root.className).not.toContain('bg-background')
      expect((root.firstElementChild as HTMLElement).style.padding).toBe(`0px ${C.margeXHorsBlocPx}px`)
      cleanup()
    }
  })

  it.each(['standard', 'cinema'] as const)('%s : toujours pleine, préférence allumée ou non', (frame) => {
    expect(renderBar({ frame, thinBar: true }).root).toHaveAttribute('data-replay-bar-form', 'pleine')
  })

  it("mouvement réduit demandé : aucune transition, la forme change d'un coup", () => {
    mockReducedMotion(true)
    const { root } = renderBar()
    expect(root.style.transition).toBe('')
    expect((root.firstElementChild as HTMLElement).style.transition).toBe('')
  })

  it('sans préférence de mouvement réduit : transitions courtes sur la hauteur et la réduction', () => {
    const { root } = renderBar()
    expect(root.style.transition).toContain('height')
    expect((root.firstElementChild as HTMLElement).style.transition).toContain('transform')
  })
})

describe('Les lignes de la barre', () => {
  let clientWidth = 0

  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(40)
    vi.spyOn(Element.prototype, 'clientWidth', 'get').mockImplementation(() => clientWidth)
  })

  /** Le seuil de la barre rendue : ses groupes visibles, boîte de 40 px (simulée) et marges propres, hors frise. */
  function requiredOf(root: HTMLElement): number {
    const row = root.firstElementChild as HTMLElement
    const groups = Array.from(row.children)
      .filter((el) => !el.hasAttribute('data-replay-bar-timeline'))
      .map((el) => {
        const style = getComputedStyle(el)
        return 40 + (parseFloat(style.marginLeft) || 0) + (parseFloat(style.marginRight) || 0)
      })
    // La marge que la barre montre (sous le bloc, étroite) est celle que la règle compte.
    return requiredWidthOneLine(groups, parseFloat(row.style.paddingLeft))
  }

  it('large : une ligne, la frise prend la place restante', () => {
    clientWidth = 2000
    const { root } = renderBar({ frame: 'cinema' })
    expect(root).toHaveAttribute('data-replay-bar-lines', '1')
    expect(root.style.height).toBe(`${C.barreHauteurPx}px`)
    const slot = root.querySelector('[data-replay-bar-timeline]') as HTMLElement
    expect(slot.style.order).toBe('')
  })

  it('étroit : deux lignes, la frise passe seule en tête sur toute la largeur, les commandes à la suite', () => {
    clientWidth = 300
    const { root } = renderBar({ frame: 'cinema' })
    expect(root).toHaveAttribute('data-replay-bar-lines', '2')
    const slot = root.querySelector('[data-replay-bar-timeline]') as HTMLElement
    expect(slot.style.order).toBe('-1')
    expect(slot.style.flex).toBe('0 0 100%')
    expect(slot.style.height).toBe(`${C.deuxLignesFriseHauteurPx}px`)
    expect(root.style.height).toBe('')
    expect(screen.getByRole('slider', { name: T.time })).toBeInTheDocument()
  })

  it('la frise a sa largeur minimale dans la barre', () => {
    clientWidth = 2000
    const { root } = renderBar({ frame: 'cinema' })
    const frise = root.querySelector('[data-replay-cursor-host]') as HTMLElement
    expect(frise.style.minWidth).toBe(`${C.friseLargeurMinPx}px`)
  })

  it('le seuil se lit sur les groupes mesurés, et la marge de bascule retient les deux lignes', () => {
    clientWidth = 2000
    const view = renderBar({ frame: 'cinema' })
    const required = requiredOf(view.root)
    clientWidth = required - 1
    view.rerender()
    expect(view.root).toHaveAttribute('data-replay-bar-lines', '2')
    clientWidth = required + MARGE_BASCULE_PX - 1
    view.rerender()
    expect(view.root).toHaveAttribute('data-replay-bar-lines', '2')
    clientWidth = required + MARGE_BASCULE_PX
    view.rerender()
    expect(view.root).toHaveAttribute('data-replay-bar-lines', '1')
  })

  it('largeur pas encore mesurée : une ligne', () => {
    clientWidth = 0
    expect(renderBar({ frame: 'cinema' }).root).toHaveAttribute('data-replay-bar-lines', '1')
  })

  it('au Plein cadre, la forme fine sur deux lignes garde les deux lignes (la décision ne dépend pas de la forme)', () => {
    clientWidth = 300
    const { root } = renderBar()
    expect(root).toHaveAttribute('data-replay-bar-form', 'fine')
    expect(root).toHaveAttribute('data-replay-bar-lines', '2')
  })
})
