/**
 * AssetCard.test.tsx — carte du tab « assets » (bordure droite).
 *
 * GH-6 (2026-07-08, décision utilisateur ferme) : sur les cartes MÉDAILLES, la
 * description tronquée sous la carte est supprimée (image + nom seulement) ; la
 * description COMPLÈTE reste au survol via le `title` du conteneur. Maps/armes :
 * inchangé (description sous la carte si présente).
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { AssetMeta } from '@/lib/api/types'
import { AssetCard } from './AssetCard'

const MEDAL: AssetMeta = {
  id: '1',
  name: 'Double Kill', // nom et description servis dans la langue de la requête
  description: 'Kill 2 enemies within 4 seconds.',
  image_url: '/static/medals/halo_infinite/1.png',
} as AssetMeta

describe('AssetCard — description médaille (GH-6)', () => {
  it('ne rend PAS la description tronquée sous une carte médaille', () => {
    render(<AssetCard asset={MEDAL} kind="medals" />)
    // Le nom reste affiché…
    expect(screen.getByText('Double Kill')).toBeInTheDocument()
    // …mais la description n'est PAS rendue comme texte sous la carte.
    expect(screen.queryByText('Kill 2 enemies within 4 seconds.')).toBeNull()
  })

  it('conserve la description complète au survol (title du conteneur)', () => {
    const { container } = render(<AssetCard asset={MEDAL} kind="medals" />)
    const card = container.querySelector('div[title]') as HTMLElement
    expect(card.getAttribute('title')).toContain('Kill 2 enemies within 4 seconds.')
  })

  it('rend toujours la description sous une carte NON-médaille (map)', () => {
    render(<AssetCard asset={MEDAL} kind="maps" />)
    expect(screen.getAllByText('Kill 2 enemies within 4 seconds.').length).toBeGreaterThan(0)
  })
})
