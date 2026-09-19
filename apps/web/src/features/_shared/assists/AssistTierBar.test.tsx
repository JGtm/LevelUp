import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { AssistTierBar } from './AssistTierBar'
import { assistTierTone } from './assistTierTone'
import { assistShareSegments } from './assistExchange'
import { ASSISTS_TEXT } from './assistsI18n'

describe('assistTierTone', () => {
  it('garde la couleur du sens telle quelle pour le ton faible', () => {
    expect(assistTierTone('var(--c)', 'low')).toBe('var(--c)')
  })

  it('monte les tons moyen et fort vers le premier plan du thème, chroma relevée, même teinte', () => {
    // Jamais une autre teinte : la clarté bouge, `h` reste celui de la couleur du sens.
    expect(assistTierTone('var(--c)', 'mid')).toBe(
      'light-dark(oklch(from var(--c) calc(l - 0.12) calc(c * 1.12) h), oklch(from var(--c) calc(l + 0.12) calc(c * 1.12) h))',
    )
    expect(assistTierTone('var(--c)', 'high')).toBe(
      'light-dark(oklch(from var(--c) calc(l - 0.24) calc(c * 1.24) h), oklch(from var(--c) calc(l + 0.24) calc(c * 1.24) h))',
    )
  })
})

describe('AssistTierBar', () => {
  it('pose trois segments aux largeurs = parts, du ton faible au ton fort', () => {
    // 7 des 12 frags assistés : 2 / 3 / 1 par tranche, 1 sans part mesurée (non dessinée).
    const segments = assistShareSegments({ total: 7, low: 2, mid: 3, high: 1 }, 12)
    const { getByTestId } = render(
      <AssistTierBar segments={segments} color="var(--c)" text={ASSISTS_TEXT.fr} locale="fr" variant="tile" testId="seg" />,
    )
    // Tons = trois clartés de la couleur du sens (skill color-tokens), plus d'opacité.
    const widths = (['low', 'mid', 'high'] as const).map((tier) => {
      const el = getByTestId(`seg-${tier}`)
      expect(el.style.backgroundColor).toBe(assistTierTone('var(--c)', tier))
      expect(el.style.opacity).toBe('')
      return parseFloat((el.closest('[style*="width"]') as HTMLElement).style.width)
    })
    expect(widths[0]).toBeCloseTo((2 / 12) * 100)
    expect(widths[1]).toBeCloseTo((3 / 12) * 100)
    expect(widths[2]).toBeCloseTo((1 / 12) * 100)
  })

  it('renverse l’ordre à gauche (du centre vers l’extérieur) et ne dessine pas une tranche vide', () => {
    const segments = assistShareSegments({ total: 3, low: 1, mid: 0, high: 2 }, 10)
    const { container, queryByTestId } = render(
      <AssistTierBar segments={segments} side="left" color="var(--c)" text={ASSISTS_TEXT.en} locale="en" variant="row" testId="seg" />,
    )
    expect(queryByTestId('seg-mid')).toBeNull()
    const order = [...container.querySelectorAll('[data-testid^="seg-"]')].map((el) => el.getAttribute('data-testid'))
    expect(order).toEqual(['seg-high', 'seg-low'])
  })
})
