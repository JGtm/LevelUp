import { describe, expect, it } from 'vitest'

import { buildOption, type HeatmapBucketCell } from './RelationsMomentsHeatmap'

// Garde-rail : la heatmap « Rythme des rencontres » ne montre JAMAIS plus de 12
// relations sur l'axe Y (les 12 les plus actives, par total décroissant).
describe('RelationsMomentsHeatmap buildOption', () => {
  it('tronque l’axe Y à 12 relations quand il y en a plus', () => {
    // 20 relations, 1 cellule chacune (bucket 0), total = i+1 → tri déterministe.
    const cells: HeatmapBucketCell[] = Array.from({ length: 20 }, (_, i) => ({
      xuid: `x${i}`,
      gamertag: `Player${i}`,
      bucket: 0,
      count: i + 1,
    }))

    const opt = buildOption(cells, ['00h'], 'Matchs', (n) => `${n}`) as {
      yAxis: { data: string[] }
    }

    // Exactement 12 lignes, et ce sont les 12 plus actives (counts 20..9).
    expect(opt.yAxis.data).toHaveLength(12)
    expect(opt.yAxis.data[0]).toBe('Player19') // total le plus élevé (20)
    expect(opt.yAxis.data).not.toContain('Player0') // total le plus faible, écarté
  })

  it('n’ajoute pas de lignes vides quand il y a moins de 12 relations', () => {
    const cells: HeatmapBucketCell[] = Array.from({ length: 5 }, (_, i) => ({
      xuid: `x${i}`,
      gamertag: `P${i}`,
      bucket: 0,
      count: 1,
    }))
    const opt = buildOption(cells, ['00h'], 'Matchs', (n) => `${n}`) as {
      yAxis: { data: string[] }
    }
    expect(opt.yAxis.data).toHaveLength(5)
  })
})
