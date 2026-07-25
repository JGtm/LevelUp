/**
 * MatchViewPage — état dédié 404 match_not_found (V72-15.4).
 *
 * Le backend ne fait plus AUCUN fetch live vers l'API du titre quand un match est
 * absent du substrat local (fallback LIVE retiré le 2026-07-25, cf. BACKLOG
 * "Retirer le fallback LIVE du Match view") : il renvoie un 404 typé
 * {code: "match_not_found"}. Ce test vérifie que la page affiche l'état "pas
 * encore synchronisé" (PageUnavailable dédié) plutôt que l'écran d'erreur
 * générique, et que les autres branches d'erreur (ADR 0029) restent inchangées.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchViewPage } from './MatchViewPage'

const hoisted = vi.hoisted(() => ({
  matchView: {
    data: undefined as unknown,
    isPending: false,
    isError: true,
    error: { code: 'match_not_found', message: 'match introuvable : m1' } as unknown,
    refetch: vi.fn(),
  },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player', matchId: 'm1' }),
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
    useRouter: () => ({ history: { length: 1, back: vi.fn() } }),
  }
})

vi.mock('./queries', () => ({
  useMatchView: () => hoisted.matchView,
}))

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { friend_gamertags: [] } }),
}))

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (selector: (s: { locale: 'fr' | 'en' }) => unknown) => selector({ locale: 'fr' }),
}))

describe('MatchViewPage — match_not_found (pas encore synchronisé)', () => {
  it('affiche l\'état dédié « pas encore synchronisé », pas l\'erreur générique', () => {
    hoisted.matchView.error = { code: 'match_not_found', message: 'match introuvable : m1' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText('Match pas encore synchronisé')).toBeInTheDocument()
    expect(
      screen.getByText(/n'est pas encore présent dans la base locale/),
    ).toBeInTheDocument()
    // Pas l'écran d'erreur générique (pageErrorTitle) ni le bouton Réessayer.
    expect(screen.queryByText('Match introuvable ou erreur de chargement.')).not.toBeInTheDocument()
    expect(screen.queryByText('Réessayer')).not.toBeInTheDocument()
  })

  it('propose des actions de navigation (Accueil / Précédent / Mes matchs)', () => {
    hoisted.matchView.error = { code: 'match_not_found', message: 'match introuvable : m1' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByRole('button', { name: 'Accueil' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Précédent' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Mes matchs' })).toBeInTheDocument()
  })

  it('conserve la branche existante match_not_participant (ADR 0029, non régressée)', () => {
    hoisted.matchView.error = { code: 'match_not_participant', message: 'non participant' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText('Match indisponible')).toBeInTheDocument()
    expect(screen.queryByText('Match pas encore synchronisé')).not.toBeInTheDocument()
  })
})
