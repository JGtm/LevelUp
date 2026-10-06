/**
 * Tests de la section « Évolution » : bascule de pas (options selon l'horizon), graphiques
 * rendus sous leur grille, état vide. ECharts est simulé (jsdom n'a pas de canvas) : le double
 * expose l'option reçue.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { response, seriesIndicator } from './tendances.fixture'
import { TendancesEvolution } from './TendancesEvolution'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (...args: unknown[]) => get(...args) } }
})

beforeEach(() => {
  get.mockReset()
  get.mockResolvedValue({
    title_slug: 'halo_infinite',
    schema_version: 1,
    locale: 'fr',
    fields: { kda: { label: 'FDA' }, enemy_mmr: { label: 'MMR adverse' } },
  })
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite', isBootstrapped: true })
})

const DATA = response({
  indicators: [
    seriesIndicator('kda'),
    seriesIndicator('enemy_mmr'),
    seriesIndicator('avg_damage_dealt'),
  ],
})

function renderSection(horizon: 7 | 30 | 90 | 365 = 90, step: 'match' | 'day' | 'week' | 'month' = 'week') {
  const onStepChange = vi.fn()
  renderWithProviders(
    <TendancesEvolution
      locale="fr"
      data={DATA}
      horizon={horizon}
      step={step}
      onStepChange={onStepChange}
    />,
  )
  return onStepChange
}

const boutonsDuPas = () =>
  within(screen.getByRole('group', { name: 'Pas de temps' }))
    .getAllByRole('button')
    .map((b) => b.textContent)

describe('TendancesEvolution — bascule de pas', () => {
  it.each([
    [7, ['Par match', 'Par jour']],
    [30, ['Par match', 'Par jour', 'Par semaine']],
    [90, ['Par jour', 'Par semaine', 'Par mois']],
    [365, ['Par semaine', 'Par mois']],
  ] as const)('horizon %i j : options %j', (horizon, attendu) => {
    renderSection(horizon, attendu[0] === 'Par match' ? 'match' : 'week')
    expect(boutonsDuPas()).toEqual(attendu)
  })

  it('le pas courant est pressé et un clic remonte le pas choisi', () => {
    const onStepChange = renderSection(90, 'week')
    const groupe = screen.getByRole('group', { name: 'Pas de temps' })
    expect(within(groupe).getByRole('button', { name: 'Par semaine' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    fireEvent.click(within(groupe).getByRole('button', { name: 'Par mois' }))
    expect(onStepChange).toHaveBeenCalledWith('month')
  })
})

describe('TendancesEvolution — graphiques', () => {
  it('un titre « Évolution », puis les grilles sous leur intertitre', async () => {
    renderSection()
    expect(screen.getByRole('heading', { name: /^Évolution/ })).toBeInTheDocument()
    const grille = screen.getByTestId('tendances-grid-versus-mmr')
    expect(within(grille).getByRole('heading', { name: 'Face au MMR adverse' })).toBeInTheDocument()
    expect(within(grille).getByTestId('tendances-chart-kda')).toBeInTheDocument()
    expect(
      within(screen.getByTestId('tendances-grid-level')).getByTestId('tendances-chart-damage'),
    ).toBeInTheDocument()
    const stub = await within(screen.getByTestId('tendances-chart-kda')).findByTestId('echarts-stub')
    const option = JSON.parse(stub.textContent ?? '{}')
    expect(option.xAxis.type).toBe('time')
    expect(option.yAxis).toHaveLength(2)
  })

  it('titre d’un graphique : « {libellé} face au MMR adverse »', async () => {
    renderSection()
    expect(await screen.findByText('FDA face au MMR adverse')).toBeInTheDocument()
  })

  it('état vide : aucune courbe à tracer sur cet horizon et ce pas', () => {
    renderWithProviders(
      <TendancesEvolution
        locale="fr"
        data={response({ indicators: [seriesIndicator('kda', { n: 1 })] })}
        horizon={90}
        step="week"
        onStepChange={() => {}}
      />,
    )
    expect(
      screen.getByText('Pas assez de matchs sur cet horizon pour ce pas de temps.'),
    ).toBeInTheDocument()
    expect(screen.queryByTestId('tendances-grid-versus-mmr')).toBeNull()
    expect(screen.getByRole('group', { name: 'Pas de temps' })).toBeInTheDocument()
  })
})
