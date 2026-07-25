/**
 * Tests ExplorerLiveStatusBadge (Lot A3 — fin de la dégradation muette) :
 *  - "ok" / undefined / null → aucun badge rendu (rien à signaler)
 *  - "no_auth" / "failed" / "local_partial" → badge discret avec le wording FR
 *    prescrit par le plan.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { ExplorerLiveStatusBadge } from './ExplorerLiveStatusBadge'

describe('ExplorerLiveStatusBadge', () => {
  it('ne rend rien pour le statut ok', () => {
    renderWithProviders(<ExplorerLiveStatusBadge status="ok" />)
    expect(screen.queryByTestId(/explorer-live-status-badge-/)).not.toBeInTheDocument()
  })

  it('ne rend rien quand le statut est absent (undefined)', () => {
    renderWithProviders(<ExplorerLiveStatusBadge />)
    expect(screen.queryByTestId(/explorer-live-status-badge-/)).not.toBeInTheDocument()
  })

  it('ne rend rien quand le statut est null', () => {
    renderWithProviders(<ExplorerLiveStatusBadge status={null} />)
    expect(screen.queryByTestId(/explorer-live-status-badge-/)).not.toBeInTheDocument()
  })

  it('affiche le wording "authentification" pour no_auth', () => {
    renderWithProviders(<ExplorerLiveStatusBadge status="no_auth" />)
    expect(screen.getByTestId('explorer-live-status-badge-no_auth')).toBeInTheDocument()
    expect(screen.getByText('Données live indisponibles (authentification)')).toBeInTheDocument()
  })

  it('affiche le wording "erreur" pour failed', () => {
    renderWithProviders(<ExplorerLiveStatusBadge status="failed" />)
    expect(screen.getByTestId('explorer-live-status-badge-failed')).toBeInTheDocument()
    expect(screen.getByText('Données live indisponibles (erreur)')).toBeInTheDocument()
  })

  it('affiche "Live partiel" pour local_partial', () => {
    renderWithProviders(<ExplorerLiveStatusBadge status="local_partial" />)
    expect(screen.getByTestId('explorer-live-status-badge-local_partial')).toBeInTheDocument()
    expect(screen.getByText('Live partiel')).toBeInTheDocument()
  })
})
