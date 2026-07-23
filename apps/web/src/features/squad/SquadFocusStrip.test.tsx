/**
 * SquadFocusStrip.test.tsx — boucle UI défis d'escouade (Lot 2).
 *
 * Couvre : le label localisé d'un défi s'affiche (pas le template_id brut) ;
 * l'état « Rejoint » remplace « Rejoindre » quand le joueur courant est déjà
 * participant ; la progression renvoyée par « Réévaluer » s'affiche par membre.
 * Les hooks squad sont mockés — on teste le rendu/la logique du composant ; le
 * parcours DB complet est couvert par la revue navigateur + les tests Go.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammateRow } from '@/lib/api/types'
import { SquadFocusStrip } from './SquadFocusStrip'

// Données de test hoistées : référençables depuis les factories vi.mock (qui
// sont remontées au-dessus des imports par vitest).
const H = vi.hoisted(() => {
  const matchedSquad = {
    squad: { id: 'sq1', name: 'Trio', created_by: 'alice', created_at: '2026-07-01T00:00:00Z' },
    members: [
      { squad_id: 'sq1', xuid: 'xA', user_id: 'alice', gamertag: 'Alice', joined_at: '2026-07-01T00:00:00Z' },
      { squad_id: 'sq1', xuid: 'xB', gamertag: 'Bob', joined_at: '2026-07-01T00:00:00Z' },
    ],
  }
  const challenge = {
    id: 'sc1',
    squad_id: 'sq1',
    template_id: 'halo_infinite.daily.headshots_session',
    title_slug: 'halo_infinite',
    mode: 'collective',
    eval_type: 'threshold',
    window_type: 'session',
    window_value: '1',
    target_per_member: 7,
    created_by: 'alice',
    created_at: '2026-07-02T00:00:00Z',
    label_fr: 'Briseur de couronnes',
    label_en: 'Crown breaker',
    participants: [
      {
        squad_challenge_id: 'sc1',
        user_id: 'alice',
        data_tier: 'full',
        current_value: 0,
        is_private: false,
        joined_at: '2026-07-02T00:00:00Z',
      },
    ],
  }
  return { matchedSquad, challenge, evaluateMutate: vi.fn() }
})

vi.mock('@/features/prestige/hooks/useSquads', () => ({
  useMySquads: () => ({ data: { squads: [H.matchedSquad], count: 1 } }),
  useSquadChallenges: () => ({ data: { squad_challenges: [H.challenge], count: 1 } }),
  useCreateSquad: () => ({ mutate: vi.fn(), isPending: false }),
  useRenameSquad: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteSquad: () => ({ mutate: vi.fn(), isPending: false }),
  useEvaluateSquadChallenge: () => ({ mutate: H.evaluateMutate, isPending: false }),
  useRefreshSquadPool: () => ({ mutate: vi.fn(), isPending: false, isSuccess: false, data: undefined }),
  useCreateSquadChallenge: () => ({ mutate: vi.fn(), isPending: false }),
  useSquadOrientation: () => ({ data: undefined }),
}))

vi.mock('@/features/prestige/hooks', () => ({
  useJoinSquadChallenge: () => ({ mutate: vi.fn(), isPending: false }),
}))

const ROW = (gamertag: string, xuid: string): TeammateRow => ({
  gamertag,
  xuid,
  encounter_count: 5,
  last_seen_at: undefined,
  with_kpis: {
    match_count: 5,
    wins: 3,
    kd_ratio: 1.5,
    win_rate: 0.6,
    accuracy: 0.45,
    kills_per_game: 12,
    assists_per_game: 4,
  },
  without_kpis: undefined,
})

function mockContext(selectedRows: TeammateRow[]) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows,
    confirmedGamertags: selectedRows.map((r) => r.gamertag),
    pageData: null,
    playerSlug: 'alice',
  })
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  H.evaluateMutate.mockReset()
})

describe('SquadFocusStrip — boucle défis (Lot 2)', () => {
  it('affiche le label localisé du défi et l\'état « Rejoint », pas le template_id', () => {
    mockContext([ROW('Bob', 'xB')])
    renderWithProviders(<SquadFocusStrip />)

    // Ouvre le panneau « Gérer » (escouade appariée).
    fireEvent.click(screen.getByRole('button', { name: 'Gérer' }))

    // Label FR affiché, template_id brut absent.
    expect(screen.getByText('Briseur de couronnes')).toBeInTheDocument()
    expect(screen.queryByText(/headshots_session/)).not.toBeInTheDocument()

    // Le joueur courant (alice) est déjà participant → bouton « Rejoint » désactivé.
    const joined = screen.getByRole('button', { name: 'Rejoint' })
    expect(joined).toBeDisabled()
  })

  it('affiche la progression par membre renvoyée par « Réévaluer »', () => {
    // Simule le retour d'évaluation (2 membres, l'un a atteint la cible).
    H.evaluateMutate.mockImplementation((_vars: unknown, opts?: { onSuccess?: (r: unknown) => void }) => {
      opts?.onSuccess?.({
        progress: [
          { xuid: 'xA', value: 9, matches: 3, completed: true },
          { xuid: 'xB', value: 4, matches: 2, completed: false },
        ],
      })
    })
    mockContext([ROW('Bob', 'xB')])
    renderWithProviders(<SquadFocusStrip />)
    fireEvent.click(screen.getByRole('button', { name: 'Gérer' }))
    fireEvent.click(screen.getByRole('button', { name: 'Réévaluer' }))

    // Gamertags résolus depuis le roster + valeur/cible + badge « Atteint ».
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('9 / 7')).toBeInTheDocument()
    expect(screen.getByText('Atteint')).toBeInTheDocument()
    expect(screen.getByText('4 / 7')).toBeInTheDocument()
  })
})
