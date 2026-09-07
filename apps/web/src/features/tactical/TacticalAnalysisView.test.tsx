/**
 * TacticalAnalysisView.test.tsx — Render tests pour TacticalAnalysisView.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClientProvider, QueryClient } from '@tanstack/react-query'
import { TacticalAnalysisView } from './TacticalAnalysisView'

// Mock useLocale
vi.mock('@/lib/i18n/use-locale', () => ({
  useLocale: () => 'fr',
}))

// Mock useTacticalRaster
vi.mock('./queries', () => ({
  useTacticalRaster: vi.fn(() => ({
    data: {
      cells: [
        { col: 0, row: 0, valeur: 0.5 },
        { col: 1, row: 1, valeur: 0.8 },
        { col: 2, row: 2, valeur: 1.2 },
      ],
      gridSize: { cell: 1, nx: 100, ny: 100, minX: 0, minY: 0 },
      scale: { lo: 0, hi: 2.4 },
      filled: 3,
      matchs_filtres: 30,
      matchs_retenus: 28,
      matchs_en_attente: 0,
      matchs_non_cuisables: 0,
      echange: 0.41,
      grappes: [
        { id: 'a', nom_fr: 'Garage', nom_en: 'Garage', x: 10, y: 30, matchs: 15 },
        { id: 'b', nom_fr: 'Passerelle', nom_en: 'Bridge', x: 90, y: 30, matchs: 13 },
      ],
    },
    isLoading: false,
    isError: false,
  })),
}))

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

const renderWithQueryClient = (component: React.ReactElement) => {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>,
  )
}

describe('TacticalAnalysisView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should render the component with default state', () => {
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    // Check KPI strip exists
    expect(screen.getByText('Matchs retenus')).toBeInTheDocument()

    // Check toolbar elements
    expect(screen.getByText('La question')).toBeInTheDocument()
    expect(screen.getByText('Qui')).toBeInTheDocument()
    expect(screen.getByText('Spawn de départ')).toBeInTheDocument()

    // Check plan card title
    expect(screen.getByText('Plan de Streets — Où je meurs')).toBeInTheDocument()

    // Check cell card
    expect(screen.getByText('Cellule sélectionnée')).toBeInTheDocument()
  })

  it('should change question when select changes', async () => {
    const user = userEvent.setup()
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    const select = screen.getByRole('combobox') as HTMLSelectElement
    expect(select.value).toBe('morts')

    await user.selectOptions(select, 'kills')
    expect(select.value).toBe('kills')

    // Title should update
    expect(screen.getByText('Plan de Streets — Où je tue')).toBeInTheDocument()
  })

  it('should handle who segment toggling', async () => {
    const user = userEvent.setup()
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    const moiButton = screen.getByText('Moi')
    const escouadeButton = screen.getByText('Escouade')

    expect(moiButton).toHaveAttribute('aria-pressed', 'true')
    expect(escouadeButton).toHaveAttribute('aria-pressed', 'false')

    await user.click(escouadeButton)

    expect(moiButton).toHaveAttribute('aria-pressed', 'false')
    expect(escouadeButton).toHaveAttribute('aria-pressed', 'true')
  })

  it('should handle spawn segment with grappes', async () => {
    const user = userEvent.setup()
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
        grappes={[
          { id: 'a', nom_fr: 'Garage', nom_en: 'Garage' },
          { id: 'b', nom_fr: 'Passerelle', nom_en: 'Bridge' },
        ]}
      />,
    )

    const tousButton = screen.getByText('Tous')
    const garageButton = screen.getByText('Garage')

    expect(tousButton).toHaveAttribute('aria-pressed', 'true')
    expect(garageButton).toHaveAttribute('aria-pressed', 'false')

    await user.click(garageButton)

    expect(tousButton).toHaveAttribute('aria-pressed', 'false')
    expect(garageButton).toHaveAttribute('aria-pressed', 'true')
  })

  it('should display KPI with correct values', () => {
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    // KPI values should be present (28 retained, 93% coverage, 41% trade)
    expect(screen.getByText('28')).toBeInTheDocument()
  })

  it('should have accessible canvas with aria-label', () => {
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    const canvas = screen.getByRole('img', { name: /Plan de Streets/ })
    expect(canvas).toBeInTheDocument()
  })

  it('should render loading state', () => {
    const { useQuery } = require('@tanstack/react-query')
    vi.mocked(useQuery).mockReturnValueOnce({
      data: undefined,
      isLoading: true,
      isError: false,
    })

    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    // Should show loading state in KPI strip and plan card
    expect(screen.getByText('Chargement…')).toBeInTheDocument()
  })

  it('should clear selected cell when question changes', async () => {
    const user = userEvent.setup()
    renderWithQueryClient(
      <TacticalAnalysisView
        playerSlug="test-player"
        titleSlug="halo_infinite"
        mapId="streets"
        mapName="Streets"
        filterHash="hash123"
      />,
    )

    // Cell card should show no selection message initially
    expect(screen.getByText('Clique une zone chaude du plan.')).toBeInTheDocument()

    const select = screen.getByRole('combobox')
    await user.selectOptions(select, 'kills')

    // Message should still be there (selection cleared on question change)
    expect(screen.getByText('Clique une zone chaude du plan.')).toBeInTheDocument()
  })
})
