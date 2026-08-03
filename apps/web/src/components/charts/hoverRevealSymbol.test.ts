/**
 * hoverRevealSymbol.test.ts — affordance de survol partagée (v7.3 lot 2, item 2.3c).
 *
 * Épingle l'invariant qui la rend efficace : le symbole est CACHÉ au repos
 * (`showSymbol: false`) mais DÉFINI (`symbol` ≠ 'none'). Avec `symbol: 'none'`,
 * ECharts n'affiche aucun point à l'emphase et le graphe se lit comme une image
 * figée — c'est exactement le défaut corrigé sur l'écart FDA et l'intensité.
 */
import { describe, expect, it } from 'vitest'

import { hoverRevealSymbol } from './_utils'

describe('hoverRevealSymbol', () => {
  it('symbole caché au repos mais défini (condition de l affichage à l emphase)', () => {
    const s = hoverRevealSymbol('#abcdef')
    expect(s.showSymbol).toBe(false)
    expect(s.symbol).toBe('circle')
    expect(s.symbol).not.toBe('none')
  })

  it('couleur de point et taille par défaut / surchargée', () => {
    expect(hoverRevealSymbol('#abcdef').itemStyle.color).toBe('#abcdef')
    expect(hoverRevealSymbol('#abcdef').symbolSize).toBe(7)
    expect(hoverRevealSymbol('#abcdef', 6).symbolSize).toBe(6)
  })

  it('grossissement à l emphase (le point survolé se distingue)', () => {
    expect(hoverRevealSymbol('#abcdef').emphasis.scale).toBeGreaterThan(1)
  })
})
