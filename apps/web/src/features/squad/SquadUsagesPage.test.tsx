/**
 * SquadUsagesPage.test.tsx — l'onglet Usages (lot 3 « sections », 2026-09-22).
 *
 * Couvre : les deux états vides hérités de Synergies (no_selection /
 * invalid_selection), l'état vide PROPRE à l'onglet (aucun film décodé : ni frags,
 * ni équipement, ni formes — jamais un onglet vide et muet), le montage des sections
 * déplacées, et le gate `expected_stats` du card « Écart cumulé au FDA attendu » qui a
 * suivi la section frags.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { SquadUsagesPage } from './SquadUsagesPage'
import { pageWithEquipmentUsage, pageWithoutAnyUsage } from './squadUsages.fixtures'

const ROW = (gamertag: string): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 5,
  last_seen_at: undefined,
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
  without_kpis: undefined,
})

function mockSquadContext(opts: {
  selectedRows: TeammateRow[]
  confirmedGamertags: string[]
  pageData?: TeammatesPageResponse | null
}) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: opts.selectedRows,
    confirmedGamertags: opts.confirmedGamertags,
    pageData: opts.pageData ?? null,
    playerSlug: 'test',
    currentPlayerXuid: '',
  })
}

/** Réponse de page portant des frags par classe — la section « Frags et armes » se monte. */
function pageWithFrags(): TeammatesPageResponse {
  return {
    ...pageWithoutAnyUsage(),
    main_player: 'A',
    frag_classes: {
      A: [{ authoritative: true, class: 'gun', kills: 42 }],
      B: [{ authoritative: true, class: 'melee', kills: 7 }],
    },
  } as TeammatesPageResponse
}

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
    ],
  })
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('SquadUsagesPage — états vides', () => {
  it('no_selection : même wording que Synergies', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: [] })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getByText(/Choisis 1 à 3 coéquipiers/)).toBeInTheDocument()
  })

  it('invalid_selection : message dédié', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: ['ghost'] })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getByText(/Aucune donnée commune/)).toBeInTheDocument()
  })

  it('aucun bloc à montrer : l\'onglet le DIT (aucun film décodé), il ne reste pas muet', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithoutAnyUsage(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getByText('Aucun film décodé pour cette sélection.')).toBeInTheDocument()
    expect(screen.queryByText('Frags et armes')).toBeNull()
  })
})

describe('SquadUsagesPage — sections', () => {
  it('« Frags et armes » coiffe la section frags quand les classes sont mesurées', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithFrags(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getByText('Frags et armes')).toBeInTheDocument()
  })

  it('l\'équipement est monté ici (une ligne par coéquipier suivi, jamais par famille)', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithEquipmentUsage(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getAllByText("Usages d'équipement").length).toBeGreaterThan(0)
    expect(screen.getAllByText('Madina').length).toBeGreaterThan(0)
    expect(screen.getByText('88 pris')).toBeInTheDocument()
  })

  it('équipement absent : ni titre de section, ni cadre vide', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithFrags(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.queryByText("Usages d'équipement")).toBeNull()
  })
})

describe('SquadUsagesPage — gate expected_stats du card FDA', () => {
  it('capability présente → « Écart cumulé au FDA attendu » à gauche de Répartition', () => {
    setTitleCaps(['expected_stats'])
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithFrags(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.getByText('Écart cumulé au FDA attendu')).toBeInTheDocument()
  })

  it('capability absente (Halo 5) → card Écart FDA masqué', () => {
    setTitleCaps(['ranked'])
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithFrags(),
    })
    renderWithProviders(<SquadUsagesPage />)
    expect(screen.queryByText('Écart cumulé au FDA attendu')).toBeNull()
  })
})
