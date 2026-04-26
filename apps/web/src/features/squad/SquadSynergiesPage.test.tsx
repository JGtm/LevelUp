/**
 * SquadSynergiesPage.test.tsx — 3 empty states + bar chart synergies
 * + dégradation gracieuse multi-titres.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as useFieldMappingsModule from '@/lib/i18n/fieldMappings'
import * as squadLayoutModule from './SquadLayout'
import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { SquadSynergiesPage } from './SquadSynergiesPage'

vi.mock('@/components/ui/plotly-chart', () => ({
  PlotlyChart: ({ figure }: { figure: unknown }) => (
    <div data-testid="plotly-chart" data-fig={JSON.stringify(figure)} />
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
    headshot_kills: dto('Tirs à la tête'),
  },
}

function mockSquadContext(opts: {
  selectedRows: TeammateRow[]
  confirmedGamertags: string[]
  pageData?: TeammatesPageResponse | null
}) {
  vi.spyOn(squadLayoutModule, 'useSquadContext').mockReturnValue({
    selectedRows: opts.selectedRows,
    confirmedGamertags: opts.confirmedGamertags,
    pageData: opts.pageData ?? null,
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

describe('SquadSynergiesPage — empty states', () => {
  it('no_selection : wording analyse, pas de Plotly', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: [] })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Choisis 1 à 3 coéquipiers/)).toBeInTheDocument()
    expect(screen.queryByTestId('plotly-chart')).toBeNull()
  })

  it('invalid_selection : message dédié', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: ['ghost'] })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Aucune donnée commune/)).toBeInTheDocument()
  })

  it('avec rows : rend le bar chart, plus de "Référence solo"', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadSynergiesPage />)
    const charts = screen.getAllByTestId('plotly-chart')
    // Au moins le bar chart synergies — les autres optionnels sont null.
    expect(charts.length).toBeGreaterThanOrEqual(1)
    const json = charts[0].getAttribute('data-fig')!
    expect(json).not.toMatch(/Référence solo/)
    expect(json).not.toMatch(/Solo ref/)
    expect(json).toContain('Avec A')
    expect(json).toContain('Avec B')
  })
})

describe('SquadSynergiesPage — locale EN', () => {
  it('compose les noms de traces avec "With" et l\'unité /game', () => {
    useAppShellStore.setState({ locale: 'en' })
    mockSquadContext({
      selectedRows: [ROW('A')],
      confirmedGamertags: ['A'],
    })
    mockMappings(FULL_MAPPINGS)
    renderWithProviders(<SquadSynergiesPage />)
    const json = screen.getAllByTestId('plotly-chart')[0].getAttribute('data-fig')!
    expect(json).toContain('With A')
    expect(json).toContain('/game')
    expect(json).not.toContain('Avec A')
    expect(json).not.toContain('/partie')
  })
})

describe('SquadSynergiesPage — multi-titres', () => {
  it('synthetic_title_b : ne crashe pas, axes filtrés', () => {
    const minimal: useFieldMappingsModule.FieldMappingsResponse = {
      title_slug: 'synthetic_title_b',
      schema_version: 1,
      locale: 'fr',
      fields: {
        kills: dto('Frags'),
      },
    }
    mockSquadContext({
      selectedRows: [ROW('A')],
      confirmedGamertags: ['A'],
    })
    mockMappings(minimal)
    expect(() => renderWithProviders(<SquadSynergiesPage />)).not.toThrow()
    const json = screen.getAllByTestId('plotly-chart')[0].getAttribute('data-fig')!
    // Seul kills est dans le mapping, donc seul axe `Frags/partie` reste.
    expect(json).toContain('Frags')
    expect(json).not.toContain('Taux de victoire')
    expect(json).not.toContain('K/D')
    expect(json).not.toContain('Assistances')
  })
})
