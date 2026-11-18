/**
 * MultiSelectFilter — fermeture : Échap et un appui hors du filtre ferment la liste ouverte ; un appui dedans non.
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MultiSelectFilter } from './MultiSelectFilter'

function ouvre() {
  render(
    <div>
      <button data-testid="dehors">dehors</button>
      <MultiSelectFilter
        options={[{ value: 'win', label: 'Victoire' }]}
        selected={new Set()}
        toggle={() => {}}
        placeholder="Résultat"
      />
    </div>,
  )
  fireEvent.click(screen.getByRole('button', { name: /Résultat/ }))
  expect(screen.getByRole('checkbox')).toBeInTheDocument()
}

describe('MultiSelectFilter — fermeture', () => {
  it('Échap ferme la liste', () => {
    ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('checkbox')).toBeNull()
  })

  it('un appui hors du filtre ferme, un appui dans la liste ne ferme pas', () => {
    ouvre()
    fireEvent.mouseDown(screen.getByRole('checkbox'))
    expect(screen.getByRole('checkbox')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(screen.queryByRole('checkbox')).toBeNull()
  })
})
