/**
 * TacticalCellCard — la carte « Cellule sélectionnée » (item 5.6, complétée par le lot M1
 * — lien « voir dans le rejeu »).
 *
 * Ce que ces tests cadenassent :
 *   - aucune cellule sélectionnée -> le placeholder, aucune section de contributions ;
 *   - chargement des contributions -> le message d'attente, pas de liste ;
 *   - contributions vides -> le message vide, jamais le placeholder de la cellule (la
 *     valeur agrégée reste affichée) ;
 *   - NOMINAL -> chaque contribution est un lien `?frame=` construit par `instantToFrame`,
 *     vers la route du rejeu du bon match ;
 *   - `matchsNonOuvrables` -> le pied de liste seulement quand il est strictement positif
 *     (0 => rien, jamais un zéro qui suggérerait une absence de restriction).
 *
 * `useRouter`/`useTitleSlug` sont MOQUÉS : ce composant ne teste pas le routage lui-même
 * (couvert par les tests de route du rejeu), seulement que `TacticalCellCard` construit le
 * bon `href`.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalCellCard } from './TacticalCellCard'

vi.mock('@tanstack/react-router', () => ({
  useRouter: () => ({
    buildLocation: ({ params }: { params: { titleSlug: string; playerSlug: string; matchId: string } }) => ({
      href: `/${params.titleSlug}/players/${params.playerSlug}/matches/${params.matchId}/replay`,
    }),
  }),
}))

vi.mock('@/lib/title-routing', () => ({
  useTitleSlug: () => 'halo_infinite',
}))

const t = getTacticalText('fr')

const CELLULE: CelluleTactique = {
  col: 4,
  lig: 6,
  valeur: 3,
  brut: 3,
  centre_x: 12,
  centre_y: 18,
  matchs: 3,
  matchs_victoire: 1,
  matchs_defaite: 2,
}

function contribution(overrides: Partial<TacticalContribution> = {}): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 4200,
    xuid: '2533274000000001',
    match_started_at: '2026-09-01T12:00:00Z',
    ...overrides,
  }
}

function renderCard(props: Partial<Parameters<typeof TacticalCellCard>[0]> = {}) {
  return renderWithProviders(
    <TacticalCellCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      question="morts"
      cellule={CELLULE}
      contributions={null}
      contributionsLoading={false}
      matchsNonOuvrables={0}
      {...props}
    />,
  )
}

describe('TacticalCellCard — placeholder tant qu’aucune cellule n’est sélectionnée', () => {
  it('affiche le placeholder, aucune section de contributions', () => {
    renderCard({ cellule: null })
    expect(screen.getByText(t.cellPlaceholder)).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-cell-contributions')).not.toBeInTheDocument()
  })
})

describe('TacticalCellCard — chargement des contributions', () => {
  it('affiche le message d’attente, aucun lien', () => {
    renderCard({ contributions: null, contributionsLoading: true })
    expect(screen.getByText(t.cellContributionsLoading)).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-cell-contribution-link')).not.toBeInTheDocument()
  })
})

describe('TacticalCellCard — contributions vides', () => {
  it('affiche le message vide, la valeur agrégée reste servie', () => {
    renderCard({ contributions: [], contributionsLoading: false })
    expect(screen.getByText(t.cellContributionsEmpty)).toBeInTheDocument()
    expect(screen.getByTestId('tactical-cell-value')).toBeInTheDocument()
  })
})

describe('TacticalCellCard — NOMINAL : liste de contributions avec lien de rejeu', () => {
  it('construit un lien `?frame=` vers la route du rejeu du match', () => {
    renderCard({ contributions: [contribution({ match_id: 'm1', instant_ms: 4200 })] })
    const lien = screen.getByTestId('tactical-cell-contribution-link')
    // 4200 ms / 100 ms (pas par défaut) = frame 42.
    expect(lien).toHaveAttribute(
      'href',
      '/halo_infinite/players/JGtm/matches/m1/replay?frame=42',
    )
  })

  it('rend un lien par contribution, dans l’ordre reçu (le tri est fait côté service)', () => {
    renderCard({
      contributions: [
        contribution({ match_id: 'tard', instant_ms: 2000, match_started_at: '2026-09-02T12:00:00Z' }),
        contribution({ match_id: 'tot', instant_ms: 500, match_started_at: '2026-09-01T12:00:00Z' }),
      ],
    })
    const liens = screen.getAllByTestId('tactical-cell-contribution-link')
    expect(liens).toHaveLength(2)
    expect(liens[0]).toHaveAttribute('href', expect.stringContaining('/matches/tard/replay?frame=20'))
    expect(liens[1]).toHaveAttribute('href', expect.stringContaining('/matches/tot/replay?frame=5'))
  })
})

describe('TacticalCellCard — pied de liste « N matchs comptés non ouvrables »', () => {
  it('rien quand le compte est à zéro', () => {
    renderCard({ contributions: [contribution()], matchsNonOuvrables: 0 })
    expect(screen.queryByTestId('tactical-cell-not-openable')).not.toBeInTheDocument()
  })

  it('affiche le pied de liste quand le compte est strictement positif', () => {
    renderCard({ contributions: [contribution()], matchsNonOuvrables: 2 })
    expect(screen.getByTestId('tactical-cell-not-openable')).toHaveTextContent(
      t.cellFooterNotOpenable(2),
    )
  })
})
