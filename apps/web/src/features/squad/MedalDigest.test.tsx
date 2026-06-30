/**
 * MedalDigest.test.tsx — correction #9 : le compteur de médailles affiche le
 * symbole multiplicateur « ×N » (ex. ×9) et non un chiffre nu.
 */
import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { getSquadText } from './i18n'
import { MedalDigest } from './MedalDigest'
import type { MedalDigestEntry } from '@/lib/api/types'

function entry(): MedalDigestEntry {
  const medal = { medal_id: 1, label: 'Double Kill', total_count: 9, match_count: 3 }
  return {
    player: 'P1',
    emblem_url: undefined,
    total_count: 9,
    distinct_types: 1,
    avg_per_match: 3,
    peak_in_match: 4,
    top_medals: [medal],
    all_medals: [medal],
  }
}

describe('MedalDigest (correction #9 — symbole ×)', () => {
  it('affiche le compteur de médaille préfixé par × (×9, pas 9 nu)', () => {
    renderWithProviders(
      <MedalDigest entries={[entry()]} mainPlayer="P1" t={getSquadText('fr').medals} />,
    )
    // Le badge de compteur rend « ×9 » (MedalChip). On tolère le caractère × Unicode.
    const matches = screen.getAllByText((_t, node) => node?.textContent === '×9')
    expect(matches.length).toBeGreaterThan(0)
  })
})
