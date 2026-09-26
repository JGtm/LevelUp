/**
 * squadFdaGapChart.test.ts — « Écart cumulé au FDA attendu » (Lot C, D3/D5).
 *
 * - `cumulativeFdaGapSeries` (pur) : cumul par match_order + trous D5 (report) +
 *   robustesse au désordre / non-fini.
 * - `buildFdaGapCumulativeOption` (pur) : 1 line/joueur, couleurs, markLine 0, pas d'aire ;
 *   lot L1 (PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : valeur de fin au bout de
 *   chaque courbe (signée, une décimale, couleur du joueur, chevauchements écartés),
 *   légende en bas et centrée.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildFdaGapCumulativeOption,
  cumulativeFdaGapSeries,
} from './squadFdaGapChart'
import { xAxisLabels } from './squadPerformanceLineCharts'

function pt(
  order: number,
  kda: number | undefined,
  kdaExpected: number | undefined,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    kda,
    kda_expected: kdaExpected,
  }
}

interface LineSeries {
  name: string
  type: string
  data: Array<number | null>
  lineStyle: { color: string }
  markLine?: { data: Array<{ yAxis: number }> }
  areaStyle?: unknown
  endLabel?: { show: boolean; color: string; formatter: (p: { value?: unknown }) => string }
  labelLayout?: { moveOverlap?: string }
}

describe('cumulativeFdaGapSeries', () => {
  it('cumul du différentiel par match_order croissant', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, 1.2), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, 0.1, 1.1])
  })

  it('report D5 : un match sans attendu saute le cumul (reporte la dernière valeur)', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, undefined), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, 0.5, 1.5])
  })

  it('report D5 côté réel manquant également', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, undefined, 1.0)], 2)
    expect(data).toEqual([0.5, 0.5])
  })

  it('valeur non-finie (Infinity) traitée comme absente (D5)', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, Infinity)], 2)
    expect(data).toEqual([0.5, 0.5])
  })

  it('match_order désordonné : trie avant de cumuler', () => {
    const data = cumulativeFdaGapSeries([pt(2, 2.0, 1.0), pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)], 3)
    expect(data).toEqual([0.5, 0.1, 1.1])
  })

  it('trou d\'intersection (aucune ligne) reste null', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, null, 1.5])
  })
})

describe('buildFdaGapCumulativeOption', () => {
  const COLORS = { Me: '#aaa', F1: '#bbb' }
  const ORDER = ['Me', 'F1']

  it('vide → option de fond minimale (pas de série)', () => {
    const opt = buildFdaGapCumulativeOption({}, { colorByPlayer: {} })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 série line par joueur : data = cumul, couleur du joueur appliquée', () => {
    const rows = {
      Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)],
      F1: [pt(0, 2.0, 1.0), pt(1, 1.0, 1.5)],
    }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ORDER })
    const series = opt.series as unknown as LineSeries[]
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'line')).toBe(true)
    expect(series[0].data).toEqual([0.5, 0.1])
    expect(series[1].data).toEqual([1, 0.5])
    expect(series[0].lineStyle.color).toBe('#aaa')
    expect(series[1].lineStyle.color).toBe('#bbb')
  })

  it('markLine 0 sur le premier joueur uniquement, aucune aire (multi-séries)', () => {
    const rows = { Me: [pt(0, 1.5, 1.0)], F1: [pt(0, 2.0, 1.0)] }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ORDER })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].markLine?.data[0].yAxis).toBe(0)
    expect(series[1].markLine).toBeUndefined()
    expect(series[0].areaStyle).toBeUndefined()
    expect(series[1].areaStyle).toBeUndefined()
  })

  it('joueur masqué (hiddenPlayers) → série vidée (null)', () => {
    const rows = {
      Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)],
      F1: [pt(0, 2.0, 1.0), pt(1, 1.0, 1.5)],
    }
    const opt = buildFdaGapCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      hiddenPlayers: new Set(['F1']),
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].data).toEqual([0.5, 0.1])
    expect(series[1].data).toEqual([null, null])
  })

  it('valeur de fin au bout de chaque courbe : signée, une décimale, couleur du joueur', () => {
    const rows = {
      Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)],
      F1: [pt(0, 0.5, 1.0), pt(1, 1.0, 1.5)],
    }
    const opt = buildFdaGapCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      intlLocale: 'fr-FR',
    })
    const series = opt.series as unknown as LineSeries[]
    for (const [i, s] of series.entries()) {
      expect(s.endLabel?.show).toBe(true)
      expect(s.endLabel?.color).toBe(i === 0 ? '#aaa' : '#bbb')
      // Étiquettes voisines écartées verticalement plutôt que superposées.
      expect(s.labelLayout?.moveOverlap).toBe('shiftY')
    }
    const fmt = series[0].endLabel!.formatter
    expect(fmt({ value: 3.03 })).toBe('+3,0')
    expect(fmt({ value: 15.41 })).toBe('+15,4')
    expect(fmt({ value: -1.28 })).toMatch(/^[-−]1,3$/)
    // Un zéro arrondi ne porte pas de signe (jamais « -0,0 »).
    expect(fmt({ value: -0.04 })).toBe('0,0')
    expect(fmt({})).toBe('')
  })

  it('locale EN : séparateur décimal point', () => {
    const rows = { Me: [pt(0, 1.5, 1.0)] }
    const opt = buildFdaGapCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ['Me'],
      intlLocale: 'en-US',
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].endLabel!.formatter({ value: 0.5 })).toBe('+0.5')
  })

  it('légende en bas et centrée ; marge droite réservée aux valeurs de fin', () => {
    const rows = { Me: [pt(0, 1.5, 1.0)], F1: [pt(0, 2.0, 1.0)] }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ORDER })
    const legend = opt.legend as { bottom: number; left: string; data: string[] }
    expect(legend.bottom).toBe(0)
    expect(legend.left).toBe('center')
    expect(legend.data).toEqual(['Me', 'F1'])
    const grid = opt.grid as { right: number }
    expect(grid.right).toBeGreaterThanOrEqual(40)
  })

  it('même abscisse que la balance des dégâts : #1..#n', () => {
    const rows = { Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2), pt(2, 1.0, 1.0)] }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ['Me'] })
    const x = opt.xAxis as { data: string[] }
    expect(x.data).toEqual(xAxisLabels(3))
  })
})
