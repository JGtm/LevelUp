/**
 * AnalysisPendingBadge — fermeture : l'infobulle ouverte au survol se ferme à Échap, et Échap n'ouvre rien.
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { AnalysisPendingBadge } from './analysis-pending-badge'

function badge() {
  render(<AnalysisPendingBadge locale="fr" inProgress={false} parts={[]} />)
  return {
    pastille: screen.getByRole('button'),
    // L'infobulle reste dans le DOM, masquée par `hidden`.
    infobulle: () => document.querySelector('[role="tooltip"]') as HTMLElement,
  }
}

describe('AnalysisPendingBadge — fermeture', () => {
  it('Échap ferme l’infobulle ouverte au survol', () => {
    const { pastille, infobulle } = badge()
    expect(infobulle()).toHaveAttribute('hidden')
    fireEvent.mouseEnter(pastille)
    expect(infobulle()).not.toHaveAttribute('hidden')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(infobulle()).toHaveAttribute('hidden')
  })

  it('une autre touche ne ferme pas ; l’infobulle fermée ne s’ouvre pas à Échap', () => {
    const { pastille, infobulle } = badge()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(infobulle()).toHaveAttribute('hidden')
    fireEvent.mouseEnter(pastille)
    fireEvent.keyDown(window, { key: 'ArrowDown' })
    expect(infobulle()).not.toHaveAttribute('hidden')
  })

  it('le défilement la ferme encore', () => {
    const { pastille, infobulle } = badge()
    fireEvent.mouseEnter(pastille)
    fireEvent.scroll(window)
    expect(infobulle()).toHaveAttribute('hidden')
  })
})
