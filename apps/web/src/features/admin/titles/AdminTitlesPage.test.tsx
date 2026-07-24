import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

// i18n : renvoie la clé brute (suffit pour vérifier le rendu structurel).
vi.mock('../useAdminText', () => ({
  useAdminT: () => (key: string) => key,
}))

// Données mockées : un titre Halo Infinite avec capabilities + feature-matrix.
vi.mock('./queries', () => ({
  useAdminTitles: () => ({
    data: {
      titles: [
        {
          slug: 'halo_infinite',
          name: 'Halo Infinite',
          provider: 'halo_infinite',
          icon_url: '',
          status: 'active',
          capabilities: ['ranked', 'career'],
          is_default: true,
          xbox_title_id: '2043073184',
          steam_app_id: '1336960',
          has_mappings: true,
        },
      ],
      count: 1,
    },
    isLoading: false,
    isError: false,
  }),
  useAdminTitleDetail: () => ({
    data: {
      slug: 'halo_infinite',
      name: 'Halo Infinite',
      provider: 'halo_infinite',
      icon_url: '',
      status: 'active',
      capabilities: ['ranked'],
      is_default: true,
      xbox_title_id: '2043073184',
      steam_app_id: '1336960',
      has_mappings: true,
      schema_version: 1,
      declared_capabilities: { 'match.history': 'supported' },
      feature_matrix: { 'match_view.cadence': 'available' },
    },
    isLoading: false,
    isError: false,
  }),
  useAdminTitleDiagnostic: () => ({
    data: {
      title_slug: 'halo_infinite',
      config_files: [{ name: 'fields.toml', present: true, required: true }],
      databases: [
        {
          name: 'metadata.duckdb',
          exists: true,
          tables: [{ name: 'season_calendars', exists: true, rows: 5 }],
        },
      ],
      drifts: [
        {
          feature: 'match_history',
          kind: 'data',
          computed: 'available',
          reason: 'feature déclarée disponible mais match_registry absente/vide',
        },
      ],
    },
    isLoading: false,
    isError: false,
  }),
  fetchTitleTomlDraft: () => Promise.resolve('DRAFT_TOML'),
}))

import { AdminTitlesPage } from './AdminTitlesPage'

describe('AdminTitlesPage', () => {
  it('rend la liste des titres + le détail du titre sélectionné', () => {
    render(<AdminTitlesPage />)
    // Le nom du titre apparaît (ligne + carte détail).
    expect(screen.getAllByText('Halo Infinite').length).toBeGreaterThan(0)
    // Détail : capability déclarée + feature (uniques → getByText sûr).
    expect(screen.getByText('match.history')).toBeTruthy()
    expect(screen.getByText('match_view.cadence')).toBeTruthy()
    // Diagnostic : fichier de config + table DB.
    expect(screen.getByText('fields.toml')).toBeTruthy()
    expect(screen.getByText('season_calendars')).toBeTruthy()
    // Notice d'aide « enregistrer un 2e titre ».
    expect(screen.getByText('admin.titles.register_help_title')).toBeTruthy()
    // Drift déclaré-vs-réalité : la feature en écart est rendue.
    expect(screen.getByText('match_history')).toBeTruthy()
  })

  it('en-tête « col_title » triable (I16) : clic ne casse pas le rendu', () => {
    render(<AdminTitlesPage />)
    const header = screen.getByText('admin.titles.col_title').closest('th') as HTMLElement
    expect(header.querySelector('button')).toBeInTheDocument()
    fireEvent.click(header)
    expect(screen.getAllByText('Halo Infinite').length).toBeGreaterThan(0)
  })

  it('copie le brouillon TOML dans le presse-papier (D10)', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    Object.assign(navigator, { clipboard: { writeText } })

    render(<AdminTitlesPage />)
    fireEvent.click(screen.getByText('admin.titles.copy_draft'))

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('DRAFT_TOML'))
    // Feedback visuel après copie.
    expect(screen.getByText('admin.titles.copy_draft_done')).toBeTruthy()
  })
})
