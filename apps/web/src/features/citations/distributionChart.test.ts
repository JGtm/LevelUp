/**
 * Tests — buildDistributionSeries (Phase 2 P2.D).
 */
import { describe, it, expect } from 'vitest'
import { buildDistributionSeries, MAX_MEDALS_IN_CHART } from './distributionChart'
import type { MedalSummary } from '@/lib/api/types'

const labels = { filteredLabel: 'Filtered', totalLabel: 'Total' } as const

let nextMedalId = 1
function makeMedal(name: string, filtered: number, total: number): MedalSummary {
  return {
    medal_name_id: nextMedalId++,
    name,
    count_filtered: filtered,
    count_total: total,
    description: null,
  }
}

describe('buildDistributionSeries', () => {
  it('retourne 1 série avec 1 datapoint par médaille', () => {
    const medals = [makeMedal('Killjoy', 5, 10), makeMedal('Sniper', 3, 8)]
    const series = buildDistributionSeries(medals, labels)
    expect(series).toHaveLength(1)
    expect(series[0].key).toBe('citations.distribution.medals')
    expect(series[0].datapoints).toHaveLength(2)
  })

  it('trie les médailles par count_total décroissant', () => {
    const medals = [
      makeMedal('Faible', 1, 2),
      makeMedal('Forte', 5, 100),
      makeMedal('Moyenne', 3, 50),
    ]
    const dps = buildDistributionSeries(medals, labels)[0].datapoints
    expect(dps.map((d) => d.category)).toEqual(['Forte', 'Moyenne', 'Faible'])
  })

  it(`limite la série à ${MAX_MEDALS_IN_CHART} médailles`, () => {
    const medals = Array.from({ length: 30 }, (_, i) =>
      makeMedal(`Medal ${i}`, i, 100 - i),
    )
    const series = buildDistributionSeries(medals, labels)
    expect(series[0].datapoints).toHaveLength(MAX_MEDALS_IN_CHART)
  })

  it('mappe filtered/total sur les composants nommés via les labels', () => {
    const series = buildDistributionSeries([makeMedal('M1', 7, 12)], labels)
    expect(series[0].datapoints[0].components).toEqual({
      Filtered: 7,
      Total: 12,
    })
  })

  it('retourne une série vide quand aucune médaille', () => {
    const series = buildDistributionSeries([], labels)
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toHaveLength(0)
  })

  it('respecte les labels custom (i18n)', () => {
    const enLabels = { filteredLabel: 'Filtré', totalLabel: 'Total' }
    const series = buildDistributionSeries([makeMedal('M', 1, 2)], enLabels)
    const components = series[0].datapoints[0].components
    expect(Object.keys(components)).toEqual(['Filtré', 'Total'])
  })
})
