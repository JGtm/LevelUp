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

  /**
   * A8 (2026-08-18) — RAMPE D'INTENSITÉ À TROIS POINTS : bleu -> rouge -> violet.
   *
   * Elle est IDENTIQUE dans toutes les palettes parce que ce sont des tokens : c'est la
   * palette qui les remappe, pas ce helper. Sous Okabe-Ito, la rampe devient Sky Blue ->
   * Vermillion -> Reddish Purple — trois couleurs de la même référence CUD, distinguables
   * entre elles sur les deux axes de confusion.
   */
  it("mode 'intensity' : bleu → rouge → violet, mêmes tokens dans toute palette", () => {
    for (const p of ALL_PALETTES) {
      expect(heatmapRampTokens('intensity', p)).toEqual(['info', 'destructive', 'extreme'])
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
