import { describe, it, expect } from 'vitest'
import { canonicalMetricKey, humanizeMetricKey, PRESTIGE_METRIC_OPTIONS } from './metricLabel'

// Ces tests couvrent ce qui RESTE dans metricLabel.ts après la suppression des
// dictionnaires FR/EN (v7.3 lot 2, item 3.3) : la résolution d'alias vers les
// FieldKey canoniques et l'humanisation de repli. Les libellés eux-mêmes ne se
// testent plus ici — ils vivent dans config/titles/{slug}/mappings/fields.toml
// et sont couverts côté Go (games/mappings) ainsi que par le rendu via
// useMetricLabel.

describe('canonicalMetricKey', () => {
  it('résout les jetons Prestige Field* vers la FieldKey canonique', () => {
    expect(canonicalMetricKey('FieldKDA')).toBe('kda')
    expect(canonicalMetricKey('FieldWinRate')).toBe('win_rate')
    expect(canonicalMetricKey('FieldAccuracy')).toBe('accuracy')
    expect(canonicalMetricKey('FieldHeadshotKills')).toBe('headshot_kills')
  })

  it('résout les clés courtes du détecteur de records', () => {
    expect(canonicalMetricKey('kpm')).toBe('kills_per_min')
    expect(canonicalMetricKey('pspm')).toBe('personal_score_per_min')
  })

  it('résout les clés historiques des catalogues de jalons', () => {
    expect(canonicalMetricKey('headshots')).toBe('headshot_kills')
    expect(canonicalMetricKey('matches_played')).toBe('match_count')
  })

  it('laisse intacte une clé déjà canonique', () => {
    expect(canonicalMetricKey('kda')).toBe('kda')
    expect(canonicalMetricKey('combat_precision_matches')).toBe('combat_precision_matches')
    expect(canonicalMetricKey('wins')).toBe('wins')
  })

  it('laisse intacte une clé inconnue (pas de mapping inventé)', () => {
    expect(canonicalMetricKey('weird_unknown_metric')).toBe('weird_unknown_metric')
  })

  it('toutes les options Prestige ont une cible canonique distincte du jeton', () => {
    for (const opt of PRESTIGE_METRIC_OPTIONS) {
      const canonical = canonicalMetricKey(opt)
      expect(canonical).not.toBe(opt)
      expect(canonical).not.toMatch(/\bField[A-Z]/)
    }
  })
})

describe('humanizeMetricKey', () => {
  it('humanise un jeton Field* (jamais la clé brute)', () => {
    const raw = 'FieldMysteryStat'
    const label = humanizeMetricKey(raw)
    expect(label).not.toBe(raw)
    expect(label).not.toMatch(/\bField[A-Z]/)
    expect(label).toBe('Mystery stat')
  })

  it('humanise une clé snake_case inconnue', () => {
    expect(humanizeMetricKey('weird_unknown_metric')).toBe('Weird unknown metric')
  })

  it('retombe sur la clé si l’humanisation ne produit rien', () => {
    expect(humanizeMetricKey('Field')).toBe('Field')
  })
})
