/**
 * Tests de la bascule partagée : état pressé, clic, groupe nommé.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { Segmented } from './segmented'

const OPTIONS = [
  { value: 7, label: '7 j' },
  { value: 30, label: '30 j' },
  { value: 90, label: '90 j' },
] as const

describe('Segmented', () => {
  it('un groupe nommé, un seul bouton pressé : celui de la valeur courante', () => {
    render(<Segmented options={OPTIONS} value={30} onChange={() => {}} ariaLabel="Horizon" />)
    expect(screen.getByRole('group', { name: 'Horizon' })).toBeInTheDocument()
    const presses = screen
      .getAllByRole('button')
      .filter((b) => b.getAttribute('aria-pressed') === 'true')
    expect(presses.map((b) => b.textContent)).toEqual(['30 j'])
  })

  it('un clic remonte la valeur de l’option cliquée', () => {
    const onChange = vi.fn()
    render(<Segmented options={OPTIONS} value={30} onChange={onChange} ariaLabel="Horizon" />)
    fireEvent.click(screen.getByRole('button', { name: '90 j' }))
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith(90)
  })
})
