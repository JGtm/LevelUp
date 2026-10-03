/**
 * Tests — WeaponIcon.
 *
 * Ce qui est verrouillé ici : le MODE de rendu vient du back (`tinted`), jamais du
 * titre ni de la forme de l'URL, et la teinte ne descend jamais à un hex.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { WeaponIcon } from './WeaponIcon'

describe('WeaponIcon', () => {
  it('rend un masque teintable quand le back déclare tinted', () => {
    render(<WeaponIcon imageUrl="/static/w/jeu/contour-01.png" tinted label="BR75" />)
    const el = screen.getByRole('img', { name: 'BR75' })
    expect(el.tagName).toBe('SPAN')
    expect(el).toHaveStyle({ maskImage: 'url(/static/w/jeu/contour-01.png)' })
    // Sans token : la couleur de texte héritée, donc le thème décide.
    expect((el.getAttribute('style') ?? '').toLowerCase()).toContain('background-color: currentcolor')
  })

  it('teinte par TOKEN sémantique — jamais un hex', () => {
    render(<WeaponIcon imageUrl="/static/w/jeu/contour-01.png" tinted label="BR75" token="team-ally" />)
    const el = screen.getByRole('img', { name: 'BR75' })
    const bg = el.getAttribute('style') ?? ''
    expect(bg).toContain('var(--ac-team-ally)')
    expect(bg).not.toMatch(/#[0-9a-fA-F]{6}/)
  })

  it('rend une image finie quand le back ne déclare pas tinted', () => {
    render(<WeaponIcon imageUrl="/static/w/Grenade.png" label="Grenade" />)
    const el = screen.getByRole('img', { name: 'Grenade' })
    expect(el.tagName).toBe('IMG')
    expect(el).toHaveAttribute('src', '/static/w/Grenade.png')
    // Une image finie n'est PAS masquée : la teinter l'aplatirait en silhouette.
    expect(el.getAttribute('style') ?? '').not.toContain('mask-image')
  })

  it('ne rend rien sans URL — le repli sur le libellé appartient au caller', () => {
    const { container } = render(<WeaponIcon imageUrl={null} label="Inconnue" />)
    expect(container).toBeEmptyDOMElement()
  })
})
