import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { AssistTierBar } from './AssistTierBar'
import { assistShareSegments } from './assistExchange'
import { ASSISTS_TEXT } from './assistsI18n'

describe('AssistTierBar', () => {
  it('pose trois segments aux largeurs = parts, du ton clair au ton plein', () => {
    // 7 des 12 frags assistés : 2 / 3 / 1 par tranche, 1 sans part mesurée (non dessinée).
    const segments = assistShareSegments({ total: 7, low: 2, mid: 3, high: 1 }, 12)
    const { getByTestId } = render(
      <AssistTierBar segments={segments} color="var(--c)" text={ASSISTS_TEXT.fr} locale="fr" variant="tile" testId="seg" />,
    )
    // Tons = opacités 35 / 65 / 100 % de la couleur du sens (skill color-tokens).
    const OPACITY = { low: '0.35', mid: '0.65', high: '1' } as const
    const widths = (['low', 'mid', 'high'] as const).map((tier) => {
      const el = getByTestId(`seg-${tier}`)
      expect(el.style.opacity).toBe(OPACITY[tier])
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
