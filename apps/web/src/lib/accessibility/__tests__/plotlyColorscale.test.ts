import { describe, it, expect, beforeEach } from 'vitest'
import { getSeriesColors } from '../plotlyColorscale'

beforeEach(() => {
  document.documentElement.style.setProperty('--ac-perf-tier-1', '#00FF00')
  document.documentElement.style.setProperty('--ac-perf-tier-2', '#80FF00')
  document.documentElement.style.setProperty('--ac-perf-tier-3', '#FFFF00')
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
