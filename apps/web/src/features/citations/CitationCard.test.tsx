/**
 * Tests — CitationCard (tuile partagée Infinite/H5).
 * Vérifie : anneau (svg) quand paliers ; fallback icône (MedalIcon, pas d'anneau)
 * sans palier ; pied progression vs total à vie ; parité doré pour une maîtrisée.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CitationCard } from './CitationCard'
import type { CitationDisplayItem } from '@/lib/citations/types'

function item(over: Partial<CitationDisplayItem>): CitationDisplayItem {
  return {
    key: 'k', name: 'Kills', imageUrl: 'http://x/i.png', pct: 60,
    tierIndex: 3, tierCount: 5, total: 45, nextTierTarget: 50,
    isMastered: false, isNewlyMastered: false, source: 'native',
    ...over,
  }
}

describe('CitationCard', () => {
  it('rend un anneau (svg) + paliers + progression vers le prochain palier quand tierée et non maîtrisée', () => {
    const { container } = render(<CitationCard item={item({})} locale="fr" />)
    expect(container.querySelector('svg')).toBeTruthy()
    expect(screen.getByText('3/5')).toBeTruthy()
    expect(screen.getByText('45/50')).toBeTruthy()
  })

  it('maîtrisée native : anneau + paliers pleins + total à vie localisé (pas de progression)', () => {
    render(<CitationCard item={item({ tierIndex: 5, total: 1234, isMastered: true, nextTierTarget: 0 })} locale="en" />)
    expect(screen.getByText('5/5')).toBeTruthy()
    expect(screen.getByText('1,234')).toBeTruthy() // en-US grouping
    expect(screen.queryByText(/\/0$/)).toBeNull()
  })

  it('sans palier (tier_count 0) : fallback icône (pas d’anneau svg) + total à vie pour la source native', () => {
    const { container } = render(<CitationCard item={item({ tierCount: 0, tierIndex: 0, total: 7 })} locale="fr" />)
    expect(container.querySelector('svg')).toBeNull()
    expect(screen.getByText('7')).toBeTruthy()
  })

  it('source infinite : anneau TOUJOURS rendu même sans palier (parité historique)', () => {
    const { container } = render(<CitationCard item={item({ source: 'infinite', tierCount: 0, tierIndex: 0 })} locale="fr" />)
    expect(container.querySelector('svg')).toBeTruthy()
  })

  it('source infinite non maîtrisée : pas de total à vie nu (uniquement progression palier)', () => {
    render(<CitationCard item={item({ source: 'infinite' })} locale="fr" />)
    expect(screen.getByText('45/50')).toBeTruthy()
    expect(screen.queryByText('45')).toBeNull()
  })
})
