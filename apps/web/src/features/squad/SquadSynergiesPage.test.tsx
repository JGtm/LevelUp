/**
 * SquadSynergiesPage.test.tsx — états vides, titres de section et ratchet du lot 3.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { SquadSynergiesPage } from './SquadSynergiesPage'
import { echangeDe } from './squadRiposte.fixtures'
import { pageWithEquipmentUsage } from './squadUsages.fixtures'

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

/** Réponse de page minimale portant le seul bloc que le test regarde. */
function pageWithAssistPairs(): TeammatesPageResponse {
  return {
    options: [],
    teammates: [],
    total_matches: 2,
    session_labels: { solo: [], squad: [] },
    friends_count: 0,
    assist_pairs: {
      matches_measured: 2,
      matches_total: 2,
      total_assists: 3,
      pairs: [
        {
          assist_xuid: 'x1',
          assist_gamertag: 'Alpha',
          killer_xuid: 'x2',
          killer_gamertag: 'Bravo',
          assist_count: 3,
          stolen_count: 1,
        },
      ],
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

describe('SquadSynergiesPage — empty states', () => {
  it('no_selection : wording analyse, pas de contenu', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: [] })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Choisis 1 à 3 coéquipiers/)).toBeInTheDocument()
  })

  it('invalid_selection : message dédié', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: ['ghost'] })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Aucune donnée commune/)).toBeInTheDocument()
  })

  it('avec rows : rend sans erreur', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    expect(() => renderWithProviders(<SquadSynergiesPage />)).not.toThrow()
  })
})

// LOT 3 « sections » (2026-09-22) — UN SEUL AXE PAR ONGLET. Synergies ne dit plus que ce
// que la composition PRODUIT ensemble : ce qu'elle UTILISE (frags et armes, équipement,
// formes) est parti sur Usages, l'impact et les médailles sur Contributions. Ces
// assertions sont le RATCHET du déménagement : elles échouent si un bloc revient ici.
describe('SquadSynergiesPage — ce qui a déménagé (lot 3)', () => {
  it('ni frags et armes, ni équipement, ni impact, ni médailles', () => {
    setTitleCaps(['expected_stats'])
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithEquipmentUsage(),
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.queryByText('Écart cumulé au FDA attendu')).toBeNull()
    expect(screen.queryByText('Frags et armes')).toBeNull()
    expect(screen.queryByText("Usages d'équipement")).toBeNull()
    expect(screen.queryByText('Équipement et armes de socle')).toBeNull()
    expect(screen.queryByText('Impact des coéquipiers')).toBeNull()
    expect(screen.queryByText(/^Médailles/)).toBeNull()
  })
})

// Les deux titres de section de l'onglet (lot 3, fusionné avec la section « Coordination »
// de la vague 3) : « Coordination » coiffe la riposte, l'appui, l'isolement et la portée ;
// « Historique » la bande de résultats ET le tableau des matchs. Un titre coiffe au moins
// deux blocs, jamais un bloc seul : sans aucun de ces blocs, pas de titre.
describe('SquadSynergiesPage — titres de section', () => {
  it('« Historique » est toujours posé au-dessus de la bande et du tableau', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText('Historique')).toBeInTheDocument()
  })

  it('« L\'échange » se monte avec le bloc echange, et pas sans lui', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.queryByText('Coordination')).toBeNull()
  })

  it('« L\'échange » présent quand le bloc echange est mesuré', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: { ...pageWithAssistPairs(), echange: echangeDe() } as TeammatesPageResponse,
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText('Coordination')).toBeInTheDocument()
  })
})

// La description de la section Assistances était traduite (FR et EN) mais jamais montée :
// ni le titre ni les en-têtes de colonne ne disent ce que le tableau MESURE. Depuis le
// 2026-09-19 elle vit dans l'INFOBULLE du titre du bloc (décision 9) : elle n'est donc plus
// dans le DOM au repos, et c'est le TITRE qui doit répondre présent.
describe('SquadSynergiesPage — section Assistances', () => {
  it('monte le bloc titré quand les assistances sont mesurées', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
      pageData: pageWithAssistPairs(),
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getAllByText("Appui").length).toBeGreaterThan(0)
    // La phrase de lecture a quitté l'écran pour l'infobulle du titre.
    expect(screen.queryByText(/^Une barre par larbin/)).toBeNull()
  })

  it('bloc absent : aucun titre, aucune section vide', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.queryByText("Assistances dans l'escouade")).toBeNull()
  })
})
