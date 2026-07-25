/**
 * Tests unitaires — ExplorerMatchesTable.placement (V72-32).
 *
 * Couvre :
 *  - `hasPlacementSignal` : true ssi placement_done ET placement_total sont
 *    tous deux non-nuls (même garde que la colonne Rang) ;
 *  - `PlacementPendingCell` : libellé court FR/EN + tooltip avec le nombre de
 *    matchs restants (total - done), sans rendre la table complète.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { ExplorerMatchRow } from '@/lib/api/types'
import { hasPlacementSignal, PlacementPendingCell } from './ExplorerMatchesTable.placement'

function row(over: Partial<ExplorerMatchRow>): ExplorerMatchRow {
  return {
    match_id: 'm',
    start_time: '',
    start_time_label: '',
    map_ui: '',
    mode_ui: '',
    playlist_label: '',
    outcome_label: '',
    outcome_code: 2,
    score_label: '',
    is_with_friends: false,
    experience_type_label: '',
    ...over,
  }
}

describe('hasPlacementSignal', () => {
  it('true si placement_done ET placement_total sont renseignés', () => {
    expect(hasPlacementSignal(row({ placement_done: 3, placement_total: 10 }))).toBe(true)
  })

  it('false si les deux sont absents (cas structurel, ex. Firefight)', () => {
    expect(hasPlacementSignal(row({ placement_done: null, placement_total: null }))).toBe(false)
  })

  it('false si un seul des deux est renseigné (donnée incohérente, ne pas fabriquer un état)', () => {
    expect(hasPlacementSignal(row({ placement_done: 3, placement_total: null }))).toBe(false)
    expect(hasPlacementSignal(row({ placement_done: null, placement_total: 10 }))).toBe(false)
  })
})

describe('PlacementPendingCell', () => {
  it('FR : libellé court "En placement" + tooltip avec les matchs restants (total - done)', () => {
    render(<PlacementPendingCell row={row({ placement_done: 3, placement_total: 10 })} locale="fr" />)
    const el = screen.getByText('En placement')
    expect(el).toHaveAttribute('title', expect.stringContaining('7'))
  })

  it('EN : libellé court "In placement" (cohérent avec common.home.rank_placement)', () => {
    render(<PlacementPendingCell row={row({ placement_done: 3, placement_total: 10 })} locale="en" />)
    expect(screen.getByText('In placement')).toBeInTheDocument()
  })

  it('dernier match de placement (done === total) → tooltip "0 matchs restants" (pas négatif)', () => {
    render(<PlacementPendingCell row={row({ placement_done: 10, placement_total: 10 })} locale="fr" />)
    const el = screen.getByText('En placement')
    expect(el).toHaveAttribute('title', expect.stringContaining('0'))
  })
})
