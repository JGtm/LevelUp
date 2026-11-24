/**
 * ReplayDimensionSwitch.test.tsx — le segment 2D / 3D : rien sans la capacité, deux positions exclusives avec
 * elle, la bascule appelle `onChange`, « 2D » et « 3D » ne se traduisent pas, le nom du groupe oui.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { ReplayDimensionSwitch } from './ReplayDimensionSwitch'

describe('ReplayDimensionSwitch', () => {
  it('non disponible : rien n’est rendu', () => {
    const { container } = render(<ReplayDimensionSwitch available={false} value="2d" onChange={vi.fn()} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('disponible : deux positions exclusives, la vue choisie cochée', () => {
    render(<ReplayDimensionSwitch available value="3d" onChange={vi.fn()} locale="fr" />)
    const groupe = screen.getByRole('radiogroup', { name: 'Dimension du rejeu' })
    expect(groupe.querySelectorAll('[role="radio"]')).toHaveLength(2)
    expect(screen.getByRole('radio', { name: '3D' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: '2D' })).toHaveAttribute('aria-checked', 'false')
  })

  it('un clic appelle la bascule avec la vue visée', () => {
    const onChange = vi.fn()
    render(<ReplayDimensionSwitch available value="2d" onChange={onChange} locale="fr" />)
    fireEvent.click(screen.getByRole('radio', { name: '3D' }))
    expect(onChange).toHaveBeenLastCalledWith('3d')
    fireEvent.click(screen.getByRole('radio', { name: '2D' }))
    expect(onChange).toHaveBeenLastCalledWith('2d')
  })

  it('en anglais seul le nom du groupe change', () => {
    render(<ReplayDimensionSwitch available value="2d" onChange={vi.fn()} locale="en" />)
    expect(screen.getByRole('radiogroup', { name: 'Replay dimension' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: '2D' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: '3D' })).toBeTruthy()
  })
})
