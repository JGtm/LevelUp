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

describe('ProductionCard — lignes « non mesuré » et notes (Vue match)', () => {
  const rowKeys = () => [...document.querySelectorAll<HTMLElement>('[data-testid^="piste-camps-row-"]')].map((r) => r.dataset.testid)

  it('`pending` : rangées avec les lignes mesurées dans l’ordre des ressources, texte à la place de la barre', () => {
    render(
      <ProductionCard
        rows={ROWS}
        pending={[{ resource: 'powerup', text: 'Aucun temps d’effet mesuré' }, { resource: 'vehicle', text: 'Non mesuré' }]}
        t={EMPRISE_TEXT.fr}
      />,
    )
    expect(rowKeys()).toEqual(['piste-camps-row-powerup', 'piste-camps-row-power_weapon', 'piste-camps-row-vehicle'])
    expect(screen.getByTestId('piste-camps-pending-powerup').textContent).toBe('Aucun temps d’effet mesuré')
  })

  it('une ligne « non mesuré » garde la barre fine de son exposition', () => {
    render(
      <ProductionCard
        rows={[]}
        pending={[{ resource: 'powerup', text: 'Frags pendant l’effet non mesurés', exposure: { kind: 'effect_ms', value: { us: 166700, them: 216900 } } }]}
        t={EMPRISE_TEXT.fr}
      />,
    )
    expect(screen.getByTestId('emprise-production-exposure-powerup')).toBeInTheDocument()
    expect(screen.getByText(EMPRISE_TEXT.fr.production.thinLegend)).toBeInTheDocument()
  })

  it('`notes` : une ligne atténuée sous la barre de la ressource nommée', () => {
    render(<ProductionCard rows={ROWS} notes={{ power_weapon: 'aucune prise d’arme spéciale mesurée' }} t={EMPRISE_TEXT.fr} />)
    expect(screen.getByTestId('emprise-production-note-power_weapon').textContent).toBe('aucune prise d’arme spéciale mesurée')
  })
})
