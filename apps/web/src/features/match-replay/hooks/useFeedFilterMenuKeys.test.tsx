/**
 * useFeedFilterMenuKeys — le clavier du menu de filtre, hors du composant : entrée du focus,
 * flèches en boucle, Début et Fin, Tab, touches retenues, Échap laissée passer.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { nextMenuIndex, useFeedFilterMenuKeys } from './useFeedFilterMenuKeys'

interface SondeProps {
  open: boolean
  setOpen: (open: boolean) => void
  activeIndex?: number
}

function Sonde({ open, setOpen, activeIndex = 1 }: SondeProps) {
  const wrapRef = useRef<HTMLDivElement>(null)
  const { onMenuKeyDown, onTriggerKeyDown } = useFeedFilterMenuKeys({ open, setOpen, wrapRef })
  return (
    <div ref={wrapRef}>
      <button data-testid="declencheur" onKeyDown={onTriggerKeyDown} />
      {open && (
        <div role="menu" data-testid="menu" onKeyDown={onMenuKeyDown}>
          {['a', 'b', 'c'].map((id, i) => (
            <button key={id} role="menuitemradio" aria-checked={i === activeIndex} data-testid={id} />
          ))}
        </div>
      )}
    </div>
  )
}

/** Les touches qui remontent jusqu'à la fenêtre : celles que le menu ne retient pas. */
function toucheRemontee(fire: () => void): boolean {
  const vue = vi.fn()
  window.addEventListener('keydown', vue)
  fire()
  window.removeEventListener('keydown', vue)
  return vue.mock.calls.length > 0
}

describe('nextMenuIndex', () => {
  it('Début et Fin ignorent la position ; suivant et précédent bouclent', () => {
    expect(nextMenuIndex('first', 2, 4)).toBe(0)
    expect(nextMenuIndex('last', 0, 4)).toBe(3)
    expect(nextMenuIndex('next', 3, 4)).toBe(0)
    expect(nextMenuIndex('next', 1, 4)).toBe(2)
    expect(nextMenuIndex('prev', 0, 4)).toBe(3)
    expect(nextMenuIndex('prev', 2, 4)).toBe(1)
    expect(nextMenuIndex('next', -1, 4)).toBe(0)
    expect(nextMenuIndex('prev', -1, 4)).toBe(2)
  })
})

describe('useFeedFilterMenuKeys', () => {
  it('à l’ouverture, le focus va à l’entrée active, à la première sinon', () => {
    const { rerender } = render(<Sonde open={false} setOpen={vi.fn()} />)
    rerender(<Sonde open setOpen={vi.fn()} />)
    expect(screen.getByTestId('b')).toHaveFocus()
    rerender(<Sonde open={false} setOpen={vi.fn()} activeIndex={-1} />)
    rerender(<Sonde open setOpen={vi.fn()} activeIndex={-1} />)
    expect(screen.getByTestId('a')).toHaveFocus()
  })

  it('les flèches haut et bas déplacent le focus en boucle, Début et Fin vont aux extrémités', () => {
    render(<Sonde open setOpen={vi.fn()} />)
    const menu = screen.getByTestId('menu')
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(screen.getByTestId('c')).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(screen.getByTestId('a')).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    expect(screen.getByTestId('c')).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'Home' })
    expect(screen.getByTestId('a')).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'End' })
    expect(screen.getByTestId('c')).toHaveFocus()
  })

  it('une flèche est consommée (preventDefault), Tab ferme le menu', () => {
    const setOpen = vi.fn()
    render(<Sonde open setOpen={setOpen} />)
    const menu = screen.getByTestId('menu')
    expect(fireEvent.keyDown(menu, { key: 'ArrowDown' })).toBe(false)
    fireEvent.keyDown(menu, { key: 'Tab' })
    expect(setOpen).toHaveBeenCalledExactlyOnceWith(false)
  })

  it('menu ouvert, aucune autre touche ne remonte vers le lecteur, sauf Échap', () => {
    render(<Sonde open setOpen={vi.fn()} />)
    const entree = screen.getByTestId('b')
    for (const key of ['ArrowLeft', 'ArrowRight', 'j', 'l', 'f', 'm', 'r', ' ']) {
      expect(toucheRemontee(() => fireEvent.keyDown(entree, { key })), key).toBe(false)
    }
    expect(toucheRemontee(() => fireEvent.keyDown(entree, { key: 'Escape' }))).toBe(true)
  })

  it('le déclencheur : flèche bas ouvre un menu fermé', () => {
    const setOpen = vi.fn()
    render(<Sonde open={false} setOpen={setOpen} />)
    expect(fireEvent.keyDown(screen.getByTestId('declencheur'), { key: 'ArrowDown' })).toBe(false)
    expect(setOpen).toHaveBeenCalledExactlyOnceWith(true)
  })

  it('le déclencheur : fermé, seul Espace est retenu ; ouvert, tout est retenu', () => {
    const { rerender } = render(<Sonde open={false} setOpen={vi.fn()} />)
    const declencheur = screen.getByTestId('declencheur')
    expect(toucheRemontee(() => fireEvent.keyDown(declencheur, { key: ' ' }))).toBe(false)
    expect(toucheRemontee(() => fireEvent.keyDown(declencheur, { key: 'j' }))).toBe(true)
    rerender(<Sonde open setOpen={vi.fn()} />)
    expect(toucheRemontee(() => fireEvent.keyDown(screen.getByTestId('declencheur'), { key: 'j' }))).toBe(false)
  })
})
