/**
 * SquadContributionsPage.test.tsx — 3 empty states + radar ECharts dégradé.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as useFieldMappingsModule from '@/lib/i18n/fieldMappings'
import * as squadContextModule from './SquadContext'
import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { SquadContributionsPage } from './SquadContributionsPage'

// Mock RadarChart : echarts-for-react absent en env portable — évite l'erreur
// de résolution Vite. Le stub expose data-series pour les assertions de contenu.
vi.mock('@/components/charts/RadarChart', () => ({
  RadarChart: ({
    series,
    axisLabels,
  }: {
    series: Array<{ key: string; axes: Array<{ axis: string }> }>
    axisLabels?: Record<string, string>
  }) => (
    <div
      data-testid="chart-card"
      data-axes={JSON.stringify(axisLabels)}
      data-series={JSON.stringify(series)}
    />
  ),
}))

const ROW = (gamertag: string): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 5,
  last_seen_at: null,
  with_kpis: {
    match_count: 5,
    wins: 3,
    kd_ratio: 1.5,
    win_rate: 0.6,
    accuracy: 0.45,
    kills_per_game: 12,
    assists_per_game: 4,
    headshot_kills_per_game: 3,
    perfect_kills_per_game: 1,
  },
  without_kpis: null,
})

const FULL_MAPPINGS: useFieldMappingsModule.FieldMappingsResponse = {
  title_slug: 'halo_infinite',
  schema_version: 1,
  locale: 'fr',
  fields: {
    kills: dto('Éliminations'),
    accuracy: dto('Précision'),
    kdr: dto('K/D'),
    win_rate: dto('Taux de victoire'),
    assists: dto('Assistances'),
  },
}

function dto(label: string): useFieldMappingsModule.FieldMappingDTO {
  return {
    label,
    storage_unit: 'count',
    display_unit: 'count',
    format: 'integer',
    display_order: 1,
    group: 'combat',
  }
}

function mockSquadContext({
  selectedRows,
  confirmedGamertags,
}: {
  selectedRows: TeammateRow[]
  confirmedGamertags: string[]
}) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows,
    confirmedGamertags,
    pageData: null as unknown as TeammatesPageResponse,
    playerSlug: 'test',
  })
}

function mockMappings(value: useFieldMappingsModule.FieldMappingsResponse | undefined) {
  vi.spyOn(useFieldMappingsModule, 'useFieldMappings').mockReturnValue({
    data: value,
    isLoading: false,
    isError: false,
    error: null,
    isSuccess: !!value,
    isPending: !value,
    isFetching: false,
    isStale: false,
    refetch: vi.fn(),
  } as unknown as ReturnType<typeof useFieldMappingsModule.useFieldMappings>)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SquadContributionsPage — empty states', () => {
  it('no_selection : aucun gamertag confirmé → wording analyse', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: [] })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByText(/Analyse de synergies/)).toBeInTheDocument()
    expect(screen.getByText(/Choisis 1 à 3 coéquipiers/)).toBeInTheDocument()
    expect(screen.queryByTestId('chart-card')).toBeNull()
  })

  it('invalid_selection : confirmedGts > 0 mais selectedRows vide', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: ['ghost'] })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByText(/Aucune donnée commune/)).toBeInTheDocument()
    expect(screen.queryByTestId('chart-card')).toBeNull()
  })

  it('rows présents : rend le radar ECharts', () => {
    mockSquadContext({
      selectedRows: [ROW('FriendA')],
      confirmedGamertags: ['FriendA'],
    })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('chart-card')).toBeInTheDocument()
    // Vérifie qu'il n'y a pas de résidu "Solo ref" dans les series.
    const card = screen.getByTestId('chart-card')
    const seriesJson = card.getAttribute('data-series') ?? ''
    expect(seriesJson).not.toMatch(/Solo ref/)
    expect(seriesJson).not.toMatch(/Référence solo/)
  })
})

describe('SquadContributionsPage — multi-titres', () => {
  it('synthetic_title_b minimaliste : ne crashe pas, axes filtrés', () => {
    const minimal: useFieldMappingsModule.FieldMappingsResponse = {
      title_slug: 'synthetic_title_b',
      schema_version: 1,
      locale: 'fr',
      fields: {
        kills: dto('Frags'),
        accuracy: dto('Taux de réussite'),
      },
    }
    mockSquadContext({
      selectedRows: [ROW('FriendA')],
      confirmedGamertags: ['FriendA'],
    })
    mockMappings(minimal)
    expect(() => renderWithProviders(<SquadContributionsPage />)).not.toThrow()
    expect(screen.getByTestId('chart-card')).toBeInTheDocument()
    // Sur un titre minimaliste, seuls kills et accuracy passent le filtre.
    const axesJson = screen.getByTestId('chart-card').getAttribute('data-axes') ?? ''
    expect(axesJson).toContain('Frags')
    expect(axesJson).not.toContain('K/D')
    expect(axesJson).not.toContain('Taux de victoire')
  })
})
