import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { RelationWinRateDonut } from './RelationWinRateDonut'

const LABELS = {
  wins: 'Victoires',
  losses: 'Défaites',
  personalAvg: 'Moy. perso',
  pointsUnit: 'pts',
  liftTooltip: 'Écart vs ta moyenne perso.',
}

describe('RelationWinRateDonut', () => {
  it('rend le donut, le repère de moyenne perso et le delta signé', () => {
    const { container } = render(
      <RelationWinRateDonut winRate={0.62} personalAvg={0.54} labels={LABELS} />,
    )

    // % de la relation présent (centre + légende victoires).
    expect(screen.getAllByText('62%').length).toBeGreaterThan(0)
    // Légende : défaites (100 - 62) + repère moyenne perso.
    expect(screen.getByText('38%')).toBeInTheDocument()
    expect(screen.getByText('Moy. perso')).toBeInTheDocument()
    expect(screen.getByText('54%')).toBeInTheDocument()

    // Repère sur l'anneau = une <line> SVG (présente car moyenne perso fournie).
    expect(container.querySelectorAll('svg line')).toHaveLength(1)
    // Delta signé « +8 pts » (0.62 − 0.54 = +8 pts), porté par le tooltip d'écart.
    const delta = container.querySelector('[title="Écart vs ta moyenne perso."]')
    expect(delta?.textContent).toMatch(/8\s*pts/)
    expect(delta?.textContent?.startsWith('+')).toBe(true)
  })

  it('sans moyenne perso : pas de repère ni de delta', () => {
    const { container } = render(
      <RelationWinRateDonut winRate={0.4} personalAvg={null} labels={LABELS} />,
    )
    expect(screen.getAllByText('40%').length).toBeGreaterThan(0)
    expect(container.querySelectorAll('svg line')).toHaveLength(0)
    expect(screen.queryByText('Moy. perso')).not.toBeInTheDocument()
    expect(container.querySelector('[title]')).toBeNull()
  })

  it('taux de victoire absent : placeholder « — »', () => {
    render(<RelationWinRateDonut winRate={null} personalAvg={0.5} labels={LABELS} />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('en dessous de la moyenne : arc déficit (rouge) + delta négatif', () => {
    const { container } = render(
      <RelationWinRateDonut winRate={0.25} personalAvg={0.51} labels={LABELS} />,
    )
    // Arc déficit = un <path> SVG (comble le vide entre le taux et le repère).
    expect(container.querySelectorAll('svg path').length).toBeGreaterThan(0)
    const delta = container.querySelector('[title="Écart vs ta moyenne perso."]')
    expect(delta?.textContent?.startsWith('−')).toBe(true)
  })

  it('au-dessus de la moyenne : aucun arc déficit', () => {
    const { container } = render(
      <RelationWinRateDonut winRate={0.62} personalAvg={0.54} labels={LABELS} />,
    )
    expect(container.querySelectorAll('svg path')).toHaveLength(0)
  })
})
