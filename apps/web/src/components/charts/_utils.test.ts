import { describe, it, expect, vi } from 'vitest'

import {
  buildDualGridOption,
  DUAL_LAYOUT_THRESHOLD,
  dualPanelHeight,
  formatDateShort,
  formatNumber,
  outcomeColor,
  seriesColor,
  tickInterval,
} from './_utils'
import type { EChartsCoreOption } from 'echarts/core'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('_utils', () => {
  describe('outcomeColor', () => {
    it('mappe win → outcome-win', () => {
      expect(outcomeColor('win')).toBe('var(outcome-win)')
    })
    it('mappe loss → outcome-loss', () => {
      expect(outcomeColor('loss')).toBe('var(outcome-loss)')
    })
    it('mappe tie → outcome-draw', () => {
      expect(outcomeColor('tie')).toBe('var(outcome-draw)')
    })
    it('mappe dnf → outcome-dnf', () => {
      expect(outcomeColor('dnf')).toBe('var(outcome-dnf)')
    })
    it('fallback inconnu → chart-series-1', () => {
      expect(outcomeColor('unknown')).toBe('var(chart-series-1)')
      expect(outcomeColor(undefined)).toBe('var(chart-series-1)')
    })
  })

  describe('seriesColor', () => {
    it('cycle modulo 8', () => {
      expect(seriesColor(0)).toBe('var(chart-series-1)')
      expect(seriesColor(7)).toBe('var(chart-series-8)')
      expect(seriesColor(8)).toBe('var(chart-series-1)') // wrap
    })
  })

  describe('tickInterval', () => {
    it('petits volumes → 1', () => {
      expect(tickInterval(5)).toBe(1)
      expect(tickInterval(10)).toBe(1)
    })
    it('volumes moyens → 2', () => {
      expect(tickInterval(20)).toBe(2)
      expect(tickInterval(30)).toBe(2)
    })
    it('grand volume → 5', () => {
      expect(tickInterval(50)).toBe(5)
    })
    it('très grand → 10+', () => {
      expect(tickInterval(120)).toBe(10)
      expect(tickInterval(240)).toBeGreaterThan(10)
    })
  })

  describe('formatDateShort', () => {
    it('formate Date FR DD/MM', () => {
      expect(formatDateShort(new Date('2026-04-27'))).toBe('27/04')
    })
    it('accepte une string ISO', () => {
      expect(formatDateShort('2026-12-01')).toBe('01/12')
    })
  })

  describe('formatNumber', () => {
    it('arrondit avec 1 décimale par défaut', () => {
      expect(formatNumber(3.456)).toBe('3.5')
    })
    it('respecte decimals', () => {
      expect(formatNumber(3.456, 2)).toBe('3.46')
    })
    it('NaN/Infinity → "-"', () => {
      expect(formatNumber(NaN)).toBe('-')
      expect(formatNumber(Infinity)).toBe('-')
    })
  })
})

// ---------------------------------------------------------------------------
// buildDualGridOption
// ---------------------------------------------------------------------------

function makeOpt(playerName: string, data: number[]): EChartsCoreOption {
  return {
    xAxis: { type: 'category', data: data.map((_, i) => `#${i + 1}`) },
    yAxis: { type: 'value' },
    series: [{ name: playerName, type: 'bar', data }],
    legend: { data: [playerName] },
    tooltip: { trigger: 'axis' },
  }
}

describe('buildDualGridOption', () => {
  const optA = makeOpt('PlayerA', [1, 2, 3])
  const optB = makeOpt('PlayerB', [4, 5, 6])

  it('side-by-side → 2 grilles horizontales (gauche/droite)', () => {
    const opt = buildDualGridOption(optA, optB, 'Titre A', 'Titre B', 'side-by-side')
    const grids = opt.grid as Array<{ left: string; right: string }>
    expect(grids).toHaveLength(2)
    expect(grids[0].right).toBe('54%')
    expect(grids[1].left).toBe('54%')
  })

  it('stacked → 2 grilles verticales (haut/bas)', () => {
    const opt = buildDualGridOption(optA, optB, 'Titre A', 'Titre B', 'stacked')
    const grids = opt.grid as Array<{ height?: string; top?: number | string }>
    expect(grids).toHaveLength(2)
    expect(grids[0].height).toBe('40%')
    expect(grids[1].top).toBe('56%')
  })

  it('xAxis/yAxis — 2 axes avec gridIndex 0 et 1', () => {
    const opt = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    const xAxes = opt.xAxis as Array<{ gridIndex: number }>
    const yAxes = opt.yAxis as Array<{ gridIndex: number }>
    expect(xAxes[0].gridIndex).toBe(0)
    expect(xAxes[1].gridIndex).toBe(1)
    expect(yAxes[0].gridIndex).toBe(0)
    expect(yAxes[1].gridIndex).toBe(1)
  })

  it('séries optB ré-indexées gridIndex/xAxisIndex/yAxisIndex = 1', () => {
    const opt = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    const series = opt.series as Array<{ name: string; gridIndex?: number; xAxisIndex?: number; yAxisIndex?: number }>
    const sA = series.find((s) => s.name === 'PlayerA')!
    const sB = series.find((s) => s.name === 'PlayerB')!
    expect(sA.gridIndex).toBeUndefined()
    expect(sB.gridIndex).toBe(1)
    expect(sB.xAxisIndex).toBe(1)
    expect(sB.yAxisIndex).toBe(1)
  })

  it('axisPointer.link présent pour crosshair synchronisé', () => {
    const opt = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    expect(opt.axisPointer).toMatchObject({ link: [{ xAxisIndex: 'all' }] })
  })

  it('dataZoom ajouté uniquement en mode stacked', () => {
    const sideBySide = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    const stacked = buildDualGridOption(optA, optB, 'A', 'B', 'stacked')
    expect(sideBySide.dataZoom).toBeUndefined()
    expect(stacked.dataZoom).toBeDefined()
  })

  it('légendes fusionnées sans doublons', () => {
    const opt = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    const legend = opt.legend as { data: string[] }
    expect(legend.data).toEqual(['PlayerA', 'PlayerB'])
  })

  it('joueur présent dans les deux légendes → pas de doublon', () => {
    const bothSame = makeOpt('SharedPlayer', [1])
    const opt = buildDualGridOption(bothSame, bothSame, 'A', 'B', 'side-by-side')
    const legend = opt.legend as { data: string[] }
    expect(legend.data).toEqual(['SharedPlayer'])
  })

  it('valueFormatter gère valeurs négatives (deaths butterfly)', () => {
    const opt = buildDualGridOption(optA, optB, 'A', 'B', 'side-by-side')
    const tooltip = opt.tooltip as { valueFormatter: (v: unknown) => string }
    expect(tooltip.valueFormatter(-5)).toBe('5.0')
    expect(tooltip.valueFormatter(3.14)).toBe('3.1')
    expect(tooltip.valueFormatter('text')).toBe('-')
  })
})

describe('dualPanelHeight', () => {
  it('side-by-side → 300px', () => {
    expect(dualPanelHeight('side-by-side')).toBe(300)
  })
  it('stacked → 600px', () => {
    expect(dualPanelHeight('stacked')).toBe(600)
  })
})

describe('DUAL_LAYOUT_THRESHOLD', () => {
  it('seuil de bascule = 14 matchs', () => {
    expect(DUAL_LAYOUT_THRESHOLD).toBe(14)
  })
})
