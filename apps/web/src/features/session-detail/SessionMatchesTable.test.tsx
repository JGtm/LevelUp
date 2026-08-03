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
    offensive_conversion: undefined,
    defensive_resistance: undefined,
    map_name: 'Tir réel',
    mode_ui: 'Oddball',
    duration_seconds: 540,
    team_mmr: 1500,
    enemy_mmr: 1400,
    delta_mmr: 100,
    perf_tier: 2,
    skill_rating_type: 'csr',
    skill_rating_value: 1450,
    skill_rating_delta: 25,
    skill_tier_label: 'Or III',
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
    // Colonne Rang = libellé du PALIER (comme l'Explorer), pas la valeur brute.
    expect(r.rating_type).toBe('CSR')
    expect(r.skill_tier_label).toBe('Or III')
    expect(r.skill_rating_delta).toBe(25) // porteur pour la colonne Δ rang injectée
  })

  it('palier passthrough ; outcome null → DNF(4) ; solo', () => {
    const row = { ...makeRow(), skill_rating_type: 'lusr', skill_tier_label: 'Diamant V', outcome: undefined }
    const r = toExplorerRow(row, false)
    expect(r.rating_type).toBe('LUSR')
    expect(r.skill_tier_label).toBe('Diamant V') // libellé fourni par le backend, tel quel
    expect(r.outcome_code).toBe(4)
    expect(r.is_with_friends).toBe(false) // solo
  })

  // V73-L2 2.5 : le score personnel avait été injecté dans `score_label` (colonne
  // « Score », qui porte le score d'ÉQUIPE sur l'Explorer) — même en-tête, deux
  // sens selon la page. Il a maintenant son propre champ ; `score_label` reste
  // vide côté session, faute de score d'équipe dans le contrat.
  it('score personnel dans personal_score, PAS dans le score d’équipe', () => {
    const r = toExplorerRow(makeRow(), true)
    expect(r.personal_score).toBe(2400)
    expect(r.score_label).toBe('')
  })

  it('score personnel absent → null (pas de 0 trompeur)', () => {
    const r = toExplorerRow({ ...makeRow(), personal_score: undefined }, true)
    expect(r.personal_score).toBeNull()
  })

  it('placement (placement_done/total) mappé', () => {
    const r = toExplorerRow({ ...makeRow(), placement_done: 3, placement_total: 5 }, false)
    expect(r.placement_done).toBe(3)
    expect(r.placement_total).toBe(5)
  })

  it('placement de chaîne perf (perf_placement_done/total) mappé indépendamment (V72-34)', () => {
    const r = toExplorerRow(
      // placement_* absents (schéma généré : number | undefined) = pas de placement
      // de classement ; seul le signal perf est posé — c'est le cas JGtm.
      { ...makeRow(), placement_done: undefined, placement_total: undefined, perf_placement_done: 8, perf_placement_total: 10 },
      false,
    )
    expect(r.perf_placement_done).toBe(8)
    expect(r.perf_placement_total).toBe(10)
    expect(r.placement_done).toBeNull()
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

  it('variant="compact" (drawer 50%) → set de colonnes réduit', () => {
    renderWithProviders(
      <SessionMatchesTable matches={[makeRow()]} playerSlug="me" variant="compact" withFriends />,
    )
    expect(screen.getByTestId('explorer-matches-table')).toBeInTheDocument()
    // Colonnes GARDÉES : Mode + Rang (palier) + Δ rang (injectée) restent visibles.
    expect(screen.getByText('Oddball')).toBeInTheDocument() // mode_ui
    expect(screen.getByText('Or III')).toBeInTheDocument() // skill_tier_label = palier (comme l'Explorer)
    expect(screen.getByText('+25')).toBeInTheDocument() // Δ rang injecté (CSR entier signé)
    // Colonnes MASQUÉES en compact : Solo/Escouade, carte, playlist.
    expect(screen.queryByText('Escouade')).not.toBeInTheDocument() // is_with_friends masqué
    expect(screen.queryByText('Tir réel')).not.toBeInTheDocument() // map_ui masqué
    expect(screen.queryByText('Ranked Arena')).not.toBeInTheDocument() // playlist_label masqué
  })

  it('variant="full" → toutes les colonnes (carte + playlist visibles)', () => {
    renderWithProviders(
      <SessionMatchesTable matches={[makeRow()]} playerSlug="me" variant="full" withFriends />,
    )
    // Contrôle : en full, les colonnes masquées en compact sont bien présentes.
    expect(screen.getByText('Tir réel')).toBeInTheDocument()
    expect(screen.getByText('Ranked Arena')).toBeInTheDocument()
    expect(screen.getByText('Escouade')).toBeInTheDocument()
    // Δ rang injecté : présent aussi en vue session pleine (comme l'ancien preset).
    expect(screen.getByText('+25')).toBeInTheDocument()
  })

  it('match en placement → colonne Rang affiche "X/Y" (prime sur le palier)', () => {
    const row = { ...makeRow(), placement_done: 3, placement_total: 5 }
    renderWithProviders(<SessionMatchesTable matches={[row]} playerSlug="me" variant="full" />)
    expect(screen.getByText('3/5')).toBeInTheDocument()
    expect(screen.queryByText('Or III')).not.toBeInTheDocument() // placement prime sur le palier
  })

})

// V72-09b : la colonne « Ouvrir sur Halo Waypoint » (I19) est un simple passage
// à travers ExplorerMatchesTable (aucune duplication) — mais AUCUN test existant
// ne couvrait ce consommateur précis (seuls ExplorerMatchesTable.test.tsx et
// SquadSynergyHistoryTable.test.tsx l'assertaient). Couverture manquante = seul
// filet qui aurait détecté une régression de gating sur cette page spécifique.
describe('SessionMatchesTable — colonne « Ouvrir sur Halo Waypoint » (I19, V72-09b)', () => {
  const WAYPOINT_LABEL = 'Ouvrir sur Halo Waypoint'

  it('vue full : lien Waypoint présent avec un href valide (capability fail-open + pref ON par défaut)', () => {
    renderWithProviders(
      <SessionMatchesTable matches={[makeRow()]} playerSlug="Chocoboflor" variant="full" />,
    )
    const link = screen.getByRole('link', { name: WAYPOINT_LABEL })
    expect(link).toHaveAttribute(
      'href',
      'https://www.halowaypoint.com/halo-infinite/players/Chocoboflor/matches/m1',
    )
  })

  it('vue compact (drawer) : colonne Waypoint masquée par COMPACT_HIDDEN_COLUMNS', () => {
    renderWithProviders(
      <SessionMatchesTable matches={[makeRow()]} playerSlug="Chocoboflor" variant="compact" />,
    )
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })
})
