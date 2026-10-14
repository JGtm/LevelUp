import { describe, expect, it } from 'vitest'

import { barLabel } from '@/components/charts/barLabel'

import { inkOn, resolveCssColor, SEGMENT_INK } from './segmentInk'

describe('inkOn', () => {
  it('encre sombre sur un aplat clair, claire sur un aplat sombre (meilleur contraste WCAG)', () => {
    expect(inkOn('#fde68a')).toBe(SEGMENT_INK.dark)
    expect(inkOn('#f59e0b')).toBe(SEGMENT_INK.dark) // ambre : le blanc n'y atteint pas 2:1
    expect(inkOn('#1e3a8a')).toBe(SEGMENT_INK.light)
    expect(inkOn('rgb(16, 24, 40)')).toBe(SEGMENT_INK.light)
  })

  it('couleur non résolue (jsdom, variable inconnue) : encre claire', () => {
    expect(inkOn('var(--absente)')).toBe(SEGMENT_INK.light)
    expect(inkOn('')).toBe(SEGMENT_INK.light)
  })

  it('un aplat translucide se juge sur son mélange avec le fond', () => {
    // Bleu nuit à 40 % sur blanc : le mélange est clair, l'encre est sombre.
    expect(inkOn('#1e3a8a')).toBe(SEGMENT_INK.light)
    expect(inkOn('#1e3a8a', { color: '#ffffff', opacity: 0.4 })).toBe(SEGMENT_INK.dark)
  })
})

describe('resolveCssColor', () => {
  it('hex court, hex long et rgb() se normalisent en #rrggbb', () => {
    expect(resolveCssColor('#abc')).toBe('#aabbcc')
    expect(resolveCssColor('#112233')).toBe('#112233')
    expect(resolveCssColor('rgb(255, 0, 16)')).toBe('#ff0010')
  })
})

describe('barLabel', () => {
  it('étiquette interne centrée sur la hauteur, à l’encre du segment ; le reste se surcharge', () => {
    const l = barLabel('#f59e0b', { fontSize: 10 })
    expect(l).toMatchObject({ show: true, position: 'inside', verticalAlign: 'middle', color: SEGMENT_INK.dark, fontSize: 10 })
  })
})
