/**
 * Tests de la page Tendances dans ses états : chargement, erreur (avec « Réessayer »), vide,
 * données (matrice, barre d’horizon, filtre de type de partie).
 *
 * ECharts est simulé (jsdom n’a pas de canvas, cf. `ActivityCalendarChart.test.tsx`) : le
 * double expose l’option reçue, ce qui permet de lire les cases de chaque grille.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { indicator, response, seriesIndicator } from './tendances.fixture'
import { TendancesTab } from './TendancesTab'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useParams: () => ({ playerSlug: 'jgtm' }) }
})

const post = vi.fn()
const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      post: (...args: unknown[]) => post(...args),
      get: (...args: unknown[]) => get(...args),
    },
  }
})

beforeEach(() => {
  post.mockReset()
  get.mockReset()
  get.mockResolvedValue({
    title_slug: 'halo_infinite',
    schema_version: 1,
    locale: 'fr',
    fields: { win_rate: { label: 'Taux de victoire' }, kda: { label: 'FDA' } },
  })
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    isBootstrapped: true,
  })
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('TendancesTab — états', () => {
  it('chargement : le texte de chargement, ni matrice ni barre', () => {
    post.mockReturnValue(new Promise(() => {}))
    renderWithProviders(<TendancesTab />)
    expect(screen.getByTestId('tendances-loading')).toBeInTheDocument()
    expect(screen.getByText('Chargement des tendances…')).toBeInTheDocument()
    expect(screen.queryByTestId('tendances-matrix')).toBeNull()
  })

  it('erreur : message et bouton « Réessayer » qui relance la requête', async () => {
    post.mockRejectedValueOnce(new Error('boom')).mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    expect(await screen.findByText('Les tendances n\'ont pas pu être chargées.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Réessayer' }))
    expect(await screen.findByTestId('tendances-matrix')).toBeInTheDocument()
    expect(post).toHaveBeenCalledTimes(2)
  })

  it('vide : aucun indicateur, la notice vide ; le filtre reste, la barre disparaît', async () => {
    post.mockResolvedValue(response({ indicators: [] }))
    renderWithProviders(<TendancesTab />)
    expect(await screen.findByText('Aucune tendance à afficher')).toBeInTheDocument()
    expect(screen.queryByTestId('tendances-matrix')).toBeNull()
    expect(screen.queryByTestId('tendances-horizon-bar')).toBeNull()
    expect(screen.getByLabelText('Type de partie')).toBeInTheDocument()
  })

  it('vide : indicateurs absents de la réponse (null)', async () => {
    post.mockResolvedValue(response({ indicators: null }))
    renderWithProviders(<TendancesTab />)
    expect(await screen.findByText('Aucune tendance à afficher')).toBeInTheDocument()
  })
})

describe('TendancesTab — données', () => {
  it('poste le corps solo (type vide, locale) sur la route de la page', async () => {
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    expect(post).toHaveBeenCalledWith(
      '/players/jgtm/pages/trends',
      { view: 'solo', game_type: '', locale: 'fr' },
      undefined,
      expect.anything(),
    )
  })

  it('matrice : titre de la carte, un intertitre et une grille par groupe, dans l’ordre', async () => {
    post.mockResolvedValue(
      response({
        indicators: [
          indicator({ key: 'kda', group: 'combat' }),
          indicator({ key: 'win_rate', group: 'results' }),
        ],
      }),
    )
    renderWithProviders(<TendancesTab />)
    const matrice = await screen.findByTestId('tendances-matrix')
    expect(screen.getByText('Indicateurs par mois et par horizon')).toBeInTheDocument()
    const groupes = within(matrice)
      .getAllByRole('heading', { level: 3 })
      .map((h) => h.textContent)
    expect(groupes).toEqual(['Résultats', 'Combat'])
    expect(
      await within(screen.getByTestId('tendances-matrix-results')).findByTestId('echarts-stub'),
    ).toBeInTheDocument()
  })

  it('grille : rampe divergente, 16 colonnes, libellé de champ, sans réglette, axe inversé', async () => {
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    const stub = await screen.findByTestId('echarts-stub')
    await waitFor(() => expect(stub.textContent).toContain('Taux de victoire'))
    const option = JSON.parse(stub.textContent ?? '{}')
    expect(option.series[0].type).toBe('heatmap')
    expect(option.series[0].data).toHaveLength(16)
    expect(option.visualMap.show).toBe(false)
    expect(option.visualMap.min).toBe(-2.5)
    expect(option.visualMap.max).toBe(2.5)
    expect(option.yAxis.inverse).toBe(true)
  })

  it('barre d’horizon : sous la matrice, 90 j pressé par défaut, un clic change l’horizon sans relire', async () => {
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    const barre = await screen.findByTestId('tendances-horizon-bar')
    const matrice = screen.getByTestId('tendances-matrix')
    expect(matrice.compareDocumentPosition(barre) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    const pas = screen.getByRole('group', { name: 'Pas de temps' })
    expect(within(barre).getByRole('button', { name: '90 j' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(pas).getByRole('button', { name: 'Par semaine' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )

    fireEvent.click(within(barre).getByRole('button', { name: '7 j' }))
    expect(within(barre).getByRole('button', { name: '7 j' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(pas).getByRole('button', { name: 'Par match' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(post).toHaveBeenCalledTimes(1)
  })

  it('évolution : sous la barre d’horizon ; le pas choisi revient au pas par défaut quand l’horizon change', async () => {
    post.mockResolvedValue(response({ indicators: [seriesIndicator('kda')] }))
    renderWithProviders(<TendancesTab />)
    const barre = await screen.findByTestId('tendances-horizon-bar')
    const evolution = screen.getByTestId('tendances-evolution')
    expect(barre.compareDocumentPosition(evolution) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(within(evolution).getByTestId('tendances-chart-kda')).toBeInTheDocument()

    const pas = screen.getByRole('group', { name: 'Pas de temps' })
    fireEvent.click(within(pas).getByRole('button', { name: 'Par mois' }))
    expect(within(pas).getByRole('button', { name: 'Par mois' })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(within(barre).getByRole('button', { name: '30 j' }))
    expect(within(pas).getByRole('button', { name: 'Par jour' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(pas).queryByRole('button', { name: 'Par mois' })).toBeNull()
    expect(post).toHaveBeenCalledTimes(1)
  })

  it('filtre : « Toutes les parties » puis une option par type (connu, manifeste, clé brute)', async () => {
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    // Les commandes sont à l'écran dès le chargement ; les options viennent de la réponse.
    await screen.findByTestId('tendances-matrix')
    const select = screen.getByLabelText('Type de partie') as HTMLSelectElement
    expect([...select.options].map((o) => o.textContent)).toEqual([
      'Toutes les parties',
      'Classé · Assassin',
      'Social · Assassin',
      'chaine_inconnue',
    ])
    expect(select.value).toBe('')
  })

  it('filtre : choisir un type refait la requête avec ce type, la matrice reste affichée', async () => {
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    const select = screen.getByLabelText('Type de partie')
    fireEvent.change(select, { target: { value: 'ranked_slayer' } })
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    expect(post.mock.calls[1][1]).toEqual({ view: 'solo', game_type: 'ranked_slayer', locale: 'fr' })
    expect(screen.getByTestId('tendances-matrix')).toBeInTheDocument()
    expect(screen.queryByTestId('tendances-loading')).toBeNull()
  })

  it('anglais : libellés de la page et du type de partie', async () => {
    useAppShellStore.setState({ locale: 'en' })
    post.mockResolvedValue(response())
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    const select = screen.getByLabelText('Game type') as HTMLSelectElement
    expect(select.options[0].textContent).toBe('All games')
    expect(screen.getByText('Indicators by month and by horizon')).toBeInTheDocument()
    expect(screen.getByText('Horizon')).toBeInTheDocument()
  })
})
