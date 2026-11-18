/**
 * InfoTooltip — fermeture : Échap ferme l'infobulle ouverte ; un appui dans le panneau ne la ferme pas.
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { InfoTooltip } from './info-tooltip'

describe('InfoTooltip — fermeture', () => {
  it('Échap ferme l’infobulle ouverte au clic', () => {
    render(<InfoTooltip content="Explication" />)
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('un appui dans le panneau ne ferme pas, un appui dehors ferme', () => {
    render(<InfoTooltip content="Explication" />)
    fireEvent.click(screen.getByRole('button'))
    fireEvent.mouseDown(screen.getByRole('tooltip'))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('tooltip')).toBeNull()
  })
})
