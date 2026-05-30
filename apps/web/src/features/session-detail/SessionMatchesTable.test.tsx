/**
 * Tests SessionMatchesTable — réutilisation du tableau Explorer via l'adapter
 * `toExplorerRow` + rendu (pastille Solo/Escouade, mapping des champs).
 *
 * Locale par défaut = fr (cf. ExplorerMatchesTable.test.tsx) → libellés français.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { SessionMatchesTable, toExplorerRow } from './SessionMatchesTable'

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
    mode_ui: 'Oddball',
    duration_seconds: 540,
    team_mmr: 1500,
    enemy_mmr: 1400,
    delta_mmr: 100,
    perf_tier: 2,
    skill_rating_type: 'csr',
    skill_rating_value: 1450,
  }
}

describe('toExplorerRow — adapter session → Explorer', () => {
  it('mappe les champs clés (escouade, map/mode/playlist, MMR, rating)', () => {
    const r = toExplorerRow(makeRow(), true)
    expect(r.match_id).toBe('m1')
    expect(r.outcome_code).toBe(2)
    expect(r.map_ui).toBe('Tir réel')
    expect(r.mode_ui).toBe('Oddball')
    expect(r.playlist_label).toBe('Ranked Arena')
    expect(r.is_with_friends).toBe(true) // escouade
    expect(r.kills).toBe(13)
    expect(r.kda).toBe(2.6)
    expect(r.team_mmr).toBe(1500)
    expect(r.delta_mmr).toBe(100)
    // Valeur de rating CSR (entier) placée dans la colonne Rang.
    expect(r.rating_type).toBe('CSR')
    expect(r.skill_tier_label).toBe('1450')
  })

  it('LUSR → 2 décimales ; outcome null → DNF(4) ; solo', () => {
    const row = { ...makeRow(), skill_rating_type: 'lusr', skill_rating_value: 1.234, outcome: null }
    const r = toExplorerRow(row, false)
    expect(r.rating_type).toBe('LUSR')
    expect(r.skill_tier_label).toBe('1.23')
    expect(r.outcome_code).toBe(4)
    expect(r.is_with_friends).toBe(false) // solo
  })
})

describe('SessionMatchesTable — rendu (réutilise ExplorerMatchesTable)', () => {
  it('rend le tableau Explorer + pastille Escouade + carte mappée', () => {
    renderWithProviders(
      <SessionMatchesTable matches={[makeRow()]} playerSlug="me" variant="full" withFriends />,
    )
    expect(screen.getByTestId('explorer-matches-table')).toBeInTheDocument()
    expect(screen.getByText('Escouade')).toBeInTheDocument() // colonne Solo/Escouade
    expect(screen.getByText('Tir réel')).toBeInTheDocument() // map_name → map_ui
  })

  it('solo → pastille Solo', () => {
    renderWithProviders(<SessionMatchesTable matches={[makeRow()]} playerSlug="me" />)
    expect(screen.getByText('Solo')).toBeInTheDocument()
  })

  it('liste vide → pas de tableau (état vide Explorer)', () => {
    renderWithProviders(<SessionMatchesTable matches={[]} playerSlug="me" />)
    expect(screen.queryByTestId('explorer-matches-table')).not.toBeInTheDocument()
  })
})
