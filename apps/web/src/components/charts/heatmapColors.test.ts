import { describe, expect, it } from 'vitest'

import type { ColorPalette } from '@/stores/settingsDraftStore'

import { heatmapRampTokens } from './heatmapColors'

const CVD_PALETTES: ColorPalette[] = ['okabe-ito', 'cividis', 'tol-bright']
const ALL_PALETTES: ColorPalette[] = ['default', ...CVD_PALETTES]

describe('heatmapRampTokens', () => {
  it("mode 'divergent' : rampe signée bas → neutre → haut (toute palette)", () => {
    for (const p of ALL_PALETTES) {
      expect(heatmapRampTokens('divergent', p)).toEqual([
        'heatmap-divergent-low',
        'divergent-neutral',
        'heatmap-divergent-high',
      ])
    }
  })

  it("mode 'frequency' : rampe neutre de fréquence (identique dans toute palette)", () => {
    for (const p of ALL_PALETTES) {
      expect(heatmapRampTokens('frequency', p)).toEqual(['heatmap-freq-low', 'heatmap-freq-high'])
    }
  })

  it("mode 'sequential' + palette default : rampe cold→hot familière conservée", () => {
    expect(heatmapRampTokens('sequential', 'default')).toEqual(['heatmap-cold', 'heatmap-hot'])
  })

  it("mode 'sequential' + palette CVD : bascule sur la rampe fréquence (luminance monotone)", () => {
    for (const p of CVD_PALETTES) {
      expect(heatmapRampTokens('sequential', p)).toEqual(['heatmap-freq-low', 'heatmap-freq-high'])
    }
  })

  it('colorPalette omis = default (séquentiel cold→hot, aucune régression)', () => {
    expect(heatmapRampTokens('sequential')).toEqual(['heatmap-cold', 'heatmap-hot'])
  })
})
