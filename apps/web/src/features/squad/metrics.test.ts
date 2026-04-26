/**
 * metrics.test.ts — Validation des listes de FieldKeys par surface.
 *
 * Vérifie que :
 *  - chaque entrée pointe vers une key non-vide
 *  - aucun doublon dans une même surface
 *  - les extracteurs ne crashent pas sur un TeammateKPIs vide / partiel
 *  - HS et PK sont distincts dans SQUAD_HSPK_METRICS
 */
import { describe, it, expect } from 'vitest'
import {
  SQUAD_KPI_METRICS,
  SQUAD_SYNERGY_METRICS,
  SQUAD_RADAR_METRICS,
  SQUAD_HSPK_METRICS,
} from './metrics'
import type { TeammateKPIs } from '@/lib/api/types'

const EMPTY_KPIS: TeammateKPIs = {
  match_count: 0,
  wins: 0,
  kd_ratio: null,
  win_rate: 0,
  accuracy: null,
  kills_per_game: null,
  assists_per_game: null,
  headshot_kills_per_game: null,
  perfect_kills_per_game: null,
}

const FULL_KPIS: TeammateKPIs = {
  match_count: 10,
  wins: 6,
  kd_ratio: 1.5,
  win_rate: 0.6,
  accuracy: 0.45,
  kills_per_game: 12,
  assists_per_game: 5,
  headshot_kills_per_game: 4,
  perfect_kills_per_game: 1,
}

function expectMetricsValid(
  name: string,
  metrics: ReadonlyArray<{ key: string; extract: (k: TeammateKPIs) => number | null }>,
) {
  it(`${name} : aucune key vide`, () => {
    metrics.forEach((m) => expect(m.key.length).toBeGreaterThan(0))
  })
  it(`${name} : aucun doublon`, () => {
    const keys = metrics.map((m) => m.key)
    expect(new Set(keys).size).toBe(keys.length)
  })
  it(`${name} : extracteur ne crashe pas sur EMPTY_KPIS`, () => {
    metrics.forEach((m) => {
      expect(() => m.extract(EMPTY_KPIS)).not.toThrow()
    })
  })
  it(`${name} : extracteur retourne un nombre fini ou null sur FULL_KPIS`, () => {
    metrics.forEach((m) => {
      const v = m.extract(FULL_KPIS)
      if (v !== null) expect(Number.isFinite(v)).toBe(true)
    })
  })
}

describe('SQUAD_KPI_METRICS', () => {
  expectMetricsValid('SQUAD_KPI_METRICS', SQUAD_KPI_METRICS)
})

describe('SQUAD_SYNERGY_METRICS', () => {
  expectMetricsValid('SQUAD_SYNERGY_METRICS', SQUAD_SYNERGY_METRICS)
})

describe('SQUAD_RADAR_METRICS', () => {
  expectMetricsValid('SQUAD_RADAR_METRICS', SQUAD_RADAR_METRICS)
  it('toutes les valeurs normalisées sont dans [0, 100] sur FULL_KPIS', () => {
    SQUAD_RADAR_METRICS.forEach((m) => {
      const v = m.extract(FULL_KPIS)
      if (v !== null) {
        expect(v).toBeGreaterThanOrEqual(0)
        expect(v).toBeLessThanOrEqual(100)
      }
    })
  })
})

describe('SQUAD_HSPK_METRICS', () => {
  it('hs et pk sont définis avec des keys distinctes', () => {
    expect(SQUAD_HSPK_METRICS.hs.key).not.toBe(SQUAD_HSPK_METRICS.pk.key)
  })

  it('hs.extract retourne 0 quand headshot_kills_per_game est null', () => {
    expect(SQUAD_HSPK_METRICS.hs.extract(EMPTY_KPIS)).toBe(0)
  })

  it('pk.extract retourne 0 quand perfect_kills_per_game est null', () => {
    expect(SQUAD_HSPK_METRICS.pk.extract(EMPTY_KPIS)).toBe(0)
  })

  it('hs.extract et pk.extract reflètent les valeurs FULL_KPIS', () => {
    expect(SQUAD_HSPK_METRICS.hs.extract(FULL_KPIS)).toBe(4)
    expect(SQUAD_HSPK_METRICS.pk.extract(FULL_KPIS)).toBe(1)
  })
})
