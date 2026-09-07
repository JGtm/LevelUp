import { describe, it, expect, vi } from 'vitest'

import {
  buildHeatmap2DOption,
  type ChartPointHeatmap,
} from './Heatmap2DChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildHeatmap2DOption', () => {
  const series: ChartSeries<ChartPointHeatmap>[] = [
    {
      key: 'heatmap.test',
      datapoints: [
        { x: 'Aquarius', y: 'main', value: 75 },
        { x: 'Aquarius', y: 'f1', value: 60 },
        { x: 'Recharge', y: 'main', value: 80 },
      ],
    },
  ]

  it('extrait les axes uniques', () => {
    const opt = buildHeatmap2DOption(series) as {
      xAxis: { data: string[] }
      yAxis: { data: string[] }
    }
    expect(opt.xAxis.data).toEqual(['Aquarius', 'Recharge'])
    expect(opt.yAxis.data).toEqual(['main', 'f1'])
  })

  it('génère data au format [xIdx, yIdx, value, detail?]', () => {
    const opt = buildHeatmap2DOption(series) as {
      series: { data: unknown[][] }[]
    }
    expect(opt.series[0].data).toHaveLength(3)
    // Aquarius/main = [0, 0, 75, undefined] — 4e élément `detail` ajouté par
    // synthesis-kpi-grid (refonte chart : payload optionnel pour tooltip riche).
    expect(opt.series[0].data[0].slice(0, 3)).toEqual([0, 0, 75])
    // Recharge/main = [1, 0, 80, undefined]
    expect(opt.series[0].data[2].slice(0, 3)).toEqual([1, 0, 80])
  })

  it('palette sequential par défaut', () => {
    const opt = buildHeatmap2DOption(series) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-cold)',
      'var(heatmap-hot)',
    ])
  })

  it('palette divergent si paletteMode=divergent', () => {
    const opt = buildHeatmap2DOption(series, { paletteMode: 'divergent' }) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-divergent-low)',
      'var(divergent-neutral)',
      'var(heatmap-divergent-high)',
    ])
  })

  it('palette CVD : une heatmap séquentielle bascule sur la rampe fréquence (CVD-safe)', () => {
    const opt = buildHeatmap2DOption(series, { colorPalette: 'cividis' }) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-freq-low)',
      'var(heatmap-freq-high)',
    ])
  })

  it('valueRange override min/max', () => {
    const opt = buildHeatmap2DOption(series, { valueRange: [0, 100] }) as {
      visualMap: { min: number; max: number }
    }
    expect(opt.visualMap.min).toBe(0)
    expect(opt.visualMap.max).toBe(100)
  })

  it('series vide retourne option minimal', () => {
    expect(buildHeatmap2DOption([])).toEqual({ backgroundColor: 'transparent' })
  })
})

// ─── LES AXES DÉRIVÉS (correction W1, revue ronde 1 du 2026-09-06) ────────────
//
// Les catégories d'axe viennent de l'ORDRE D'APPARITION des points. Une matrice
// carrée dont l'appelant sauterait la diagonale sortirait donc avec un axe X décalé
// d'un cran par rapport à l'axe Y — la matrice se lirait de travers sans que rien ne
// le signale. Ces tests figent le contrat : émettez toutes les cases dans l'ordre,
// et les deux axes coïncident.

function matriceCarree(noms: string[]): ChartSeries<ChartPointHeatmap>[] {
  const datapoints: ChartPointHeatmap[] = []
  for (const y of noms) {
    for (const x of noms) {
      datapoints.push({ x, y, value: x === y ? null : 1 })
    }
  }
  return [{ key: 'matrice', datapoints }]
}

describe('buildHeatmap2DOption — axes d’une matrice carrée', () => {
  it('rend xs == ys == roster pour quatre joueurs', () => {
    const roster = ['A', 'B', 'C', 'D']
    const opt = buildHeatmap2DOption(matriceCarree(roster)) as {
      xAxis: { data: string[] }
      yAxis: { data: string[] }
    }
    expect(opt.xAxis.data).toEqual(roster)
    expect(opt.yAxis.data).toEqual(roster)
  })

  it('rend xs == ys sur un duo (le cas où l’inversion était totale)', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as {
      xAxis: { data: string[] }
      yAxis: { data: string[] }
    }
    expect(opt.xAxis.data).toEqual(['A', 'B'])
    expect(opt.yAxis.data).toEqual(opt.xAxis.data)
  })

  it('place les cases vides SUR la diagonale', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B', 'C'])) as {
      series: { data: [number, number, number | string][] }[]
    }
    const vides = opt.series[0].data.filter(([, , v]) => typeof v !== 'number')
    expect(vides.map(([x, y]) => `${x}-${y}`)).toEqual(['0-0', '1-1', '2-2'])
  })

  it('exclut les cases vides de l’échelle (une case absente n’est pas un zéro)', () => {
    const series: ChartSeries<ChartPointHeatmap>[] = [
      {
        key: 'm',
        datapoints: [
          { x: 'A', y: 'A', value: null },
          { x: 'B', y: 'A', value: 4 },
          { x: 'A', y: 'B', value: 7 },
          { x: 'B', y: 'B', value: null },
        ],
      },
    ]
    const opt = buildHeatmap2DOption(series) as { visualMap: { min: number; max: number } }
    expect(opt.visualMap.min).toBe(4)
    expect(opt.visualMap.max).toBe(7)
  })
})

// ─── LE TOOLTIP AU CHOIX DE L'APPELANT (correction W5) ────────────────────────

describe('buildHeatmap2DOption — formatTooltip', () => {
  const cellule: ChartSeries<ChartPointHeatmap>[] = [
    { key: 'm', datapoints: [{ x: 'Bob', y: 'Alice', value: 4, detail: { count: 4 } }] },
  ]
  type Fmt = { tooltip: { formatter: (p: { data: unknown[] }) => string } }

  it('emploie le formateur de l’appelant quand il en passe un', () => {
    const opt = buildHeatmap2DOption(cellule, {
      formatTooltip: (p) => `${p.y} a vengé ${p.x} ${p.value} fois`,
    }) as unknown as Fmt
    expect(opt.tooltip.formatter({ data: [0, 0, 4, { count: 4 }] })).toBe(
      'Alice a vengé Bob 4 fois',
    )
  })

  it('retombe sur le libellé historique quand il n’en passe pas', () => {
    // INVERSION JOUÉE : sans la branche `if (formatTooltip)`, la matrice d'échange
    // annoncerait « Win Rate: 400.0 % » pour 4 vengeances.
    const opt = buildHeatmap2DOption(cellule) as unknown as Fmt
    const rendu = opt.tooltip.formatter({ data: [0, 0, 4, { count: 4 }] })
    expect(rendu).toContain('Win Rate')
    expect(rendu).toContain('400.0%')
  })

  it('ne dit RIEN sur une case vide, pas même « 0 »', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as unknown as Fmt
    expect(opt.tooltip.formatter({ data: [0, 0, '-', undefined] })).toBe('')
  })
})
