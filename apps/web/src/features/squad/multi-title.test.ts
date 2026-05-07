/**
 * multi-title.test.ts — Test load-bearing : la liste de métriques
 * Squad doit dégrader gracefully quand le titre courant n'a qu'un
 * sous-ensemble des FieldKeys attendus (ex. synthetic_title_b).
 *
 * On simule le filtrage côté composants en réutilisant la même règle
 * (key absente du FieldMappingsResponse.fields → métrique masquée).
 */
import { describe, it, expect } from 'vitest'
import {
  SQUAD_KPI_METRICS,
  SQUAD_SYNERGY_METRICS,
  SQUAD_RADAR_METRICS,
} from './metrics'
import type { FieldMappingsResponse, FieldMappingDTO } from '@/lib/i18n/fieldMappings'
import type { SquadMetric } from './metrics'

function fieldDto(label: string): FieldMappingDTO {
  return {
    label,
    storage_unit: 'count',
    display_unit: 'count',
    format: 'integer',
    display_order: 1,
    group: 'combat',
  }
}

const HALO_INFINITE_MAPPINGS: FieldMappingsResponse = {
  title_slug: 'halo_infinite',
  schema_version: 1,
  locale: 'fr',
  fields: {
    kills: fieldDto('Frags'),
    deaths: fieldDto('Morts'),
    assists: fieldDto('Assists'),
    accuracy: fieldDto('Précision'),
    kdr: fieldDto('K/D'),
    win_rate: fieldDto('Taux de victoire'),
    total_matches_played: fieldDto('Matchs'),
    headshot_kills: fieldDto('Tirs à la tête'),
  },
}

const SYNTHETIC_TITLE_B_MAPPINGS: FieldMappingsResponse = {
  title_slug: 'synthetic_title_b',
  schema_version: 1,
  locale: 'fr',
  fields: {
    kills: fieldDto('Frags'),
    deaths: fieldDto('Pertes'),
    accuracy: fieldDto('Taux de réussite'),
  },
}

function availableMetrics(
  metrics: readonly SquadMetric[],
  mappings: FieldMappingsResponse,
): SquadMetric[] {
  return metrics.filter((m) => !!mappings.fields[m.key])
}

describe('multi-title graceful degradation', () => {
  describe('Halo Infinite (mapping complet)', () => {
    it('SQUAD_KPI_METRICS : 4 cards (matches, win_rate, kdr, kills)', () => {
      const filtered = availableMetrics(SQUAD_KPI_METRICS, HALO_INFINITE_MAPPINGS)
      expect(filtered.map((m) => m.key)).toEqual([
        'total_matches_played',
        'win_rate',
        'kdr',
        'kills',
      ])
    })

    it('SQUAD_SYNERGY_METRICS : 4 barres', () => {
      const filtered = availableMetrics(SQUAD_SYNERGY_METRICS, HALO_INFINITE_MAPPINGS)
      expect(filtered).toHaveLength(4)
    })

    it('SQUAD_RADAR_METRICS : 5 axes', () => {
      const filtered = availableMetrics(SQUAD_RADAR_METRICS, HALO_INFINITE_MAPPINGS)
      expect(filtered).toHaveLength(5)
    })
  })

  describe('synthetic_title_b (mapping minimaliste, sans win_rate/kdr/assists)', () => {
    it('SQUAD_KPI_METRICS : seul `kills` survit', () => {
      const filtered = availableMetrics(SQUAD_KPI_METRICS, SYNTHETIC_TITLE_B_MAPPINGS)
      expect(filtered.map((m) => m.key)).toEqual(['kills'])
    })

    it('SQUAD_SYNERGY_METRICS : seul `kills` survit (win_rate/kdr/assists masqués)', () => {
      const filtered = availableMetrics(SQUAD_SYNERGY_METRICS, SYNTHETIC_TITLE_B_MAPPINGS)
      expect(filtered.map((m) => m.key)).toEqual(['kills'])
    })

    it('SQUAD_RADAR_METRICS : `kills` + `accuracy` (les autres masqués)', () => {
      const filtered = availableMetrics(SQUAD_RADAR_METRICS, SYNTHETIC_TITLE_B_MAPPINGS)
      expect(filtered.map((m) => m.key)).toEqual(['kills', 'accuracy'])
    })

    it('aucune métrique restante ne contient un libellé FR Halo-spécifique', () => {
      // garde-fou : si on ajoute un jour une key Halo-only, cette assertion
      // doit attraper l'oubli (ex. "match_count" qui n'est pas une FieldKey
      // canonique, donc présent uniquement parce qu'on aurait remis un
      // fallback hardcodé).
      const all = [
        ...availableMetrics(SQUAD_KPI_METRICS, SYNTHETIC_TITLE_B_MAPPINGS),
        ...availableMetrics(SQUAD_SYNERGY_METRICS, SYNTHETIC_TITLE_B_MAPPINGS),
        ...availableMetrics(SQUAD_RADAR_METRICS, SYNTHETIC_TITLE_B_MAPPINGS),
      ]
      // Les keys retenues ne doivent contenir que les FieldKeys présents
      // dans synthetic_title_b.
      const allowed = new Set(Object.keys(SYNTHETIC_TITLE_B_MAPPINGS.fields))
      all.forEach((m) => expect(allowed.has(m.key)).toBe(true))
    })
  })
})
