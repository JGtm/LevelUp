import { afterEach, describe, expect, it } from 'vitest'
import { focusByKeyboard, focusByMouse } from '@/test/focus'
import { isKeyboardFocus } from './focusVisible'

/** Un bouton dont le gestionnaire de focus note la réponse de `isKeyboardFocus` au moment de l'événement. */
function boutonEcoute(): { button: HTMLButtonElement; reponses: boolean[] } {
  const button = document.createElement('button')
  const reponses: boolean[] = []
  button.addEventListener('focus', (e) => reponses.push(isKeyboardFocus(e.target)))
  document.body.appendChild(button)
  return { button, reponses }
}

describe('isKeyboardFocus', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('vrai pour un focus venu du clavier', () => {
    const { button, reponses } = boutonEcoute()
    focusByKeyboard(button)
    expect(reponses).toEqual([true])
  })

  it('faux pour un focus venu de la souris', () => {
    const { button, reponses } = boutonEcoute()
    focusByMouse(button)
    expect(reponses).toEqual([false])
  })

  it('faux pour un élément qui ne porte pas le focus', () => {
    const { button } = boutonEcoute()
    expect(isKeyboardFocus(button)).toBe(false)
  })

  it('repli vrai quand le sélecteur est inconnu du navigateur', () => {
    const { button } = boutonEcoute()
    button.matches = (() => {
      throw new DOMException('not a valid selector', 'SyntaxError')
    }) as unknown as Element['matches']
    expect(isKeyboardFocus(button)).toBe(true)
  })

  it('repli vrai sans cible ou pour une cible qui n’est pas un élément', () => {
    expect(isKeyboardFocus(null)).toBe(true)
    expect(isKeyboardFocus(window)).toBe(true)
  })
})
