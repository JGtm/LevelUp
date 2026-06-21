/**
 * Tests unitaires — MedalIcon (lot FRONT title-agnostic médailles).
 *
 * Vérifie la bascule sprite/img selon la présence de `spriteSheet` :
 *  - sprite présent  → <div role="img"> avec background-image + background-position
 *  - sprite absent   → <img src={imageUrl}> (comportement HINF, masquage onError)
 */
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MedalIcon } from './MedalIcon'

describe('MedalIcon', () => {
  it('rend un sprite (background-image + position) quand spriteSheet est fourni', () => {
    render(
      <MedalIcon
        label="Perfect Kill"
        spriteSheet="https://example.com/medals.png"
        spriteLeft={148}
        spriteTop={74}
        spriteWidth={74}
        spriteHeight={74}
        size={40}
      />,
    )
    const node = screen.getByRole('img', { name: 'Perfect Kill' })
    expect(node.tagName).toBe('DIV')
    // La cellule interne porte le background-image + background-position.
    const cell = node.firstElementChild as HTMLElement
    expect(cell).toBeTruthy()
    expect(cell.style.backgroundImage).toContain('https://example.com/medals.png')
    expect(cell.style.backgroundPosition).toBe('-148px -74px')
    // Aucune balise <img> émise en mode sprite.
    expect(node.querySelector('img')).toBeNull()
  })

  it('rend un <img src=imageUrl> quand spriteSheet est absent (HINF)', () => {
    render(<MedalIcon label="Killtacular" imageUrl="/static/medals/hi/123.png" size={32} />)
    const img = screen.getByRole('img', { name: 'Killtacular' }) as HTMLImageElement
    expect(img.tagName).toBe('IMG')
    expect(img.getAttribute('src')).toBe('/static/medals/hi/123.png')
  })

  it('masque l\'<img> en cas d\'erreur de chargement (404 PNG)', () => {
    render(<MedalIcon label="Killtacular" imageUrl="/static/medals/hi/404.png" />)
    const img = screen.getByRole('img', { name: 'Killtacular' }) as HTMLImageElement
    fireEvent.error(img)
    expect(img.style.display).toBe('none')
  })

  it('ne rend rien sans sprite ni imageUrl', () => {
    const { container } = render(<MedalIcon label="Vide" />)
    expect(container.firstChild).toBeNull()
  })
})
