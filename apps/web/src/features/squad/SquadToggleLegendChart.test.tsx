/**
 * SquadToggleLegendChart.test.tsx — légende React des sous-charts performance.
 *
 * Couvre l'aide contextuelle par type (v7.3 lot 2, item 2.2) : un type porteur
 * d'un texte `info` rend une icône ⓘ À CÔTÉ de son bouton (jamais dedans :
 * InfoTooltip est lui-même un `<button>`), un type sans `info` rend exactement
 * l'ancien bouton nu, et le bouton reste un toggle dans les deux cas.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { SquadToggleLegendChart } from './SquadToggleLegendChart'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const SERIES = [{ key: 's', datapoints: [1] }]

function renderLegend(info?: string) {
  return render(
    <SquadToggleLegendChart
      title="Frags / Morts"
      series={SERIES}
      players={['Me']}
      colorByPlayer={{ Me: '#aaa' }}
      types={[
        { key: 'Frags', label: 'Frags' },
        { key: 'Bonus', label: 'Bonus', info },
      ]}
      initialHiddenTypes={new Set(['Bonus'])}
      buildOption={() => ({})}
    />,
  )
}

describe('SquadToggleLegendChart — aide contextuelle par type', () => {
  it('type avec `info` : une icône ⓘ rendue à côté du bouton, pas dedans', () => {
    renderLegend('Bonus = assistances ÷ 3.')
    const bonus = screen.getByRole('button', { name: 'Bonus' })
    const infoIcon = screen.getByRole('button', { name: /info/i })
    expect(infoIcon).toBeInTheDocument()
    expect(bonus.contains(infoIcon)).toBe(false)
  })

  it('le tooltip affiche le texte au survol de l icône', () => {
    renderLegend('Bonus = assistances ÷ 3.')
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Bonus = assistances ÷ 3.')
  })

  it('type sans `info` : aucune icône ⓘ ajoutée', () => {
    renderLegend(undefined)
    expect(screen.queryByRole('button', { name: /info/i })).toBeNull()
  })

  it('le bouton reste un toggle (aria-pressed bascule)', () => {
    renderLegend('Bonus = assistances ÷ 3.')
    const bonus = screen.getByRole('button', { name: 'Bonus' })
    // Masqué au montage (initialHiddenTypes).
    expect(bonus).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(bonus)
    expect(bonus).toHaveAttribute('aria-pressed', 'true')
  })
})
