/**
 * SquadKillMechanicsChart.test — sur un titre qui porte la capability `native_kill_mechanics`, une
 * sélection sans mécanique de frag garde la carte (titre et état vide), au lieu de la retirer.
 */
import { screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'

import { SquadKillMechanicsChart } from './SquadKillMechanicsChart'

vi.mock('echarts-for-react', () => ({ default: () => <div data-testid="echarts-mock" /> }))

describe('SquadKillMechanicsChart', () => {
  it('sans donnée ou sans barre : titre gardé et état vide dans la carte', async () => {
    for (const data of [undefined, null, { players: ['A'], bars: [] }]) {
      const { unmount } = renderWithProviders(
        <SquadKillMechanicsChart data={data} title="Mécaniques de frag" colorByPlayer={{}} labelOf={(m) => m} />,
      )
      expect(await screen.findByText('Mécaniques de frag')).toBeInTheDocument()
      expect(screen.getByTestId('chart-card-empty').querySelector('.border-dashed')).not.toBeNull()
      expect(screen.queryByTestId('echarts-mock')).toBeNull()
      unmount()
    }
  })

  it('avec des barres : le graphe est monté', async () => {
    renderWithProviders(
      <SquadKillMechanicsChart
        data={{ players: ['A'], bars: [{ mechanic: 'assassination', kills_by_player: { A: 3 }, total_squad: 3 }] }}
        title="Mécaniques de frag"
        colorByPlayer={{ A: 'var(--c)' }}
        labelOf={(m) => m}
      />,
    )
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(within(document.body).queryByTestId('chart-card-empty')).toBeNull()
  })
})
