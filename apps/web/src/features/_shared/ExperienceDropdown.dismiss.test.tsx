/**
 * Fermeture d'ExperienceDropdown : Échap, clic dehors, clic dedans, et écouteurs absents tant que le menu est fermé.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { ExperienceDropdown } from './ExperienceDropdown'

const labels = { placeholder: 'Expérience', all: 'Tous', ranked: 'Classé', unranked: 'Non classé' }

function rendu(onChange: () => void = () => {}) {
  return render(
    <div>
      <p>hors du menu</p>
      <ExperienceDropdown value="all" onChange={onChange} counts={[]} labels={labels} />
    </div>,
  )
}

function ouvre() {
  const declencheur = screen.getByRole('button', { name: /Expérience/ })
  fireEvent.click(declencheur)
  expect(screen.getByRole('button', { name: /Classé/ })).toBeInTheDocument()
  return declencheur
}

describe('ExperienceDropdown — fermeture', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('Échap ferme le menu et rend le focus au bouton', () => {
    rendu()
    const declencheur = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('button', { name: /^Classé/ })).toBeNull()
    expect(document.activeElement).toBe(declencheur)
  })

  it('le déclencheur annonce son état ouvert', () => {
    rendu()
    const declencheur = screen.getByRole('button', { name: /Expérience/ })
    expect(declencheur).toHaveAttribute('aria-haspopup', 'listbox')
    expect(declencheur).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(declencheur)
    expect(declencheur).toHaveAttribute('aria-expanded', 'true')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(declencheur).toHaveAttribute('aria-expanded', 'false')
  })

  it('un clic hors du menu le ferme', () => {
    rendu()
    ouvre()
    fireEvent.mouseDown(screen.getByText('hors du menu'))
    expect(screen.queryByRole('button', { name: /^Classé/ })).toBeNull()
  })

  it('un clic dans le menu ne le ferme pas', () => {
    rendu()
    ouvre()
    fireEvent.mouseDown(screen.getByRole('button', { name: /^Classé/ }))
    expect(screen.getByRole('button', { name: /^Classé/ })).toBeInTheDocument()
  })

  it('choisir une option notifie et ferme le menu', () => {
    const onChange = vi.fn()
    rendu(onChange)
    ouvre()
    fireEvent.click(screen.getByRole('button', { name: /^Classé/ }))
    expect(onChange).toHaveBeenCalledWith('ranked')
    expect(screen.queryByRole('button', { name: /^Non classé/ })).toBeNull()
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
