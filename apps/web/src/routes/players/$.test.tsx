/**
 * 3a — Tests du splat legacy `/players/$` (D-5, D-8) : projection DÉCLARATIVE de la
 * décision PURE `buildLegacyRedirect` (déjà couverte en matrice table-driven par
 * `buildLegacyRedirect.test.ts` — ce test ne vérifie QUE le câblage de projection :
 * gate bootstrap, `history.replace` du href, `<Navigate>` pour le cas hors matrice).
 *
 * `useLocation`/`useRouter`/`Navigate` sont mockés (location contrôlée, `history.replace`
 * espionné, `Navigate` = marqueur) ; `buildLegacyRedirect` reste RÉEL. Store piloté par
 * setState direct (pattern `$titleSlug.test.tsx`).
 */
import type { ComponentType } from 'react'
import { describe, it, expect, beforeAll, beforeEach, vi } from 'vitest'
import { act, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

const historyReplace = vi.fn<(path: string) => void>()
// locationRef : l'« URL » courante lue par useLocation() (source BRUTE, cf. splat).
const locationRef = { pathname: '/players/jgtm/home', searchStr: '', hash: '' }

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({ ...opts }),
    useLocation: () => locationRef,
    useRouter: () => ({ history: { replace: historyReplace } }),
    Navigate: ({ to }: { to: string }) => <div data-testid="navigate" data-to={to} />,
  }
})

// Import APRÈS le mock (le composant lit createFileRoute + les hooks au load).
import { Route } from './$'

// `Route.component` n'est pas exposé par le type public → cast (accès interne au mock).
// Le composant porte `.preload` (wrapper lazy autoCodeSplitting) → préchargé en beforeAll.
const routeComponent = (
  Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } }
).component
const LegacyPlayersRedirect: ComponentType = routeComponent

beforeAll(async () => {
  await routeComponent.preload?.()
})

function setStore(partial: Partial<ReturnType<typeof useAppShellStore.getState>>) {
  act(() => useAppShellStore.setState(partial))
}

beforeEach(() => {
  historyReplace.mockClear()
  locationRef.pathname = '/players/jgtm/home'
  locationRef.searchStr = ''
  locationRef.hash = ''
  useAppShellStore.setState({ isBootstrapped: true, currentTitleSlug: 'halo_infinite' })
})

describe('LegacyPlayersRedirect (3a)', () => {
  it('non bootstrappé → ne rend rien et NE redirige PAS (attend la session, trou n°1 D-8)', () => {
    setStore({ isBootstrapped: false, currentTitleSlug: 'halo_infinite' })
    renderWithProviders(<LegacyPlayersRedirect />)
    expect(historyReplace).not.toHaveBeenCalled()
    expect(screen.queryByTestId('navigate')).toBeNull()
  })

  it('bootstrappé (halo_5) → history.replace vers /t/halo_5/… avec ?f= et #hash préservés', () => {
    locationRef.pathname = '/players/x/stats/timeseries'
    locationRef.searchStr = '?f=abc'
    locationRef.hash = 'h' // useLocation() exclut le # (réintroduit par buildLegacyRedirect)
    setStore({ isBootstrapped: true, currentTitleSlug: 'halo_5' })
    renderWithProviders(<LegacyPlayersRedirect />)
    expect(historyReplace).toHaveBeenCalledTimes(1)
    expect(historyReplace).toHaveBeenCalledWith('/fr/t/halo_5/players/x/stats/timeseries?f=abc#h')
    expect(screen.queryByTestId('navigate')).toBeNull()
  })

  it('pathname /players nu (hors matrice legacy) → <Navigate to="/" replace>', () => {
    locationRef.pathname = '/players'
    renderWithProviders(<LegacyPlayersRedirect />)
    expect(historyReplace).not.toHaveBeenCalled()
    expect(screen.getByTestId('navigate').getAttribute('data-to')).toBe('/')
  })

  // Non-régression course post-replace : après le replace, le splat re-rend avec la
  // location DÉJÀ /t/… (pas encore démonté). Il ne doit NI re-rediriger NI tomber sur
  // le fallback <Navigate to="/"> (qui perdrait suffixe + ?f= + #hash — bug observé).
  it('location transitoire /t/… post-replace → null, ni 2e replace ni fallback index', () => {
    locationRef.pathname = '/players/x/stats/timeseries'
    locationRef.searchStr = '?f=abc'
    locationRef.hash = 'deep'
    setStore({ isBootstrapped: true, currentTitleSlug: 'halo_5' })
    const { rerender } = renderWithProviders(<LegacyPlayersRedirect />)

    // 1er rendu : redirection legacy correcte (suffixe + ?f= + #hash préservés).
    expect(historyReplace).toHaveBeenCalledTimes(1)
    expect(historyReplace).toHaveBeenLastCalledWith(
      '/fr/t/halo_5/players/x/stats/timeseries?f=abc#deep',
    )

    // Transition : le routeur a basculé la location vers la cible /t/… ; le splat
    // re-rend avec cette location transitoire AVANT son démontage.
    historyReplace.mockClear()
    act(() => {
      locationRef.pathname = '/fr/t/halo_5/players/x/stats/timeseries'
    })
    rerender(<LegacyPlayersRedirect />)

    expect(historyReplace).not.toHaveBeenCalled() // pas de 2e replace
    expect(screen.queryByTestId('navigate')).toBeNull() // pas d'écrasement par <Navigate to="/">
  })
})
