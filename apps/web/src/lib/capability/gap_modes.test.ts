import { describe, expect, it } from 'vitest'

import { SECTION_GAP_MODES, resolveGapMode } from './gap_modes'

describe('gap_modes', () => {
  it('SECTION_GAP_MODES contient les sections pilotes Squad + MatchView', () => {
    expect(SECTION_GAP_MODES['squad.impact_heatmap']).toBeDefined()
    expect(SECTION_GAP_MODES['match_view.dominance_badge']).toBeDefined()
    expect(SECTION_GAP_MODES['match_view.radar_participation']).toBe('cta')
  })

  it('resolveGapMode retourne le mode défini pour une section connue', () => {
    expect(resolveGapMode('match_view.radar_participation')).toBe('cta')
    expect(resolveGapMode('squad.medals_gallery')).toBe('hide')
    expect(resolveGapMode('squad.impact_heatmap')).toBe('placeholder')
  })

  it('resolveGapMode retourne placeholder par défaut pour clé inconnue', () => {
    expect(resolveGapMode('section.inexistante')).toBe('placeholder')
    expect(resolveGapMode('')).toBe('placeholder')
  })

  it('toutes les valeurs sont des modes valides', () => {
    const validModes = new Set(['hide', 'placeholder', 'cta'])
    for (const [key, mode] of Object.entries(SECTION_GAP_MODES)) {
      expect(validModes.has(mode), `clé "${key}" : mode "${mode}" invalide`).toBe(true)
    }
  })
})
