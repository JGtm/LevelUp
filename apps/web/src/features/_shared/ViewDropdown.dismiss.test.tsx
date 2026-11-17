/**
 * Fermeture de ViewDropdown : Échap, clic dehors, clic dedans, et écouteurs absents tant que le menu est fermé.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { ViewDropdown } from './ViewDropdown'

const labels = { view: 'Vue', viewAll: 'Tous', viewSolo: 'Solo', viewSquad: 'Escouade' }

function rendu(onChange: () => void = () => {}) {
  return render(
    <div>
      <p>hors du menu</p>
      <ViewDropdown value="all" onChange={onChange} labels={labels} />
    </div>,
  )
}

function ouvre() {
  const declencheur = screen.getByRole('button', { name: /Vue/ })
  fireEvent.click(declencheur)
  expect(screen.getByRole('button', { name: 'Solo' })).toBeInTheDocument()
  return declencheur
}

describe('ViewDropdown — fermeture', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('Échap ferme le menu et rend le focus au bouton', () => {
    rendu()
    const declencheur = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('button', { name: 'Solo' })).toBeNull()
    expect(document.activeElement).toBe(declencheur)
  })

  it('un clic hors du menu le ferme', () => {
    rendu()
    ouvre()
    fireEvent.mouseDown(screen.getByText('hors du menu'))
    expect(screen.queryByRole('button', { name: 'Solo' })).toBeNull()
  })

  it('un clic dans le menu ne le ferme pas', () => {
    rendu()
    ouvre()
    fireEvent.mouseDown(screen.getByRole('button', { name: 'Solo' }))
    expect(screen.getByRole('button', { name: 'Solo' })).toBeInTheDocument()
  })

  it('choisir une option notifie et ferme le menu', () => {
    const onChange = vi.fn()
    rendu(onChange)
    ouvre()
    fireEvent.click(screen.getByRole('button', { name: 'Escouade' }))
    expect(onChange).toHaveBeenCalledWith('squad')
    expect(screen.queryByRole('button', { name: 'Solo' })).toBeNull()
  })

  it('n écoute rien tant que le menu est fermé', () => {
    const ajoute = vi.spyOn(document, 'addEventListener')
    const ajouteFenetre = vi.spyOn(window, 'addEventListener')
    rendu()
    expect(ajoute.mock.calls.filter(([type]) => type === 'mousedown')).toHaveLength(0)
    expect(ajouteFenetre.mock.calls.filter(([type]) => type === 'keydown')).toHaveLength(0)
    ouvre()
    expect(ajoute.mock.calls.filter(([type]) => type === 'mousedown')).toHaveLength(1)
    expect(ajouteFenetre.mock.calls.filter(([type]) => type === 'keydown')).toHaveLength(1)
  })
})
