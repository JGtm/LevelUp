/**
 * Tests du builder de la matrice : groupes et ordre, filtre `in_matrix`, ordre d'émission des
 * cases, `null` / 0 / borne de la couleur, `detail.count`, colonnes, infobulle.
 */
import { describe, expect, it } from 'vitest'

import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'

import { MONTHS, horizonCells, indicator, monthCells } from './tendances.fixture'
import {
  buildMatrixGrids,
  formatMatrixTooltip,
  matrixGridHeight,
  matrixLeftMargin,
  monthShortLabel,
} from './tendancesMatrix.logic'

const labelOf = (key: string, variant?: string) => (variant ? `${key}/${variant}` : key)

function points(grid: { series: { datapoints: ChartPointHeatmap[] }[] }) {
  return grid.series[0].datapoints
}

describe('buildMatrixGrids — groupes', () => {
  it('une grille par groupe présent, dans l’ordre level → activity, hors indicateurs hors matrice', () => {
    const grids = buildMatrixGrids(
      [
        indicator({ key: 'match_count', group: 'activity' }),
        indicator({ key: 'kda', group: 'combat' }),
        indicator({ key: 'enemy_mmr', group: 'level' }),
        indicator({ key: 'kills_per_match', group: 'combat', in_matrix: false }),
        indicator({ key: 'win_rate', group: 'results' }),
      ],
      MONTHS,
      'fr',
      labelOf,
    )
    expect(grids.map((g) => g.group)).toEqual(['level', 'results', 'combat', 'activity'])
    expect(grids.find((g) => g.group === 'combat')?.rows).toBe(1)
  })

  it('aucun indicateur en matrice : aucune grille', () => {
    expect(buildMatrixGrids([indicator({ in_matrix: false })], MONTHS, 'fr', labelOf)).toEqual([])
    expect(buildMatrixGrids(null, null, 'fr', labelOf)).toEqual([])
  })

  it('un groupe inconnu n’est pas une grille', () => {
    expect(buildMatrixGrids([indicator({ group: 'inconnu' })], MONTHS, 'fr', labelOf)).toEqual([])
  })

  it('les groupes de la vue Escouade (squad, members) suivent les autres, dans cet ordre', () => {
    const grids = buildMatrixGrids(
      [
        indicator({ key: 'kda', group: 'members', variant: 'Alice' }),
        indicator({ key: 'win_rate', group: 'squad' }),
        indicator({ key: 'activity_x', group: 'activity' }),
      ],
      MONTHS,
      'fr',
      labelOf,
    )
    expect(grids.map((g) => g.group)).toEqual(['activity', 'squad', 'members'])
  })
})

describe('buildMatrixGrids — cases', () => {
  const grids = buildMatrixGrids(
    [indicator({ key: 'a' }), indicator({ key: 'b', variant: 'v' })],
    MONTHS,
    'fr',
    labelOf,
  )
  const pts = points(grids[0])

  it('émet toutes les cases : 16 colonnes × 2 lignes, colonnes d’abord puis lignes', () => {
    expect(pts).toHaveLength(32)
    expect(pts[0].y).toBe('a')
    expect(pts[1].y).toBe('b/v')
    expect(pts[0].x).toBe(pts[1].x)
    expect(pts[2].x).not.toBe(pts[0].x)
  })

  it('colonnes : 12 mois courts dans la locale, puis 365 j, 90 j, 30 j, 7 j', () => {
    const colonnes = [...new Set(pts.map((p) => p.x))]
    expect(colonnes).toHaveLength(16)
    expect(colonnes.slice(0, 12)).toEqual(MONTHS.map((m) => monthShortLabel(m, 'fr')))
    expect(colonnes.slice(12)).toEqual(['365 j', '90 j', '30 j', '7 j'])
  })

  it('colonnes d’horizon en anglais : « 365 d »…', () => {
    const en = buildMatrixGrids([indicator()], MONTHS, 'en', labelOf)
    const colonnes = [...new Set(points(en[0]).map((p) => p.x))]
    expect(colonnes.slice(12)).toEqual(['365 d', '90 d', '30 d', '7 d'])
    expect(colonnes[0]).toBe(monthShortLabel('2025-10', 'en'))
  })

  it('nom court du mois : fuseau neutre (pas de glissement au 1er du mois)', () => {
    expect(monthShortLabel('2026-01', 'en')).toBe('Jan')
    expect(monthShortLabel('2026-12', 'en')).toBe('Dec')
  })
})

