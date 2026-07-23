import { describe, it, expect } from 'vitest'
import { metricLabel, PRESTIGE_METRIC_OPTIONS } from './metricLabel'

describe('metricLabel', () => {
  it('résout les clés Prestige Field* vers le libellé canonique', () => {
    expect(metricLabel('FieldKDA', 'fr')).toBe('KDA')
    expect(metricLabel('FieldWinRate', 'fr')).toBe('Taux de victoire')
    expect(metricLabel('FieldWinRate', 'en')).toBe('Win rate')
    expect(metricLabel('FieldAccuracy', 'fr')).toBe('Précision')
  })

  it('résout les clés canoniques snake_case', () => {
    expect(metricLabel('matches_played', 'fr')).toBe('Matchs joués')
    expect(metricLabel('kda', 'en')).toBe('KDA')
    expect(metricLabel('combat_precision_matches', 'fr')).toBe('Matchs de précision')
  })

  it('humanise une clé inconnue (jamais la clé brute)', () => {
    const raw = 'FieldMysteryStat'
    const label = metricLabel(raw, 'fr')
    expect(label).not.toBe(raw)
    expect(label).not.toMatch(/\bField[A-Z]/)
    expect(label).toBe('Mystery stat')
  })

  it('humanise une clé snake_case inconnue', () => {
    const raw = 'weird_unknown_metric'
    const label = metricLabel(raw, 'en')
    expect(label).not.toBe(raw)
    expect(label).toBe('Weird unknown metric')
  })

  it('toutes les options Prestige ont un libellé non technique', () => {
    for (const opt of PRESTIGE_METRIC_OPTIONS) {
      for (const locale of ['fr', 'en'] as const) {
        const label = metricLabel(opt, locale)
        expect(label).not.toBe(opt)
        expect(label).not.toMatch(/\bField[A-Z]/)
      }
    }
  })
})
