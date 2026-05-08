/**
 * Tests composant — MultiSelectFilter.
 *
 * Couvre l'affichage du count, le grayout des options à count=0 et le
 * comportement du toggle (notamment : une option à count=0 mais déjà
 * cochée doit pouvoir être décochée).
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MultiSelectFilter, type MultiSelectOption } from './MultiSelectFilter'

function openDropdown() {
  // Le bouton porte le placeholder ; cliquer l'ouvre.
  const trigger = screen.getByRole('button')
  fireEvent.click(trigger)
}

describe('MultiSelectFilter', () => {
  it("n'affiche rien si options vide et alwaysShow=false", () => {
    const { container } = render(
      <MultiSelectFilter
        options={[]}
        selected={new Set()}
        toggle={() => {}}
        placeholder="Outcome"
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('affiche les counts à droite des labels', () => {
    const opts: MultiSelectOption[] = [
      { value: 'win', label: 'Victoire', count: 12 },
      { value: 'loss', label: 'Défaite', count: 5 },
    ]
    render(
      <MultiSelectFilter
        options={opts}
        selected={new Set()}
        toggle={() => {}}
        placeholder="Outcome"
      />,
    )
    openDropdown()
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('grise (disabled checkbox) une option à count=0 non cochée', () => {
    const opts: MultiSelectOption[] = [
      { value: 'win', label: 'Victoire', count: 4 },
      { value: 'loss', label: 'Défaite', count: 0 },
    ]
    const toggle = vi.fn()
    render(
      <MultiSelectFilter
        options={opts}
        selected={new Set()}
        toggle={toggle}
        placeholder="Outcome"
      />,
    )
    openDropdown()
    const checkboxes = screen.getAllByRole('checkbox') as HTMLInputElement[]
    // Win = enabled, Loss = disabled
    expect(checkboxes[0].disabled).toBe(false)
    expect(checkboxes[1].disabled).toBe(true)

    // Cliquer sur Loss (count=0) ne doit PAS déclencher toggle
    fireEvent.click(checkboxes[1])
    expect(toggle).not.toHaveBeenCalled()
  })

  it('garde une option à count=0 cliquable si elle est déjà cochée (pour la décocher)', () => {
    const opts: MultiSelectOption[] = [
      { value: 'loss', label: 'Défaite', count: 0 },
    ]
    const toggle = vi.fn()
    render(
      <MultiSelectFilter
        options={opts}
        selected={new Set(['loss'])}
        toggle={toggle}
        placeholder="Outcome"
      />,
    )
    openDropdown()
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement
    expect(checkbox.disabled).toBe(false)
    expect(checkbox.checked).toBe(true)

    fireEvent.click(checkbox)
    expect(toggle).toHaveBeenCalledWith('loss')
  })

  it('toggle est appelé avec la bonne value quand on coche une option active', () => {
    const opts: MultiSelectOption[] = [
      { value: 'win', label: 'Victoire', count: 12 },
    ]
    const toggle = vi.fn()
    render(
      <MultiSelectFilter
        options={opts}
        selected={new Set()}
        toggle={toggle}
        placeholder="Outcome"
      />,
    )
    openDropdown()
    fireEvent.click(screen.getByRole('checkbox'))
    expect(toggle).toHaveBeenCalledWith('win')
  })

  it("n'affiche pas de count si l'option n'en porte pas (compat)", () => {
    const opts: MultiSelectOption[] = [
      { value: 'all', label: 'Tous' }, // pas de count
    ]
    render(
      <MultiSelectFilter
        options={opts}
        selected={new Set()}
        toggle={() => {}}
        placeholder="Filtre"
      />,
    )
    openDropdown()
    // Le label est visible
    expect(screen.getByText('Tous')).toBeInTheDocument()
    // Pas de chiffre rendu à côté (le span count est conditionnel)
    expect(screen.queryByText(/^\d+$/)).toBeNull()
  })
})
