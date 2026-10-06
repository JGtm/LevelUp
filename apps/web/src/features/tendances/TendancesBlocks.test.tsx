/**
 * Tests de rendu des quatre derniers blocs de la vue Solo : calendrier, « Victoires et
 * défaites », médailles, types de partie, dans leurs états (données, vide). ECharts est simulé
 * (jsdom n'a pas de canvas) : le double expose l'option reçue.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { response } from './tendances.fixture'
import { TendancesCalendar } from './TendancesCalendar'
import { TendancesMix } from './TendancesMix'
import { TendancesMedals, TendancesWinLoss } from './TendancesOutcomes'

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
    fields: { kda: { label: 'FDA' }, accuracy: { label: 'Précision' } },
  })
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite', isBootstrapped: true })
})

const optionIn = async (testId: string) => {
  const stub = await within(screen.getByTestId(testId)).findByTestId('echarts-stub')
  return JSON.parse(stub.textContent ?? '{}')
}

describe('TendancesCalendar', () => {
  it('jours joués : carte titrée, grille de cases, rampe divergente 0..1 sans réglette', async () => {
    const data = response({
      calendar: [
        { date: '2026-10-01', matches: 4, wins: 3, losses: 1, win_rate: 0.75 },
        { date: '2026-10-05', matches: 2, wins: 0, losses: 2, win_rate: 0 },
      ],
    })
    renderWithProviders(<TendancesCalendar locale="fr" data={data} horizon={7} />)
    expect(screen.getByRole('heading', { name: /Calendrier des résultats/ })).toBeInTheDocument()
    const option = await optionIn('tendances-calendar')
    expect(option.series[0].type).toBe('heatmap')
    expect(option.series[0].data).toHaveLength(14)
    expect(option.visualMap.show).toBe(false)
    expect(option.visualMap.min).toBe(0)
    expect(option.visualMap.max).toBe(1)
    expect(option.yAxis.inverse).toBe(true)
  })

  it('aucun jour joué sur l’horizon : la notice d’état vide, pas de graphique', () => {
    const data = response({
      calendar: [{ date: '2026-01-01', matches: 1, wins: 1, losses: 0, win_rate: 1 }],
    })
    renderWithProviders(<TendancesCalendar locale="fr" data={data} horizon={7} />)
    expect(screen.getByText('Aucun jour joué')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
  })
})

describe('TendancesWinLoss', () => {
  const rows = [
    { key: 'kda', group: 'stats', matches: 42, win_mean: 5.678, loss_mean: 1.234, z_win: 0.8, z_loss: -0.6, r: 0.35 },
    { key: 'accuracy', group: 'stats', matches: 42, win_mean: 0.5, loss_mean: 0.45, z_win: 0.3, z_loss: -0.2, r: 0.1 },
  ]

  it('sous le titre de section : le graphique, deux séries de points, libellés de champ', async () => {
    renderWithProviders(
      <TendancesWinLoss
        locale="fr"
        data={response({ win_loss: [{ days: 90, matches: 60, required: 20, rows }] })}
        horizon={90}
      />,
    )
    expect(screen.getByRole('heading', { name: 'Victoires et défaites' })).toBeInTheDocument()
    expect(screen.getByText('Moyenne par match en défaite et en victoire')).toBeInTheDocument()
    const option = await optionIn('tendances-winloss')
    expect(option.series.map((s: { type: string }) => s.type)).toEqual(['custom', 'scatter', 'scatter'])
    expect(option.series[1].name).toBe('Défaites')
    expect(option.series[2].name).toBe('Victoires')
    expect(option.xAxis.axisLabel).toEqual({ show: false })
    expect(option.legend.left).toBe('center')
    await vi.waitFor(async () => {
      expect((await optionIn('tendances-winloss')).yAxis.data).toEqual(['FDA', 'Précision'])
    })
  })

  it('pas assez de matchs : l’état vide dit combien il en faut', () => {
    renderWithProviders(
      <TendancesWinLoss
        locale="fr"
        data={response({ win_loss: [{ days: 90, matches: 4, required: 10, rows: [] }] })}
        horizon={90}
      />,
    )
    expect(
      screen.getByText('Pas assez de matchs sur cet horizon : 4 victoires et défaites, il en faut 10.'),
    ).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
  })

  it('suit l’horizon : un bloc absent pour cet horizon donne l’état vide', () => {
    renderWithProviders(
      <TendancesWinLoss
        locale="fr"
        data={response({ win_loss: [{ days: 90, matches: 60, required: 20, rows }] })}
        horizon={7}
      />,
    )
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
    expect(screen.getByText(/Pas assez de matchs sur cet horizon/)).toBeInTheDocument()
  })
})

describe('TendancesMedals', () => {
  const block = {
    days: 30,
    compared: true,
    rows: [{ medal_id: 1, name: 'Tueur', rate: 0.5, prev_rate: 0.4 }],
  }

  it('comparé : deux séries nommées par la durée, axe X gradué', async () => {
    renderWithProviders(
      <TendancesMedals locale="fr" data={response({ medals: [block] })} horizon={30} />,
    )
    expect(screen.getByText('Médailles par match')).toBeInTheDocument()
    const option = await optionIn('tendances-medals')
    expect(option.series[1].name).toBe("30 jours d'avant")
    expect(option.series[2].name).toBe('30 derniers jours')
    expect(option.legend.data.map((e: { name: string }) => e.name)).toEqual([
      "30 jours d'avant",
      '30 derniers jours',
    ])
    expect(option.xAxis.axisLabel.show).not.toBe(false)
    expect(option.yAxis.data).toEqual(['Tueur'])
  })

  it('non comparé : un seul point, la légende ne nomme que l’horizon', async () => {
    renderWithProviders(
      <TendancesMedals
        locale="fr"
        data={response({ medals: [{ ...block, compared: false }] })}
        horizon={30}
      />,
    )
    const option = await optionIn('tendances-medals')
    expect(option.series[1].data).toEqual([])
    expect(option.legend.data.map((e: { name: string }) => e.name)).toEqual(['30 derniers jours'])
  })

  it('bloc absent ou sans ligne : état vide', () => {
    const { unmount } = renderWithProviders(
      <TendancesMedals locale="fr" data={response({ medals: [] })} horizon={30} />,
    )
    expect(screen.getByText('Aucune médaille')).toBeInTheDocument()
    unmount()
    renderWithProviders(
      <TendancesMedals
        locale="fr"
        data={response({ medals: [{ days: 30, compared: false, rows: [] }] })}
        horizon={30}
      />,
    )
    expect(screen.getByText('Aucune médaille')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
  })
})

describe('TendancesMix', () => {
  const data = response({
    mix: {
      day: [
        { t: '2026-09-30T22:00:00Z', counts: { ranked_slayer: 3, arena_slayer: 1 } },
        { t: '2026-10-04T22:00:00Z', counts: { arena_slayer: 2 } },
      ],
      week: [],
      month: [],
    },
  })

  it('un bâton par intervalle, un composant par type de partie, légende du wrapper', async () => {
    renderWithProviders(<TendancesMix locale="fr" data={data} horizon={7} step="match" />)
    expect(screen.getByText('Matchs par type de partie')).toBeInTheDocument()
    const option = await optionIn('tendances-mix')
    expect(option.xAxis.data).toHaveLength(2)
    expect(option.series.map((s: { name: string }) => s.name)).toEqual([
      'Classé · Assassin',
      'Social · Assassin',
    ])
    expect(option.series[0].data).toEqual([3, 0])
    expect(option.legend.bottom).toBe(0)
  })

  it('aucun intervalle sur l’horizon (pas hebdomadaire vide) : état vide du wrapper', () => {
    renderWithProviders(<TendancesMix locale="fr" data={data} horizon={7} step="week" />)
    expect(screen.getByText('Aucun match sur cet horizon.')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
  })
})
