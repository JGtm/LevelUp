import { describe, it, expect, beforeEach } from 'vitest'
import { buildDivergentColorscale, buildOrdinalColorscale, getSeriesColors } from '../plotlyColorscale'

beforeEach(() => {
  // Inject CSS vars so resolveToken() returns predictable values in jsdom
  document.documentElement.style.setProperty('--ac-divergent-neg', '#FF0000')
  document.documentElement.style.setProperty('--ac-divergent-neutral', '#808080')
  document.documentElement.style.setProperty('--ac-divergent-pos', '#00FF00')
  document.documentElement.style.setProperty('--ac-perf-tier-1', '#00FF00')
  document.documentElement.style.setProperty('--ac-perf-tier-2', '#80FF00')
  document.documentElement.style.setProperty('--ac-perf-tier-3', '#FFFF00')
  document.documentElement.style.setProperty('--ac-perf-tier-4', '#FF8000')
  document.documentElement.style.setProperty('--ac-perf-tier-5', '#FF0000')
})

describe('buildDivergentColorscale', () => {
  it('retourne 3 stops [0, 0.5, 1] dans l\'ordre négatif → neutre → positif', () => {
    const cs = buildDivergentColorscale('divergent-neg', 'divergent-neutral', 'divergent-pos')
    expect(cs).toHaveLength(3)
    expect(cs[0][0]).toBe(0)
    expect(cs[1][0]).toBe(0.5)
    expect(cs[2][0]).toBe(1)
    expect(cs[0][1]).toBe('#FF0000')
    expect(cs[1][1]).toBe('#808080')
    expect(cs[2][1]).toBe('#00FF00')
  })
})

describe('buildOrdinalColorscale', () => {
  it('répartit N tokens en stops équidistants', () => {
    const cs = buildOrdinalColorscale(['perf-tier-5', 'perf-tier-3', 'perf-tier-1'])
    expect(cs).toHaveLength(3)
    expect(cs[0][0]).toBe(0)
    expect(cs[1][0]).toBe(0.5)
    expect(cs[2][0]).toBe(1)
  })

  it('gère 1 seul token (stop 0 et 1 identiques)', () => {
    const cs = buildOrdinalColorscale(['perf-tier-1'])
    expect(cs).toHaveLength(2)
    expect(cs[0][0]).toBe(0)
    expect(cs[1][0]).toBe(1)
    expect(cs[0][1]).toBe(cs[1][1])
  })
})

describe('getSeriesColors', () => {
  it('retourne N couleurs en cyclant sur les tokens', () => {
    const colors = getSeriesColors(5, ['perf-tier-1', 'perf-tier-2', 'perf-tier-3'])
    expect(colors).toHaveLength(5)
    expect(colors[0]).toBe(colors[3])
    expect(colors[1]).toBe(colors[4])
  })

  it('retourne 0 couleurs pour n=0', () => {
    expect(getSeriesColors(0, ['perf-tier-1'])).toHaveLength(0)
  })
})
