/**
 * Tests — le bloc « Citations bientôt terminées » choisit sa source sur la capability de
 * titre `citations` (moteur calculé servi ou non), jamais sur le slug du titre.
 *
 * Oracle : lequel des deux hooks de données est appelé. Les hooks sont remplacés par des
 * espions qui restent en chargement (le composant ne rend alors rien) : seule la source
 * choisie compte ici.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest } from '@/lib/i18n/generated/home'

const useCitationsPage = vi.fn()
const useCommendationTotals = vi.fn()
const navigateMock = vi.fn()

vi.mock('@/features/citations/queries', () => ({
  useCitationsPage: (...args: unknown[]) => useCitationsPage(...args),
}))
vi.mock('@/features/commendations/queries', () => ({
  useCommendationTotals: (...args: unknown[]) => useCommendationTotals(...args),
}))
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigateMock }
})
// Tout est maîtrisé : la section se rend (message de félicitations) et son lien « Tout voir » est actif.
vi.mock('@/lib/citations/normalize', () => ({
  normalizeInfinitePage: () => ({ categories: [] }),
  normalizeNativeTotals: () => ({ categories: [] }),
}))
vi.mock('@/lib/citations/nearCompletion', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/citations/nearCompletion')>()),
  selectNearCompletion: () => [],
  allCitationsMastered: () => true,
}))

import { HomeCitationsNearCompletion } from './HomeCitationsNearCompletion'

type Titles = ReturnType<typeof useAppShellStore.getState>['availableTitles']

function setTitle(slug: string, capabilities: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: slug,
    availableTitles: [
      { slug, name: slug, status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as Titles,
  })
}

describe('HomeCitationsNearCompletion : choix de la source', () => {
  beforeEach(() => {
    useCitationsPage.mockReset().mockReturnValue({ data: undefined, isLoading: true })
    useCommendationTotals.mockReset().mockReturnValue({ data: undefined, isLoading: true })
    navigateMock.mockReset()
  })

  it('lit les citations calculées quand le titre déclare la capability citations', () => {
    setTitle('halo_infinite', ['career', 'citations'])
    renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)

    // Totaux de carrière : le hook ne reçoit que le joueur, aucun filtre ni empreinte de filtre.
    expect(useCitationsPage).toHaveBeenCalledWith('p')
    expect(useCommendationTotals).not.toHaveBeenCalled()
  })

  it('lit les commendations natives quand le titre ne la déclare pas', () => {
    setTitle('halo_5', ['career'])
    renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)

    expect(useCommendationTotals).toHaveBeenCalled()
    expect(useCitationsPage).not.toHaveBeenCalled()
  })

  it('ne se fie pas au slug : un titre quelconque sans la capability lit les natives', () => {
    setTitle('titre_b', ['career'])
    renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)

    expect(useCommendationTotals).toHaveBeenCalled()
    expect(useCitationsPage).not.toHaveBeenCalled()
  })

  it('ne se fie pas au slug : halo_5 qui déclare citations lit les citations calculées', () => {
    setTitle('halo_5', ['career', 'citations'])
    renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)

    expect(useCitationsPage).toHaveBeenCalled()
    expect(useCommendationTotals).not.toHaveBeenCalled()
  })

  it('« Tout voir » navigue vers la route finale, jamais vers la redirection', () => {
    for (const [capabilities, finale] of [
      [['career', 'citations'], '/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations'],
      [['career'], '/{-$lang}/t/$titleSlug/players/$playerSlug/career/commendations'],
    ] as const) {
      navigateMock.mockReset()
      setTitle('titre_x', [...capabilities])
      useCitationsPage.mockReturnValue({ data: {}, isLoading: false })
      useCommendationTotals.mockReturnValue({ data: {}, isLoading: false })
      const { unmount } = renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)
      fireEvent.click(screen.getByRole('button', { name: formatMessage(homeManifest, 'home.near_completion.view_all', useAppShellStore.getState().locale) }))
      expect(navigateMock).toHaveBeenCalledWith(expect.objectContaining({ to: finale }))
      unmount()
    }
  })
})
