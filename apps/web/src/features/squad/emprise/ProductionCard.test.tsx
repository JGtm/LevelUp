/**
 * ProductionCard.test.tsx — « Frags obtenus avec les ressources » en pleine page (compte · part,
 * ligne d'exposition en valeur) et en vue compacte (parts seules, ligne d'exposition en parts ;
 * maquette Sessions, `makeProd` avec `cp`).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ProductionCard } from './ProductionCard'
import { EMPRISE_TEXT } from './empriseStrings'
import type { ProductionRow } from './production.logic'

const ROWS: ProductionRow[] = [
  { resource: 'power_weapon', kills: { us: 30, them: 10 }, exposure: { kind: 'pickups', value: { us: 13, them: 21 } } },
]

const segmentTexts = () => [...document.querySelectorAll<HTMLElement>('[data-fit-label]')].map((l) => l.textContent)

describe('ProductionCard', () => {
  it('pleine page : compte · part, ligne d’exposition en valeur', () => {
    render(<ProductionCard rows={ROWS} t={EMPRISE_TEXT.fr} />)
    expect(segmentTexts()).toEqual(['30 · 75 %', '25 % · 10'])
    expect(screen.getByTestId('emprise-production-exposure-power_weapon').textContent).toBe(
      'prises sur les socles : 13 prises · 38,2 %21 prises',
    )
  })

  it('compact : parts entières seules, ligne d’exposition en parts', () => {
    render(
      <ProductionCard rows={ROWS} t={EMPRISE_TEXT.fr} compact={{ exposureLine: (name, pct) => `${name} : ${pct}` }} />,
    )
    expect(segmentTexts()).toEqual(['75 %', '25 %'])
    expect(screen.getByTestId('emprise-production-exposure-power_weapon').textContent).toBe('prises sur les socles : 38 %62 %')
  })
})