describe('buildMatrixGrids — valeur de la couleur et nombre écrit', () => {
  it('value = z borné ; nombre (detail.count) = valeur formatée', () => {
    const grids = buildMatrixGrids(
      [
        indicator({
          months: monthCells(MONTHS.map(() => 0.5432), 9),
          horizons: horizonCells({ 90: { value: 0.6, z: -9 } }),
        }),
      ],
      MONTHS,
      'fr',
      labelOf,
    )
    const pts = points(grids[0])
    expect(pts[0].value).toBe(2.5)
    expect(pts[0].detail?.count).toBe('54,3\u00a0%')
    const h90 = pts.find((p) => p.x === '90 j')
    expect(h90?.value).toBe(-2.5)
    expect(h90?.detail?.count).toBe('60,0\u00a0%')
  })

  it('horizon sans comparaison (z absent) : couleur 0, nombre quand même écrit', () => {
    const grids = buildMatrixGrids(
      [
        indicator({
          horizons: horizonCells({ 7: { z: undefined, prev_value: undefined, prev_matches: 3 } }),
        }),
      ],
      MONTHS,
      'fr',
      labelOf,
    )
    const h7 = points(grids[0]).find((p) => p.x === '7 j')
    expect(h7?.value).toBe(0)
    expect(h7?.detail?.count).toBe('50,0\u00a0%')
  })

  it('cellule sans valeur (mois vide, horizon absent) : value null, sans detail', () => {
    const grids = buildMatrixGrids(
      [
        indicator({
          months: monthCells([undefined, 0.4]),
          horizons: horizonCells().filter((h) => h.days !== 30),
        }),
      ],
      MONTHS,
      'fr',
      labelOf,
    )
    const pts = points(grids[0])
    expect(pts[0].value).toBeNull()
    expect(pts[0].detail).toBeUndefined()
    expect(pts[1].value).not.toBeNull()
    expect(pts.find((p) => p.x === '30 j')?.value).toBeNull()
  })
})

describe('mise en page', () => {
  it('hauteur proportionnelle au nombre de lignes', () => {
    expect(matrixGridHeight(4) - matrixGridHeight(2)).toBe(2 * (matrixGridHeight(2) - matrixGridHeight(1)))
    expect(matrixGridHeight(3)).toBeGreaterThan(matrixGridHeight(2))
  })

  it('marge gauche COMMUNE : fixée par la plus longue étiquette, bornée', () => {
    const court = buildMatrixGrids([indicator({ key: 'ab' })], MONTHS, 'fr', labelOf)
    const long = buildMatrixGrids(
      [indicator({ key: 'x'.repeat(200) })],
      MONTHS,
      'fr',
      labelOf,
    )
    expect(matrixLeftMargin(court)).toBe(120)
    expect(matrixLeftMargin(long)).toBe(260)
    expect(matrixLeftMargin([...court, ...long])).toBe(260)
  })
})

describe('formatMatrixTooltip', () => {
  function cell(overrides: Parameters<typeof horizonCells>[0], x = '90 j') {
    const grids = buildMatrixGrids(
      [indicator({ key: 'win_rate', horizons: horizonCells(overrides) })],
      MONTHS,
      'fr',
      labelOf,
    )
    return points(grids[0]).find((p) => p.x === x) as ChartPointHeatmap
  }

  it('horizon comparé : valeur, période d’avant, variation signée', () => {
    const html = formatMatrixTooltip(cell({ 90: { value: 0.55, prev_value: 0.5 } }), 'fr')
    expect(html).toContain('win_rate · 90 derniers jours')
    expect(html).toContain('Valeur : 55,0\u00a0% (40 matchs)')
    expect(html).toContain('Période d&#39;avant : 50,0\u00a0% (38 matchs)')
    expect(html).toContain('Variation : +5,0\u00a0pts')
  })

  it('horizon non comparé : dit combien de matchs il y avait avant et combien il en faut', () => {
    const html = formatMatrixTooltip(
      cell({ 7: { z: undefined, prev_value: undefined, prev_matches: 4 } }, '7 j'),
      'fr',
    )
    expect(html).toContain('Pas de comparaison (4 matchs sur la période d&#39;avant, il en faut 10)')
    expect(html).not.toContain('Variation')
  })

  it('horizon non comparé faute de matchs sur la période courante : le dit', () => {
    const html = formatMatrixTooltip(
      cell({ 7: { z: undefined, prev_value: undefined, matches: 3, prev_matches: 30 } }, '7 j'),
      'fr',
    )
    expect(html).toContain('Pas de comparaison (3 matchs sur la période, il en faut 10)')
  })

  it('mois : libellé, mois en toutes lettres, valeur et matchs', () => {
    const grids = buildMatrixGrids([indicator()], MONTHS, 'en', labelOf)
    const html = formatMatrixTooltip(points(grids[0])[11], 'en')
    expect(html).toContain('win_rate · September 2026')
    expect(html).toContain('Value: 50.0% (20 matches)')
    expect(html).not.toContain('Previous period')
  })

  it('échappe le libellé de la ligne', () => {
    const grids = buildMatrixGrids([indicator()], MONTHS, 'fr', () => '<b>x</b>')
    const html = formatMatrixTooltip(points(grids[0])[0], 'fr')
    expect(html).toContain('&lt;b&gt;x&lt;/b&gt;')
    expect(html).not.toContain('<b>x</b>')
  })

  it('case sans detail : chaîne vide', () => {
    expect(formatMatrixTooltip({ x: 'a', y: 'b', value: null }, 'fr')).toBe('')
  })
})
