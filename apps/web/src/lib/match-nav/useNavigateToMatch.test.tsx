/**
 * useNavigateToMatch — persistance du contexte de navigation à travers le HOOK : un contexte sans
 * liste d'identifiants mais avec descripteur (Accueil : récents) survit à un rechargement de la
 * page du match (sessionStorage), un contexte sans rien à retenir n'écrit rien.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook } from '@testing-library/react'

import { useNavigateToMatch } from './useNavigateToMatch'
import { readNavContext, type MatchNavContext } from './navContext'

const navigateSpy = vi.fn()

vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigateSpy }))
vi.mock('@/lib/title-routing', () => ({ useTitleSlug: () => 'halo_infinite' }))

beforeEach(() => {
  sessionStorage.clear()
  navigateSpy.mockReset()
})

describe('useNavigateToMatch — persistance du contexte', () => {
  it('contexte sans liste mais avec descripteur : conservé pour le match cible (F5, nouvel onglet)', () => {
    const ctx: MatchNavContext = { source: 'home_recent', matchIds: [], contextDescriptor: { kind: 'recent' } }
    const { result } = renderHook(() => useNavigateToMatch('chief'))
    result.current('m1', ctx)
    expect(readNavContext('m1')).toEqual(ctx)
    expect(navigateSpy).toHaveBeenCalledTimes(1)
    expect(navigateSpy.mock.calls[0][0].params).toEqual({ titleSlug: 'halo_infinite', playerSlug: 'chief', matchId: 'm1' })
  })

  it('contexte avec liste : conservé', () => {
    const ctx: MatchNavContext = { source: 'history', matchIds: ['m1', 'm2'] }
    const { result } = renderHook(() => useNavigateToMatch('chief'))
    result.current('m1', ctx)
    expect(readNavContext('m1')).toEqual(ctx)
  })

  it('ni liste ni descripteur : rien n\'est écrit, la navigation a lieu', () => {
    const { result } = renderHook(() => useNavigateToMatch('chief'))
    result.current('m1', { source: 'history', matchIds: [] })
    result.current('m2')
    expect(readNavContext('m1')).toBeNull()
    expect(readNavContext('m2')).toBeNull()
    expect(navigateSpy).toHaveBeenCalledTimes(2)
  })

  it('sans match_id : aucune navigation', () => {
    const { result } = renderHook(() => useNavigateToMatch('chief'))
    result.current('', { source: 'history', matchIds: ['a'] })
    expect(navigateSpy).not.toHaveBeenCalled()
  })
})
