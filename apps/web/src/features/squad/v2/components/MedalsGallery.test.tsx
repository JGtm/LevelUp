import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MedalsGallery } from './MedalsGallery'
import type { MedalsGalleryEntry } from '../types'

const baseLabels = { emptyMatch: 'Aucune médaille' }

describe('MedalsGallery', () => {
  const entries: MedalsGalleryEntry[] = [
    {
      match_id: 'm1',
      medals_by_xuid: {
        main: [
          { medal_id: 100, count: 2, label: 'Killing Spree' },
          { medal_id: 200, count: 1, label: 'Headshot' },
        ],
        f1: [{ medal_id: 100, count: 1, label: 'Killing Spree' }],
      },
    },
    {
      match_id: 'm2',
      medals_by_xuid: {
        main: [{ medal_id: 300, count: 1, label: 'MVP' }],
      },
    },
  ]

  it('rend une carte par match', () => {
    render(<MedalsGallery entries={entries} squadOrder={['main', 'f1']} labels={baseLabels} />)
    expect(screen.getAllByTestId('medals-gallery-card')).toHaveLength(2)
  })

  it('affiche le compteur ×N pour count > 1', () => {
    render(<MedalsGallery entries={entries} squadOrder={['main', 'f1']} labels={baseLabels} />)
    // Killing Spree count=2 → "×2" present
    expect(screen.getByText('×2')).toBeTruthy()
  })

  it("respecte l'ordre du squad pour l'affichage", () => {
    render(<MedalsGallery entries={entries} squadOrder={['main', 'f1']} labels={baseLabels} />)
    // Le rendu interne suit squadOrder; main avant f1 dans le DOM
    const card1 = screen.getAllByTestId('medals-gallery-card')[0]
    const text = card1.textContent ?? ''
    expect(text.indexOf('main')).toBeLessThan(text.indexOf('f1'))
  })

  it('entries vides → composant absent', () => {
    const { container } = render(
      <MedalsGallery entries={[]} squadOrder={['main']} labels={baseLabels} />,
    )
    expect(container.firstChild).toBeNull()
  })
})
