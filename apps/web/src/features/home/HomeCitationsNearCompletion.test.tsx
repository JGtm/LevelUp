/**
 * Tests — le bloc « Citations bientôt terminées » choisit sa source sur la capability de
 * titre `citations` (moteur calculé servi ou non), jamais sur le slug du titre.
 *
 * Oracle : lequel des deux hooks de données est appelé. Les hooks sont remplacés par des
 * espions qui restent en chargement (le composant ne rend alors rien) : seule la source
 * choisie compte ici.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

const useCitationsPage = vi.fn()
const useCommendationTotals = vi.fn()

vi.mock('@/features/citations/queries', () => ({
  useCitationsPage: (...args: unknown[]) => useCitationsPage(...args),
}))
vi.mock('@/features/commendations/queries', () => ({
  useCommendationTotals: (...args: unknown[]) => useCommendationTotals(...args),
}))
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => vi.fn() }
})

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
  })

  it('lit les citations calculées quand le titre déclare la capability citations', () => {
    setTitle('halo_infinite', ['career', 'citations'])
    renderWithProviders(<HomeCitationsNearCompletion playerSlug="p" />)

    expect(useCitationsPage).toHaveBeenCalled()
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
})
