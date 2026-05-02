/**
 * SquadImpactScoreboard.test.tsx — TanStack Table teammates.07.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadImpactMatrix } from '@/lib/api/types'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

function matrix(): SquadImpactMatrix {
  return {
    matches: [
      { match_id: 'm1', outcome: 2 },
      { match_id: 'm2', outcome: 3 },
    ],
    badge_ord: [
      'first_blood', 'clutch_finisher', 'last_casualty', 'last_group_kill',
      'first_group_death', 'silent_hero', 'false_brother', 'top_killer',
    ],
    players: [
      {
        player: 'Champ',
        score: 4,
        counts: [
          { badge_key: 'first_blood', count: 2 },
          { badge_key: 'clutch_finisher', count: 1 },
          { badge_key: 'last_casualty', count: 0 },
          { badge_key: 'last_group_kill', count: 0 },
          { badge_key: 'first_group_death', count: 0 },
          { badge_key: 'silent_hero', count: 0 },
          { badge_key: 'false_brother', count: 0 },
          { badge_key: 'top_killer', count: 1 },
        ],
      },
      {
        player: 'Mid',
        score: 0,
        counts: [
          { badge_key: 'first_blood', count: 0 },
          { badge_key: 'clutch_finisher', count: 0 },
          { badge_key: 'last_casualty', count: 0 },
          { badge_key: 'last_group_kill', count: 0 },
          { badge_key: 'first_group_death', count: 0 },
          { badge_key: 'silent_hero', count: 0 },
          { badge_key: 'false_brother', count: 0 },
          { badge_key: 'top_killer', count: 0 },
        ],
      },
      {
        player: 'WeakLink',
        score: -3,
        counts: [
          { badge_key: 'first_blood', count: 0 },
          { badge_key: 'clutch_finisher', count: 0 },
          { badge_key: 'last_casualty', count: 1 },
          { badge_key: 'last_group_kill', count: 1 },
          { badge_key: 'first_group_death', count: 0 },
          { badge_key: 'silent_hero', count: 0 },
          { badge_key: 'false_brother', count: 1 },
          { badge_key: 'top_killer', count: 0 },
        ],
      },
    ],
    cells: [
      { player: 'Champ', match_id: 'm1', badge_keys: ['first_blood', 'clutch_finisher'] },
      { player: 'WeakLink', match_id: 'm2', badge_keys: ['last_casualty', 'false_brother'] },
    ],
  }
}

describe('SquadImpactScoreboard', () => {
  it('rend le tableau avec 3 joueurs + colonnes match + agrégat + score + rang', () => {
    renderWithProviders(<SquadImpactScoreboard matrix={matrix()} />)
    const table = screen.getByTestId('squad-impact-scoreboard')
    expect(table).toBeInTheDocument()

    expect(screen.getByText('Champ')).toBeInTheDocument()
    expect(screen.getByText('Mid')).toBeInTheDocument()
    expect(screen.getByText('WeakLink')).toBeInTheDocument()

    // Colonnes match (numéro 1 et 2 dans header) — vérifie via title=match_id.
    expect(screen.getByTitle('m1')).toBeInTheDocument()
    expect(screen.getByTitle('m2')).toBeInTheDocument()

    // Score Champ = +4, WeakLink = -3.
    expect(screen.getByText('+4')).toBeInTheDocument()
    expect(screen.getByText('-3')).toBeInTheDocument()

    // Rangs : Champion (rank 1), Maillon faible (rank N, score < 0).
    expect(screen.getByText(/Champion/)).toBeInTheDocument()
    expect(screen.getByText(/Maillon faible/)).toBeInTheDocument()
  })

  it('cellule joueur×match contient les emojis des badges (empilés 2/ligne)', () => {
    renderWithProviders(<SquadImpactScoreboard matrix={matrix()} />)
    // Champ a deux badges sur m1 (first_blood ⚡, clutch_finisher 🎯).
    expect(screen.getByText('⚡🎯')).toBeInTheDocument()
    // WeakLink a deux badges sur m2 (last_casualty 💀, false_brother 🗡️).
    expect(screen.getByText('💀🗡️')).toBeInTheDocument()
  })

  it('cellule agrégat à 0 affiche "—"', () => {
    renderWithProviders(<SquadImpactScoreboard matrix={matrix()} />)
    const dashes = screen.getAllByText('—')
    // Beaucoup de zéros — au moins 1 par badge sans count pour Mid.
    expect(dashes.length).toBeGreaterThan(0)
  })

  it('aucun match ou aucun joueur → null', () => {
    const empty: SquadImpactMatrix = { matches: [], players: [], cells: [], badge_ord: [] }
    const { container } = renderWithProviders(<SquadImpactScoreboard matrix={empty} />)
    expect(container.querySelector('[data-testid="squad-impact-scoreboard"]')).toBeNull()
  })

  it('passager clandestin si score >= 0 sur le rang dernier', () => {
    const m = matrix()
    // Rends le dernier non-négatif en bumpant son score.
    m.players[2].score = 1
    renderWithProviders(<SquadImpactScoreboard matrix={m} />)
    expect(screen.getByText(/Passager/)).toBeInTheDocument()
  })
})
