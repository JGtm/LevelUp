/**
 * Tests ExplorerTargetSeasonCSR — liste CSR compacte (playlist + tier + rating).
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { CareerPlaylistCSR } from '@/lib/api/types'

import { ExplorerTargetSeasonCSR } from './ExplorerTargetSeasonCSR'

function csr(over: Partial<CareerPlaylistCSR> & { playlist_id: string }): CareerPlaylistCSR {
  const empty = { value: 0, tier: '', sub_tier: 0, measurement_matches_remaining: 0, placement_total: 0 }
  return {
    playlist_id: over.playlist_id,
    playlist_name: over.playlist_name ?? '',
    queue: '',
    input: '',
    current: { ...empty, ...(over.current ?? {}) },
    season: { ...empty },
    all_time: { ...empty },
  }
}

describe('ExplorerTargetSeasonCSR', () => {
  it('rend nom de playlist + tier (FR), sans la valeur de rating', () => {
    const csrs = [
      csr({ playlist_id: 'p1', playlist_name: 'Ranked Arena', current: { value: 1523, tier: 'Diamond', sub_tier: 3, measurement_matches_remaining: 0, placement_total: 0 } }),
      csr({ playlist_id: 'p2', playlist_name: 'Ranked Slayer', current: { value: 1810, tier: 'Onyx', sub_tier: 0, measurement_matches_remaining: 0, placement_total: 0 } }),
    ]
    renderWithProviders(<ExplorerTargetSeasonCSR csrs={csrs} title="CSR" emptyMessage="Aucun classement" />)
    expect(screen.getByText('Ranked Arena')).toBeInTheDocument()
    // Tier traduit en FR (locale par défaut = fr) + sous-palier en romain.
    expect(screen.getByText('Diamant III')).toBeInTheDocument()
    // Onyx sans sub-tier (inchangé en FR)
    expect(screen.getByText('Onyx')).toBeInTheDocument()
    // La valeur de rating brute n'est plus affichée.
    expect(screen.queryByText('1523')).not.toBeInTheDocument()
    expect(screen.queryByText('1810')).not.toBeInTheDocument()
  })

  it('rend un placeholder titré (jamais masqué) si liste vide', () => {
    renderWithProviders(
      <ExplorerTargetSeasonCSR csrs={[]} title="CSR" emptyMessage="Aucun classement CSR à afficher" />,
    )
    // Carte titrée + message centré rendus (visibilité des manques), plus de null.
    expect(screen.getByTestId('explorer-target-season-csr')).toBeInTheDocument()
    expect(screen.getByText('CSR')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-target-season-csr-empty')).toBeInTheDocument()
    expect(screen.getByText('Aucun classement CSR à afficher')).toBeInTheDocument()
  })

  // Lot A3 (fin de la dégradation muette) : badge discret dans la barre de titre
  // quand liveStatus explique un vide/partiel — sans lui (status="ok"/absent),
  // aucun badge (cf. tests ci-dessus, pas de régression).
  it('affiche le badge live_status quand no_auth (liste vide)', () => {
    renderWithProviders(
      <ExplorerTargetSeasonCSR
        csrs={[]}
        title="CSR"
        emptyMessage="Aucun classement CSR à afficher"
        liveStatus="no_auth"
      />,
    )
    expect(screen.getByTestId('explorer-live-status-badge-no_auth')).toBeInTheDocument()
    expect(screen.getByText('Données live indisponibles (authentification)')).toBeInTheDocument()
  })

  it("n'affiche aucun badge quand liveStatus vaut ok (liste non vide)", () => {
    const csrs = [csr({ playlist_id: 'p1', playlist_name: 'Ranked Arena' })]
    renderWithProviders(
      <ExplorerTargetSeasonCSR csrs={csrs} title="CSR" emptyMessage="Aucun classement" liveStatus="ok" />,
    )
    expect(screen.queryByTestId(/explorer-live-status-badge-/)).not.toBeInTheDocument()
  })
})
