/**
 * Tests — applyActiveTitle (module title-routing, D-6/D-10).
 *
 * Fonction effectful (ex-switchTitle). Contrats NOUVEAUX à verrouiller ici
 * (l'ordre load-bearing complet reste couvert par appShellStore.applyActiveTitle.test.ts,
 * qui exerce la séquence de bout en bout) :
 *  - no-op si le slug est déjà courant ;
 *  - THROW en cas d'échec, SANS rollback interne (le chemin d'erreur appartient à
 *    l'appelant — D-6, câblé Phase 4b : layout toaste + navigue) ;
 *  - isTitleSwitching remis à false dans le finally, même sur échec.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'

const post = vi.fn()
const get = vi.fn()
const setApiTitleSlug = vi.fn()

vi.mock('@/lib/api/client', () => ({
  api: {
    post: (...args: unknown[]) => post(...args),
    get: (...args: unknown[]) => get(...args),
  },
  setApiTitleSlug: (slug: string | null) => setApiTitleSlug(slug),
  getApiTitleSlug: () => 'halo_infinite',
  setApiLocale: vi.fn(),
}))

vi.mock('@/app/queryClient', () => ({
  queryClient: {
    cancelQueries: vi.fn(async () => {}),
    clear: vi.fn(),
  },
}))

import { applyActiveTitle } from './applyActiveTitle'
import { useAppShellStore } from '@/stores/appShellStore'
import type { BootstrapResponse } from '@/lib/api/types'

const PLAYER = {
  player_slug: 'p1', gamertag: 'P1', xuid: '1', waypoint_player: 'P1', is_demo: false, sync_enabled: true,
}

// Fixture bootstrap minimale (hydrateFromBootstrap défaute le reste des champs).
const BOOTSTRAP_H5 = {
  current_title_slug: 'halo_5',
  current_player: PLAYER,
  available_players: [PLAYER],
  available_titles: [],
  locale: 'fr',
  hints_visible_default: true,
} as unknown as BootstrapResponse

describe('applyActiveTitle', () => {
  beforeEach(() => {
    post.mockReset()
    get.mockReset()
    setApiTitleSlug.mockClear()
    useAppShellStore.setState({
      currentTitleSlug: 'halo_infinite',
      currentPlayer: PLAYER,
      availablePlayers: [PLAYER],
      isTitleSwitching: false,
    })
  })

  it('no-op si le slug est déjà courant', async () => {
    await applyActiveTitle('halo_infinite')
    expect(post).not.toHaveBeenCalled()
    expect(get).not.toHaveBeenCalled()
    expect(useAppShellStore.getState().isTitleSwitching).toBe(false)
  })

  it('happy path : POST session → re-bootstrap → hydrate le nouveau titre', async () => {
    post.mockResolvedValueOnce(undefined)
    get.mockResolvedValueOnce(BOOTSTRAP_H5)

    await applyActiveTitle('halo_5')

    expect(post).toHaveBeenCalledWith('/session/context', { title_slug: 'halo_5' })
    expect(setApiTitleSlug).toHaveBeenCalledWith('halo_5')
    expect(get).toHaveBeenCalledWith('/bootstrap')
    expect(useAppShellStore.getState().currentTitleSlug).toBe('halo_5')
    expect(useAppShellStore.getState().isTitleSwitching).toBe(false)
  })

  it('THROW sans rollback interne si le re-bootstrap échoue', async () => {
    post.mockResolvedValueOnce(undefined)
    get.mockRejectedValueOnce(new Error('boom'))

    await expect(applyActiveTitle('halo_5')).rejects.toThrow('boom')

    // Aucun rollback interne : le nouveau titre posé avant l'échec RESTE en place
    // (c'est l'appelant — layout t/$titleSlug (4b) / fallback TitleSwitcher — qui
    // décide du chemin d'erreur, D-6).
    expect(useAppShellStore.getState().currentTitleSlug).toBe('halo_5')
    // finally : drapeau de bascule bien remis à false.
    expect(useAppShellStore.getState().isTitleSwitching).toBe(false)
  })
})
