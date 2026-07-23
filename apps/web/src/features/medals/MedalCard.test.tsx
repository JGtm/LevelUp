/**
 * Tests — MedalCard.
 * Vérifie : compteur ROUGE (token destructive) + aria « jamais obtenue » quand
 * count===0 ; compteur normal (foreground) + aria « obtenue N fois » sinon ;
 * pastille de rareté pour une clé Halo Infinite (heroic) ; PAS de pastille pour
 * une difficulty_key numérique Halo 5.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MedalCard } from './MedalCard'
import type { MedalSummaryItem } from '@/lib/api/types'

function medal(over: Partial<MedalSummaryItem>): MedalSummaryItem {
  return {
    medal_id: 1,
    name: 'Perfection',
    description: '',
    difficulty: 'Heroic',
    difficulty_key: 'heroic',
    difficulty_rank: 1,
    category: 'multikill',
    super_section: 'classics',
    personal_score: 0,
    count: 0,
    sort: 0,
    image_url: 'http://x/m.png',
    ...over,
  }
}

describe('MedalCard', () => {
  it('count===0 : compteur en token destructive (rouge) + aria « jamais obtenue » + icône estompée', () => {
    const { container } = render(<MedalCard item={medal({ count: 0, difficulty_key: 'heroic' })} locale="fr" />)
    const counter = screen.getByText('0')
    expect(counter.className).toContain('text-destructive')
    expect(counter.className).not.toContain('text-foreground')
    expect(container.querySelector('[aria-label="Jamais obtenue"]')).toBeTruthy()
    expect(container.querySelector('.opacity-60')).toBeTruthy()
    // Rareté Halo Infinite (heroic) → pastille affichée.
    expect(screen.getByText('Héroïque')).toBeTruthy()
  })

  it('count>0 : compteur en token foreground (pas destructive) + aria « obtenue N fois »', () => {
    const { container } = render(
      <MedalCard item={medal({ count: 5, difficulty_key: 'legendary', name: 'Assassin' })} locale="fr" />,
    )
    const counter = screen.getByText('5')
    expect(counter.className).toContain('text-foreground')
    expect(counter.className).not.toContain('text-destructive')
    expect(container.querySelector('[aria-label="Obtenue 5 fois"]')).toBeTruthy()
    expect(screen.getByText('Légendaire')).toBeTruthy()
  })

  it('Halo 5 (difficulty_key numérique) : AUCUNE pastille de rareté factice', () => {
    render(<MedalCard item={medal({ count: 3, difficulty_key: '2', name: 'Assassinat' })} locale="fr" />)
    expect(screen.getByText('3')).toBeTruthy()
    expect(screen.queryByText('Héroïque')).toBeNull()
    expect(screen.queryByText('Légendaire')).toBeNull()
    expect(screen.queryByText('Mythique')).toBeNull()
    expect(screen.queryByText('Normale')).toBeNull()
  })
})
