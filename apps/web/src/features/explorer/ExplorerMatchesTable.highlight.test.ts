/**
 * Tests unitaires — ExplorerMatchesTable.highlight (logique pure MVP/LVP).
 *
 * Couvre (sans jsdom — assertions sur les objets style renvoyés) :
 *  - meilleur (vert outcome-win, gras 600) et pire (rouge outcome-loss, gras 500)
 *    sur une colonne hétérogène ;
 *  - `deaths` inversé (le minimum est le meilleur) ;
 *  - colonne uniforme → aucun surlignage ; valeur nulle → neutre ;
 *  - garde ≥ 2 valeurs non nulles ; perf sans tier non extraite ;
 *  - `ownTeamScore` (1er entier du libellé « A - B »).
 */
import { describe, it, expect } from 'vitest'

import type { ExplorerMatchRow } from '@/lib/api/types'
import {
  EXPLORER_INVERTED,
  columnHighlightStyle,
  computeColumnExtremes,
  ownTeamScore,
} from './ExplorerMatchesTable.highlight'

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

describe('ExplorerMatchesTable.highlight — extrêmes & styles', () => {
  it('surligne meilleur (vert 600) et pire (rouge 500) sur une colonne hétérogène', () => {
    const rows = [row({ kills: 2 }), row({ kills: 30 }), row({ kills: 10 })]
    const ex = computeColumnExtremes(rows)
    expect(ex.kills).toEqual({ min: 2, max: 30 })
    const best = columnHighlightStyle('kills', 30, ex)
    const worst = columnHighlightStyle('kills', 2, ex)
    expect(best.backgroundColor).toContain('--ac-outcome-win')
    expect(best.color).toContain('--ac-outcome-win')
    expect(best.fontWeight).toBe(600)
    expect(worst.backgroundColor).toContain('--ac-outcome-loss')
    expect(worst.fontWeight).toBe(500)
    // Valeur intermédiaire → neutre.
    expect(columnHighlightStyle('kills', 10, ex)).toEqual({})
  })

  it('deaths est inversé : le minimum est le meilleur (vert)', () => {
    expect(EXPLORER_INVERTED.deaths).toBe(true)
    const rows = [row({ deaths: 1 }), row({ deaths: 9 }), row({ deaths: 4 })]
    const ex = computeColumnExtremes(rows)
    expect(columnHighlightStyle('deaths', 1, ex).backgroundColor).toContain('--ac-outcome-win')
    expect(columnHighlightStyle('deaths', 9, ex).backgroundColor).toContain('--ac-outcome-loss')
  })

  it('colonne uniforme (toutes valeurs égales) → aucun surlignage', () => {
    const rows = [row({ kda: 2 }), row({ kda: 2 }), row({ kda: 2 })]
    const ex = computeColumnExtremes(rows)
    expect(ex.kda).toEqual({ min: 2, max: 2 })
    expect(columnHighlightStyle('kda', 2, ex)).toEqual({})
  })

  it('valeur nulle → neutre', () => {
    const rows = [row({ perf_score: 80, perf_tier: 1 }), row({ perf_score: 40, perf_tier: 4 })]
    const ex = computeColumnExtremes(rows)
    expect(columnHighlightStyle('perf_score', null, ex)).toEqual({})
  })

  it('garde ≥ 2 valeurs non nulles (une seule valeur → pas de highlight)', () => {
    const rows = [row({ kills: 5 }), row({ kills: null })]
    const ex = computeColumnExtremes(rows)
    expect(ex.kills).toEqual({ min: null, max: null })
    expect(columnHighlightStyle('kills', 5, ex)).toEqual({})
  })

  it('perf_score sans perf_tier n’est pas extrait (cellule « - » non surlignée)', () => {
    const rows = [
      row({ perf_score: 90 }), // pas de tier → exclu
      row({ perf_score: 30, perf_tier: 5 }),
      row({ perf_score: 60, perf_tier: 3 }),
    ]
    const ex = computeColumnExtremes(rows)
    expect(ex.perf_score).toEqual({ min: 30, max: 60 })
  })

  it('ownTeamScore extrait le 1er entier du libellé « A - B »', () => {
    expect(ownTeamScore('50 - 42')).toBe(50)
    expect(ownTeamScore('50-30')).toBe(50)
    expect(ownTeamScore('0 - 5')).toBe(0)
    expect(ownTeamScore('-')).toBeNull()
    expect(ownTeamScore('')).toBeNull()
    expect(ownTeamScore(null)).toBeNull()
  })
})
