/**
 * Tests SessionMatchesTable — couverture des presets de colonnes.
 *
 * variant="full"    : colonnes riches (Carte, Playlist, Précision, Durée, Rating,
 *                     ΔMMR) + icône d'ouverture.
 * variant="compact" : set réduit 7 col (Issue·Mode·K/D/A·KDA·Perf·Rating·ΔMMR) —
 *                     pas d'Ouvrir/Heure/Carte/Playlist/Précision/Durée.
 *
 * Locale par défaut = fr (cf. ExplorerMatchesTable.test.tsx) → libellés français.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { SessionMatchesTable } from './SessionMatchesTable'

function makeRow(): SessionDetailMatchRow {
  return {
    match_id: 'm1',
    start_time: '2026-05-26T20:00:00Z',
    outcome: 2,
    playlist_name: 'Ranked Arena',
    pair_name: 'Oddball',
    is_ranked: true,
    kills: 13,
    deaths: 5,
    assists: 6,
    kda: 2.6,
    accuracy: 55,
    personal_score: 2400,
    performance_score: 72,
    session_label: 'S1',
    dominant_category: 'Ranked',
    offensive_conversion: null,
    defensive_resistance: null,
    map_name: 'Tir réel',
    duration_seconds: 540,
    team_mmr: 1500,
    enemy_mmr: 1400,
    delta_mmr: 100,
    perf_tier: 2,
    skill_rating_type: 'csr',
    skill_rating_value: 1450,
  }
}

describe('SessionMatchesTable — presets de colonnes', () => {
  it('variant="full" affiche les colonnes riches + valeurs', () => {
    renderWithProviders(<SessionMatchesTable matches={[makeRow()]} playerSlug="me" variant="full" />)

    // En-têtes riches (full uniquement).
    expect(screen.getByText('Carte')).toBeInTheDocument()
    expect(screen.getByText('Durée')).toBeInTheDocument()
    expect(screen.getByText('ΔMMR')).toBeInTheDocument()
    expect(screen.getByText('Rating')).toBeInTheDocument()
    // Cellules enrichies (indépendantes des field-mappings).
    expect(screen.getByText('Tir réel')).toBeInTheDocument() // map_name
    expect(screen.getByText('1450 CSR')).toBeInTheDocument() // rating_value + type
    expect(screen.getByText('+100')).toBeInTheDocument() // delta_mmr
    // Icône d'ouverture présente en full.
    expect(screen.getByRole('button', { name: 'Ouvrir' })).toBeInTheDocument()
  })

  it('variant="compact" masque Ouvrir/Heure/Carte/Playlist/Précision/Durée', () => {
    renderWithProviders(<SessionMatchesTable matches={[makeRow()]} playerSlug="me" variant="compact" />)

    // Colonnes retirées du preset compact.
    expect(screen.queryByText('Carte')).not.toBeInTheDocument()
    expect(screen.queryByText('Heure')).not.toBeInTheDocument()
    expect(screen.queryByText('Durée')).not.toBeInTheDocument()
    expect(screen.queryByText('Playlist')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Ouvrir' })).not.toBeInTheDocument()
    // Colonnes conservées (set réduit 7 col).
    expect(screen.getByText('Mode')).toBeInTheDocument()
    expect(screen.getByText('ΔMMR')).toBeInTheDocument()
    expect(screen.getByText('Rating')).toBeInTheDocument()
    expect(screen.getByText('1450 CSR')).toBeInTheDocument()
  })

  it('liste vide → état vide explicite', () => {
    renderWithProviders(<SessionMatchesTable matches={[]} playerSlug="me" />)
    expect(screen.queryByTestId('session-matches-table')).not.toBeInTheDocument()
    expect(screen.getByText('Aucun match dans cette session')).toBeInTheDocument()
  })
})
