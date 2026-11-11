/**
 * Tests — l'état du panneau de joueur côté React (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LES GESTES DE LA PAGE passent par le réducteur pur : clic de fiche, bascule d'action, retour au
 *     fil, geste manuel de cadrage.
 *  2. LE JOUEUR SE RETROUVE AU ROSTER : `focus` porte le joueur et ses actions, `null` panneau fermé.
 *  3. UN AUTRE MATCH REPART À ZÉRO : l'état n'est jamais mémorisé (D-4) — pas d'un match à l'autre,
 *     alors que la route garde le même composant quand seul `$matchId` change.
 *  4. LES GESTES GARDENT LEUR IDENTITÉ : ils sont passés en props à tout le canvas.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { usePlayerPanel } from './usePlayerPanel'

const A: ReplayPlayer = { xuid: 'A', lives: [] }
const B: ReplayPlayer = { xuid: 'B', lives: [] }

describe('usePlayerPanel', () => {
  it('démarre fermé, sans joueur', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B]))
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
  })

  it('un clic de fiche ouvre le panneau sur ce joueur ; un autre le remplace', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B]))
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toEqual({ player: A, follow: false, solo: false, hide: false })
    act(() => result.current.onCard('B'))
    expect(result.current.focus?.player).toBe(B)
  })

  it('les actions suivent les exclusions, et « Retour au fil » remet tout à zéro', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B]))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('follow'))
    act(() => result.current.onToggle('solo'))
    expect(result.current.focus).toMatchObject({ follow: true, solo: true, hide: false })
    act(() => result.current.onToggle('hide'))
    expect(result.current.focus).toMatchObject({ follow: false, solo: false, hide: true })
    act(() => result.current.onClose())
    expect(result.current.focus).toBeNull()
    expect(result.current.state.hide).toBe(false)
  })

  it('un geste manuel de cadrage arrête Suivre et rien d’autre', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B]))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('follow'))
    act(() => result.current.onToggle('solo'))
    act(() => result.current.onFollowStop())
    expect(result.current.focus).toMatchObject({ follow: false, solo: true })
  })

  it('un autre match repart de zéro, même avec le même joueur au roster', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B]), {
      initialProps: { scope: 'm1' },
    })
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('hide'))
    expect(result.current.focus).toMatchObject({ hide: true })
    rerender({ scope: 'm2' })
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toMatchObject({ hide: false })
  })

  it('les gestes gardent leur identité d’un rendu à l’autre', () => {
    const { result, rerender } = renderHook(() => usePlayerPanel('m1', [A, B]))
    const first = result.current
    rerender()
    expect(result.current.onCard).toBe(first.onCard)
    expect(result.current.onToggle).toBe(first.onToggle)
    expect(result.current.onClose).toBe(first.onClose)
    expect(result.current.onFollowStop).toBe(first.onFollowStop)
  })

  it('un joueur absent du roster ferme le panneau à l’écran (jamais un joueur deviné)', () => {
    const { result, rerender } = renderHook(({ players }) => usePlayerPanel('m1', players), {
      initialProps: { players: [A, B] },
    })
    act(() => result.current.onCard('B'))
    rerender({ players: [A] })
    expect(result.current.focus).toBeNull()
  })
})
