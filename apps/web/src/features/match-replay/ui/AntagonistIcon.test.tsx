/**
 * Tests — AntagonistIcon : l'icône d'un titre (némésis, souffre-douleur), partagée par la frise et la fiche.
 *
 * Ils protègent :
 *  1. L'IMAGE EST UN MASQUE sur un aplat `currentColor` : l'icône prend la couleur du texte qui
 *     l'entoure (ou celle qu'on lui pose), jamais une couleur écrite ici ;
 *  2. LE RÔLE CHOISIT L'IMAGE : némésis ou souffre-douleur, la variante noire (alpha seul) ;
 *  3. L'ICÔNE EST DÉCORATIVE : son nom accessible est celui du bouton ou du libellé qui la porte.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { AntagonistIcon } from './AntagonistIcon'

function icon(container: HTMLElement): HTMLElement {
  return container.querySelector('[data-antagonist-icon]') as HTMLElement
}

describe('AntagonistIcon', () => {
  it('la némésis masque l image de la némésis', () => {
    const { container } = render(<AntagonistIcon role="nemesis" size={12} />)
    expect(icon(container).getAttribute('data-antagonist-icon')).toBe('nemesis')
    expect(icon(container).getAttribute('style')).toContain('/icons/nemesis-black.png')
  })

  it('le souffre-douleur masque l image du souffre-douleur', () => {
    const { container } = render(<AntagonistIcon role="victim" size={12} />)
    expect(icon(container).getAttribute('data-antagonist-icon')).toBe('victim')
    expect(icon(container).getAttribute('style')).toContain('/icons/victim-black.png')
  })

  it('la taille est celle de la boîte, en pixels', () => {
    const { container } = render(<AntagonistIcon role="nemesis" size={14} />)
    expect(icon(container).style.width).toBe('14px')
    expect(icon(container).style.height).toBe('14px')
  })

  it('l aplat suit la couleur du texte', () => {
    const { container } = render(<AntagonistIcon role="nemesis" size={12} />)
    expect(icon(container).style.backgroundColor.toLowerCase()).toBe('currentcolor')
  })

  it('une encre passée en paramètre devient la couleur du texte de l icône', () => {
    const { container } = render(<AntagonistIcon role="victim" size={12} ink="var(--ac-team-enemy)" />)
    expect(icon(container).style.color).toBe('var(--ac-team-enemy)')
  })

  it('l icône est décorative', () => {
    const { container } = render(<AntagonistIcon role="nemesis" size={12} />)
    expect(icon(container).getAttribute('aria-hidden')).toBe('true')
  })
})
