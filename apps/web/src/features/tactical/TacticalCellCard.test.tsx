/**
 * TacticalCellCard — la carte « Cellule sélectionnée » (item 5.6, posée par le lot M1
 * — lien « voir dans le rejeu » ; horloge EXACTE, lot M1b du 2026-09-08).
 *
 * Ce que ces tests cadenassent :
 *   - aucune cellule sélectionnée -> le placeholder, aucune section de contributions ;
 *   - chargement des contributions -> le message d'attente, pas de liste ;
 *   - contributions vides -> le message vide, jamais le placeholder de la cellule (la
 *     valeur agrégée reste affichée) ;
 *   - NOMINAL -> chaque contribution est un `<Link>` DU ROUTEUR (jamais un `<a href>`
 *     natif, qui rechargerait le document entier — c'est le défaut corrigé le 2026-09-13)
 *     portant `?t=<instant_ms>&clock=<clock>` (JAMAIS une frame pré-calculée ici — la route
 *     convertit, cf. TacticalCellCard.tsx) vers la route du rejeu du bon match, avec LE
 *     CLOCK DE LA CONTRIBUTION (pas une valeur fixe) ;
 *   - l'ISSUE du match est dite à côté de la date, avec le mot du titre (`outcomes.toml`)
 *     et jamais un mot en dur ; une issue inconnue n'affiche RIEN ;
 *   - `matchsNonOuvrables` -> le pied de liste seulement quand il est strictement positif
 *     (0 => rien, jamais un zéro qui suggérerait une absence de restriction).
 *
 * `Link`/`useTitleSlug` sont MOQUÉS : ce composant ne teste pas le routage lui-même
 * (couvert par les tests de route du rejeu), seulement que `TacticalCellCard` passe la
 * bonne cible, les bons params et la bonne recherche — ET QU'IL PASSE PAR `<Link>`, ce que
 * le mock cadenasse : un retour au `<a href>` natif ferait échouer l'import.
 */
import type React from 'react'
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalCellCard } from './TacticalCellCard'

vi.mock('@tanstack/react-router', () => ({
  // Le `<Link>` du routeur, réduit à ce que la carte en attend : une balise `<a>` dont le
  // `href` reflète `to`/`params`/`search`. Il n'existe QUE si le composant importe bien
  // `Link` — un retour au `<a href>` natif casserait ce test à l'import.
  Link: ({
    params,
    search,
    children,
    ...rest
  }: {
    to: string
    params: { titleSlug: string; playerSlug: string; matchId: string }
    search?: Record<string, unknown>
    children?: React.ReactNode
  } & Record<string, unknown>) => {
    const base = `/${params.titleSlug}/players/${params.playerSlug}/matches/${params.matchId}/replay`
    const qs = search
      ? Object.entries(search)
          .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
          .join('&')
      : ''
    return (
      <a href={qs ? `${base}?${qs}` : base} {...(rest as Record<string, unknown>)}>
        {children}
      </a>
    )
  },
}))

// L'ISSUE vient des mappings du titre (`outcomes.toml`) : moquée ici pour que le test
// porte sur CE QUE LA CARTE EN FAIT, pas sur le chargement des mappings.
vi.mock('@/lib/i18n/fieldMappings', () => ({
  useOutcomeMapping: (key: string) =>
    key === 'win'
      ? { label: 'Victoire', color_token: 'outcome.positive' }
      : key === 'loss'
        ? { label: 'Défaite', color_token: 'outcome.negative' }
        : undefined,
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
    clock: 'match',
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
  it('construit un lien `?t=&clock=` (horloge MATCH) vers la route du rejeu du match', () => {
    renderCard({
      contributions: [contribution({ match_id: 'm1', instant_ms: 4200, clock: 'match' })],
    })
    const lien = screen.getByTestId('tactical-cell-contribution-link')
    expect(lien).toHaveAttribute(
      'href',
      '/halo_infinite/players/JGtm/matches/m1/replay?t=4200&clock=match',
    )
  })

  it('construit un lien `?t=&clock=` (horloge FILM) pour une contribution `temps`/`routes`', () => {
    renderCard({
      contributions: [contribution({ match_id: 'm1', instant_ms: 4200, clock: 'film' })],
    })
    const lien = screen.getByTestId('tactical-cell-contribution-link')
    expect(lien).toHaveAttribute(
      'href',
      '/halo_infinite/players/JGtm/matches/m1/replay?t=4200&clock=film',
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
    expect(liens[0]).toHaveAttribute('href', expect.stringContaining('/matches/tard/replay?t=2000&clock=match'))
    expect(liens[1]).toHaveAttribute('href', expect.stringContaining('/matches/tot/replay?t=500&clock=match'))
  })
})

describe('TacticalCellCard — issue du match à côté de la date', () => {
  it('affiche le mot du titre pour une victoire', () => {
    renderCard({ contributions: [contribution({ resultat: 'win' })] })
    expect(screen.getByText('Victoire')).toBeInTheDocument()
  })

  it("n'affiche RIEN quand l'issue est inconnue (jamais un mot par défaut)", () => {
    renderCard({ contributions: [contribution({ resultat: undefined })] })
    expect(screen.queryByText('Victoire')).not.toBeInTheDocument()
    expect(screen.queryByText('Défaite')).not.toBeInTheDocument()
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
