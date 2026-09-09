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

// Une case peut désormais être un tuple `[x, y, value, detail?]` (case mesurée)
// OU un objet `{ value: [...], itemStyle }` (case vide, décision D3 — cf.
// Heatmap2DChart.tsx `HeatCellDatum`). Ce helper lit le tuple quelle que soit la
// forme, pour que les tests n'aient pas à connaître laquelle une case donnée prend.
type RawCell =
  | [number, number, number | string, Record<string, unknown>?]
  | { value: [number, number, number | string, Record<string, unknown>?]; itemStyle?: Record<string, unknown> }

function tupleOf(d: RawCell): [number, number, number | string, Record<string, unknown>?] {
  return Array.isArray(d) ? d : d.value
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
      series: { data: RawCell[] }[]
    }
    const vides = opt.series[0].data.filter((d) => typeof tupleOf(d)[2] !== 'number')
    expect(vides.map((d) => `${tupleOf(d)[0]}-${tupleOf(d)[1]}`)).toEqual(['0-0', '1-1', '2-2'])
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

  it('ne dit RIEN non plus sur la VRAIE case vide produite (objet itemStyle, pas un tuple à la main)', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as unknown as {
      series: { data: RawCell[] }[]
      tooltip: { formatter: (p: { data: RawCell }) => string }
    }
    const caseVide = opt.series[0].data[0] // A-A, sur la diagonale
    expect(opt.tooltip.formatter({ data: caseVide })).toBe('')
  })
})

// ─── C1 — PADDING DES CASES ET ABSENCE VISIBLE (plan vague C formes, D2/D3) ────

describe('buildHeatmap2DOption — padding des cases (D2)', () => {
  const series: ChartSeries<ChartPointHeatmap>[] = [
    { key: 'm', datapoints: [{ x: 'A', y: 'A', value: 1 }, { x: 'B', y: 'A', value: 2 }] },
  ]

  it('deux cases voisines ne se touchent pas : borderWidth > 0 dans l’option', () => {
    const opt = buildHeatmap2DOption(series) as {
      series: { itemStyle: { borderWidth: number; borderColor: string; borderRadius: number } }[]
    }
    const { itemStyle } = opt.series[0]
    expect(itemStyle.borderWidth).toBeGreaterThan(0)
    expect(itemStyle.borderColor).toBeTruthy()
    expect(itemStyle.borderRadius).toBeGreaterThanOrEqual(0)
  })
})

describe('buildHeatmap2DOption — absence visible (D3)', () => {
  it('une case null reçoit un itemStyle à decal (hachure), les cases mesurées n’en ont pas', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as {
      series: { data: RawCell[] }[]
    }
    const [caseVide, caseMesuree] = opt.series[0].data
    expect(Array.isArray(caseVide)).toBe(false)
    if (Array.isArray(caseVide)) throw new Error('unreachable')
    expect(caseVide.itemStyle?.decal).toBeTruthy()
    // Case mesurée (hors diagonale) : tuple brut, aucun itemStyle propre.
    expect(Array.isArray(caseMesuree)).toBe(true)
  })

  it('une case null reçoit un tiret en étiquette, jamais une chaîne vide', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as {
      series: { data: RawCell[]; label: { formatter: (p: { data: RawCell }) => string } }[]
    }
    const caseVide = opt.series[0].data[0]
    expect(opt.series[0].label.formatter({ data: caseVide })).toBe('—')
  })

  it('une case mesurée garde son étiquette (le compte), inchangée', () => {
    const series: ChartSeries<ChartPointHeatmap>[] = [
      { key: 'm', datapoints: [{ x: 'A', y: 'A', value: 4, detail: { count: 4 } }] },
    ]
    const opt = buildHeatmap2DOption(series) as {
      series: { data: RawCell[]; label: { formatter: (p: { data: RawCell }) => string } }[]
    }
    expect(opt.series[0].label.formatter({ data: opt.series[0].data[0] })).toBe('4')
  })

  it('active aria.decal (requis par ECharts pour peindre les itemStyle.decal manuels)', () => {
    const opt = buildHeatmap2DOption(matriceCarree(['A', 'B'])) as { aria: { decal: { show: boolean } } }
    expect(opt.aria.decal.show).toBe(true)
  })
})

// ─── C1 — PLAFOND DE SATURATION OPTIONNEL (sans régression par défaut) ────────

describe('buildHeatmap2DOption — saturationCap (optionnel)', () => {
  const series: ChartSeries<ChartPointHeatmap>[] = [
    { key: 'm', datapoints: [{ x: 'A', y: 'y', value: 10 }, { x: 'B', y: 'y', value: 80 }] },
  ]

  it('sans la prop, le max reste la valeur réelle la plus haute (non-régression)', () => {
    const opt = buildHeatmap2DOption(series) as { visualMap: { max: number } }
    expect(opt.visualMap.max).toBe(80)
  })

  it('avec la prop, le max est le plafond même si la donnée le dépasse', () => {
    const opt = buildHeatmap2DOption(series, { saturationCap: 30 }) as { visualMap: { max: number } }
    expect(opt.visualMap.max).toBe(30)
  })

  it('valueRange garde priorité sur saturationCap (valueRange fixe déjà min ET max)', () => {
    const opt = buildHeatmap2DOption(series, {
      valueRange: [0, 100],
      saturationCap: 30,
    }) as { visualMap: { min: number; max: number } }
    expect(opt.visualMap.max).toBe(100)
  })
})
