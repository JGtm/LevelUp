/**
 * Tests unitaires — InfoTooltip (rendu via portal, DEC-TOOLTIP).
 *
 * Couvre :
 *  - le panneau est rendu dans document.body (portal), HORS d'un conteneur
 *    `overflow-hidden` clippant (régression du clipping KpiCard résolue) ;
 *  - ouverture au clic et au focus ; fermeture au clic extérieur et au scroll ;
 *  - `role="tooltip"` + aria-label du bouton conservés ;
 *  - non-régression : plusieurs instances (≈ consommateurs) restent indépendantes.
 */
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'

import { InfoTooltip } from './info-tooltip'

describe('InfoTooltip — portal (DEC-TOOLTIP)', () => {
  it('rend le panneau dans document.body, hors du conteneur overflow-hidden', () => {
    render(
      <div className="overflow-hidden" data-testid="clipper">
        <InfoTooltip content="Explication détaillée" />
      </div>,
    )
    // Fermé au départ.
    expect(screen.queryByRole('tooltip')).toBeNull()
    // Ouverture au clic sur le bouton (i).
    fireEvent.click(screen.getByRole('button'))
    const panel = screen.getByRole('tooltip')
    expect(panel).toHaveTextContent('Explication détaillée')
    // Le panneau est enfant de document.body (portal), PAS du conteneur clippant.
    const clipper = screen.getByTestId('clipper')
    expect(clipper.contains(panel)).toBe(false)
    expect(document.body.contains(panel)).toBe(true)
    // Style fixed (échappe l'overflow).
    expect(panel).toHaveStyle({ position: 'fixed' })
  })

  it('ouvre au focus du bouton et ferme au clic extérieur', () => {
    render(<InfoTooltip content="abc" />)
    const btn = screen.getByRole('button')
    fireEvent.focus(btn)
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('ferme au scroll (listener capture ajouté à l’ouverture)', () => {
    render(<InfoTooltip content="abc" />)
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.scroll(window)
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('conserve role="tooltip" et l’aria-label du bouton', () => {
    render(<InfoTooltip content="abc" />)
    const btn = screen.getByRole('button')
    expect(btn).toHaveAttribute('aria-label')
    fireEvent.click(btn)
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
  })

  it('non-régression : deux consommateurs restent indépendants', () => {
    render(
      <div>
        <span data-testid="c1">
          <InfoTooltip content="premier" />
        </span>
        <span data-testid="c2">
          <InfoTooltip content="second" />
        </span>
      </div>,
    )
    // Aucun tooltip ouvert au montage.
    expect(screen.queryAllByRole('tooltip')).toHaveLength(0)
    // Ouvrir le premier seulement.
    fireEvent.click(within(screen.getByTestId('c1')).getByRole('button'))
    const panels = screen.getAllByRole('tooltip')
    expect(panels).toHaveLength(1)
    expect(panels[0]).toHaveTextContent('premier')
  })
})
