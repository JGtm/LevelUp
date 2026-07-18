/**
 * Tests unitaires — ExplorerMatchesTable.highlight (logique pure MVP/LVP en
 * BANDE DE DÉCILE).
 *
 * Couvre (sans jsdom — assertions sur les objets style renvoyés) :
 *  - décile HAUT (≥ p90) surligné best (vert outcome-win, gras 600), décile BAS
 *    (≤ p10) worst (rouge outcome-loss, gras 500), teinte douce 16 % ;
 *  - `deaths` inversé (le décile bas est le meilleur) ;
 *  - garde `< MIN_DECILE_SAMPLE` valeurs → aucun surlignage ;
 *  - colonne quasi-uniforme (`p10 === p90`) → neutre ; valeur nulle → neutre ;
 *  - perf sans tier non comptée dans les déciles ;
 *  - `decileCellState` (bande) ; `team_mmr` non inversé (haut = meilleur).
 */
import { describe, it, expect } from 'vitest'

import type { ExplorerMatchRow } from '@/lib/api/types'
import {
  DECILE_TINT_PCT,
  EXPLORER_INVERTED,
  MIN_DECILE_SAMPLE,
  columnHighlightStyle,
  computeColumnDeciles,
  decileCellState,
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

describe('ExplorerMatchesTable.highlight — déciles & styles', () => {
  it('surligne le décile HAUT (best 600) et le décile BAS (worst 500), teinte douce', () => {
    const rows = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100].map((kills) => row({ kills }))
    const d = computeColumnDeciles(rows)
    expect(d.kills).toEqual({ p10: 10, p90: 90 })
    const best = columnHighlightStyle('kills', 100, d)
    expect(best.backgroundColor).toContain('--ac-outcome-win')
    expect(best.backgroundColor).toContain(`${DECILE_TINT_PCT}%`)
    expect(best.color).toContain('--ac-outcome-win')
    expect(best.fontWeight).toBe(600)
    // La borne p90 elle-même est dans la bande (≥ p90).
    expect(columnHighlightStyle('kills', 90, d).fontWeight).toBe(600)
    const worst = columnHighlightStyle('kills', 10, d)
    expect(worst.backgroundColor).toContain('--ac-outcome-loss')
    expect(worst.fontWeight).toBe(500)
    // Milieu de distribution → neutre.
    expect(columnHighlightStyle('kills', 50, d)).toEqual({})
  })

  it('deaths est inversé : décile BAS = best (vert), décile HAUT = worst (rouge)', () => {
    expect(EXPLORER_INVERTED.deaths).toBe(true)
    const rows = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((deaths) => row({ deaths }))
    const d = computeColumnDeciles(rows)
    expect(d.deaths).toEqual({ p10: 1, p90: 9 })
    expect(columnHighlightStyle('deaths', 1, d).backgroundColor).toContain('--ac-outcome-win')
    expect(columnHighlightStyle('deaths', 10, d).backgroundColor).toContain('--ac-outcome-loss')
    expect(columnHighlightStyle('deaths', 5, d)).toEqual({})
  })

  it(`< MIN_DECILE_SAMPLE (${MIN_DECILE_SAMPLE}) valeurs → pas de décile, aucun surlignage`, () => {
    const rows = Array.from({ length: MIN_DECILE_SAMPLE - 1 }, (_, i) => row({ kills: i + 1 }))
    const d = computeColumnDeciles(rows)
    expect(d.kills).toEqual({ p10: null, p90: null })
    expect(columnHighlightStyle('kills', MIN_DECILE_SAMPLE - 1, d)).toEqual({})
  })

  it('colonne quasi-uniforme (p10 === p90) → aucun surlignage', () => {
    const rows = Array.from({ length: 10 }, () => row({ kda: 2 }))
    const d = computeColumnDeciles(rows)
    expect(d.kda).toEqual({ p10: 2, p90: 2 })
    expect(columnHighlightStyle('kda', 2, d)).toEqual({})
  })

  it('valeur nulle → neutre (même avec des déciles valides)', () => {
    const rows = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100].map((kills) => row({ kills }))
    const d = computeColumnDeciles(rows)
    expect(columnHighlightStyle('kills', null, d)).toEqual({})
  })

  it('perf_score sans perf_tier n’est pas comptée dans les déciles', () => {
    const withTier = Array.from({ length: 10 }, (_, i) =>
      row({ perf_score: (i + 1) * 10, perf_tier: 3 }),
    ) // 10..100 avec tier
    const noTier = row({ perf_score: 9999 }) // pas de tier → exclu
    const d = computeColumnDeciles([...withTier, noTier])
    // 9999 ignorée → p90 = 90 (et non 9999).
    expect(d.perf_score).toEqual({ p10: 10, p90: 90 })
  })

  it('team_mmr : non inversé (haut = meilleur), décile HAUT = best, décile BAS = worst', () => {
    expect(EXPLORER_INVERTED.team_mmr).toBe(false)
    const rows = [1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900].map((team_mmr) =>
      row({ team_mmr }),
    )
    const d = computeColumnDeciles(rows)
    expect(d.team_mmr).toEqual({ p10: 1000, p90: 1800 })
    expect(columnHighlightStyle('team_mmr', 1900, d).backgroundColor).toContain('--ac-outcome-win')
    expect(columnHighlightStyle('team_mmr', 1000, d).backgroundColor).toContain('--ac-outcome-loss')
    expect(columnHighlightStyle('team_mmr', 1400, d)).toEqual({})
  })
})

describe('decileCellState — bande de décile', () => {
  const d = { p10: 10, p90: 90 }

  it('non inversé : ≥ p90 → best, ≤ p10 → worst, entre → neutre', () => {
    expect(decileCellState(95, d, false)).toBe('best')
    expect(decileCellState(90, d, false)).toBe('best')
    expect(decileCellState(5, d, false)).toBe('worst')
    expect(decileCellState(10, d, false)).toBe('worst')
    expect(decileCellState(50, d, false)).toBe('neutral')
  })

  it('inversé : ≤ p10 → best, ≥ p90 → worst', () => {
    expect(decileCellState(5, d, true)).toBe('best')
    expect(decileCellState(95, d, true)).toBe('worst')
    expect(decileCellState(50, d, true)).toBe('neutral')
  })

  it('seuils nuls, p10 === p90, ou valeur nulle → neutre', () => {
    expect(decileCellState(5, { p10: null, p90: null }, false)).toBe('neutral')
    expect(decileCellState(5, { p10: 7, p90: 7 }, false)).toBe('neutral')
    expect(decileCellState(null, d, false)).toBe('neutral')
  })
})
