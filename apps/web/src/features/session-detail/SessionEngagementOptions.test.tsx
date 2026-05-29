/**
 * Tests des helpers purs de SessionEngagementOptions (binning + option courbe/moyenne).
 */
import { describe, expect, it } from 'vitest'

import { binValues, buildEngagementLineOption } from './SessionEngagementOptions'

describe('binValues', () => {
  it('découpe en K bins en conservant le nombre total', () => {
    const bins = binValues([0, 1, 2, 3, 4, 5, 6, 7], 4)
    expect(bins).toHaveLength(4)
    expect(bins.reduce((acc, b) => acc + b.count, 0)).toBe(8)
  })

  it('valeurs identiques → un seul bin', () => {
    expect(binValues([2, 2, 2])).toEqual([{ binStart: 2, binEnd: 2, count: 3 }])
  })

  it('liste vide → []', () => {
    expect(binValues([])).toEqual([])
  })
})

describe('buildEngagementLineOption', () => {
  it('produit une ligne + une markLine de moyenne', () => {
    const opt = buildEngagementLineOption(
      [
        {
          key: 'e',
          datapoints: [
            { x: 0, y: 2 },
            { x: 1, y: 4 },
          ],
        },
      ],
      { meanLabel: 'Moyenne' },
    ) as unknown as { series?: Array<{ type: string; markLine: { data: Array<{ yAxis: number }> } }> }
    expect(opt.series![0].type).toBe('line')
    expect(opt.series![0].markLine.data[0].yAxis).toBe(3) // (2 + 4) / 2
  })
})
