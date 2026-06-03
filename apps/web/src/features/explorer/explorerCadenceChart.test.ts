/**
 * Tests buildCadenceBarsOption — barres divergentes K/D/A (Morts sous l'axe).
 */
import { describe, expect, it } from 'vitest'

import { buildCadenceBarsOption } from './explorerCadenceChart'

interface BarOption {
  series: Array<{ data: Array<{ value: number }> }>
  xAxis: { data: string[] }
}

describe('buildCadenceBarsOption', () => {
  it('frags & assistances positifs, morts négatives (sous l\'axe zéro)', () => {
    const opt = buildCadenceBarsOption(
      { kills: 12, deaths: 8, assists: 4 },
      { frags: 'Frags', deaths: 'Morts', assists: 'Assists' },
      1,
    ) as unknown as BarOption

    const data = opt.series[0].data
    expect(data[0].value).toBe(12) // frags au-dessus
    expect(data[1].value).toBe(-8) // morts sous l'axe (négatif)
    expect(data[2].value).toBe(4) // assistances au-dessus
    expect(opt.xAxis.data).toEqual(['Frags', 'Morts', 'Assists'])
  })
})
