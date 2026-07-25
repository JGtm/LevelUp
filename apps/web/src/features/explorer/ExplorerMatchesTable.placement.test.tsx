/**
 * Tests unitaires — ExplorerMatchesTable.placement (V72-32, étendu V72-34).
 *
 * Couvre :
 *  - `hasPlacementSignal` : true ssi placement_done ET placement_total sont
 *    tous deux non-nuls (signal du CLASSEMENT, colonne Note/Rang) ;
 *  - `hasPerfPlacementSignal` : idem sur perf_placement_* (signal de la CHAÎNE DE
 *    PERFORMANCE, colonnes Perf/ΔPerf) — INDÉPENDANT du précédent ;
 *  - `PlacementPendingCell` : libellé court FR/EN + tooltip avec le nombre de
 *    matchs restants (total - done), pour les deux variantes.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { ExplorerMatchRow } from '@/lib/api/types'
import {
  hasPerfPlacementSignal,
  hasPlacementSignal,
  PlacementPendingCell,
} from './ExplorerMatchesTable.placement'

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

  it('IGNORE le signal perf : un placement de chaîne perf ne classe pas la Note en placement', () => {
    expect(hasPlacementSignal(row({ perf_placement_done: 8, perf_placement_total: 10 }))).toBe(false)
  })
})

describe('hasPerfPlacementSignal', () => {
  it('true si perf_placement_done ET perf_placement_total sont renseignés', () => {
    expect(hasPerfPlacementSignal(row({ perf_placement_done: 8, perf_placement_total: 10 }))).toBe(true)
  })

  it('false si absents (chaîne perf calibrée ou retard de sync → "-" honnête)', () => {
    expect(hasPerfPlacementSignal(row({ perf_placement_done: null, perf_placement_total: null }))).toBe(false)
  })

  it('false si un seul des deux est renseigné', () => {
    expect(hasPerfPlacementSignal(row({ perf_placement_done: 8, perf_placement_total: null }))).toBe(false)
    expect(hasPerfPlacementSignal(row({ perf_placement_done: null, perf_placement_total: 10 }))).toBe(false)
  })

  it('IGNORE le signal de classement : cas JGtm (LUSR établi, chaîne perf jeune)', () => {
    // Pas de placement_* (LUSR établi) mais chaîne perf sous le seuil : SEUL le
    // signal perf doit répondre — c'est le trou que V72-34 comble.
    const r = row({ placement_done: null, placement_total: null, perf_placement_done: 8, perf_placement_total: 10 })
    expect(hasPlacementSignal(r)).toBe(false)
    expect(hasPerfPlacementSignal(r)).toBe(true)
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

  it('variant="perf" : lit perf_placement_* (cas JGtm 8/10 → 2 matchs restants)', () => {
    render(
      <PlacementPendingCell
        row={row({ perf_placement_done: 8, perf_placement_total: 10 })}
        locale="fr"
        variant="perf"
      />,
    )
    const el = screen.getByText('En placement')
    expect(el).toHaveAttribute('title', expect.stringContaining('2'))
  })

  it('variant="perf" ne lit PAS placement_* (signaux étanches)', () => {
    // placement_* à 1/10 (6 restants si mal lu) vs perf 9/10 (1 restant attendu).
    render(
      <PlacementPendingCell
        row={row({ placement_done: 1, placement_total: 10, perf_placement_done: 9, perf_placement_total: 10 })}
        locale="fr"
        variant="perf"
      />,
    )
    const el = screen.getByText('En placement')
    expect(el).toHaveAttribute('title', expect.stringContaining('1'))
    expect(el.getAttribute('title')).not.toContain('9 ')
  })
})
