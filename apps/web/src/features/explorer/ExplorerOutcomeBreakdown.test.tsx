/**
 * Tests ExplorerOutcomeBreakdown — « Répartition des résultats » de l'encart cible,
 * rendu 2.A horizontal (D17) :
 *  - une piste épaisse empilée V/N/D, compte ET part écrits dans les segments qui tiennent ;
 *  - le taux de victoire en chiffre d'appel ;
 *  - la bande des résultats alimentée par `common_matches`, du plus ANCIEN au plus récent
 *    (l'API sert récent→ancien) ;
 *  - sans matchs communs : barre et taux servis quand même, sans bande.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerCommonMatchRow } from '@/lib/api/types'

import { ExplorerOutcomeBreakdown } from './ExplorerOutcomeBreakdown'

function row(id: string, outcome: 'win' | 'loss' | 'tie', start: string): ExplorerCommonMatchRow {
  return {
    match_id: id,
    start_time: start,
    map_ui: 'Aquarius',
    mode_ui: 'Slayer',
    were_teammates: true,
    player_outcome: outcome === 'win' ? 2 : outcome === 'loss' ? 3 : 1,
    outcome,
    kills: 10,
    deaths: 8,
    kda: 1.2,
  }
}

// API : récent → ancien.
const MATCHES = [
  row('m3', 'loss', '2026-09-03T10:00:00Z'),
  row('m2', 'tie', '2026-09-02T10:00:00Z'),
  row('m1', 'win', '2026-09-01T10:00:00Z'),
]

describe('ExplorerOutcomeBreakdown', () => {
  it('rend une piste empilée dont chaque segment porte son compte et sa part', () => {
    renderWithProviders(
      <ExplorerOutcomeBreakdown wins={8} draws={1} losses={5} winRate={0.571} locale="fr" />,
    )
    expect(screen.getByTestId('explorer-outcome-track')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-outcome-track-win')).toHaveTextContent('8 · 57 %')
    expect(screen.getByTestId('explorer-outcome-track-loss')).toHaveTextContent('5 · 36 %')
    // Le nul (7 %) ne tient pas son écriture, mais existe bien et porte son infobulle.
    expect(screen.getByTestId('explorer-outcome-track-draw')).toBeInTheDocument()
    // …et son compte est repris en légende.
    expect(screen.getByTestId('explorer-outcome-breakdown')).toHaveTextContent('1 Nuls')
  })

  it('le taux de victoire est le chiffre d’appel', () => {
    renderWithProviders(
      <ExplorerOutcomeBreakdown wins={8} draws={1} losses={5} winRate={0.571} locale="fr" />,
    )
    expect(screen.getByText('57,1 %')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-outcome-breakdown')).toHaveTextContent('de victoires')
  })

  it('la bande des résultats suit l’ordre ancien → récent', () => {
    renderWithProviders(
      <ExplorerOutcomeBreakdown
        wins={1}
        draws={1}
        losses={1}
        winRate={0.333}
        commonMatches={MATCHES}
        locale="fr"
      />,
    )
    const tape = screen.getByTestId('explorer-outcome-tape')
    expect(tape).toBeInTheDocument()
    expect(tape).toHaveTextContent('Les 3 résultats, du plus ancien au plus récent')
  })

  it('sans matchs communs : barre et taux, pas de bande', () => {
    renderWithProviders(
      <ExplorerOutcomeBreakdown wins={2} draws={0} losses={1} winRate={0.667} locale="fr" />,
    )
    expect(screen.getByTestId('explorer-outcome-track')).toBeInTheDocument()
    expect(screen.queryByTestId('explorer-outcome-tape')).not.toBeInTheDocument()
  })

  it('aucun résultat : rien du tout', () => {
    const { container } = renderWithProviders(
      <ExplorerOutcomeBreakdown wins={0} draws={0} losses={0} locale="fr" />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
