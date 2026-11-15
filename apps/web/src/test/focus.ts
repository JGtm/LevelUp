/**
 * Donner le focus en tests comme le ferait un navigateur. jsdom devine `:focus-visible` d'après les
 * dernières interactions qu'il a vues, et sa réponse dépend de l'ordre des requêtes de la page de test
 * (un `getByRole` placé avant le premier focus la fige à faux) : on fixe donc la réponse du sélecteur le
 * temps d'un geste. Le focus reste un vrai focus (événements, `document.activeElement`, gestionnaires React).
 */
import { act } from '@testing-library/react'
import { vi } from 'vitest'

/** Exécute `geste` pendant que `:focus-visible` répond `focusVisible` à tout élément. */
export function withFocusVisible(focusVisible: boolean, geste: () => void): void {
  const matches = Element.prototype.matches
  const spy = vi.spyOn(Element.prototype, 'matches').mockImplementation(function (this: Element, selector: string) {
    return selector === ':focus-visible' ? focusVisible : matches.call(this, selector)
  })
  try {
    act(geste)
  } finally {
    spy.mockRestore()
  }
}

/** Focus venu du clavier (Tab) : `:focus-visible` vrai. */
export function focusByKeyboard(element: HTMLElement): void {
  withFocusVisible(true, () => element.focus())
}

/** Focus venu de la souris : `:focus-visible` faux. */
export function focusByMouse(element: HTMLElement): void {
  withFocusVisible(false, () => element.focus())
}
