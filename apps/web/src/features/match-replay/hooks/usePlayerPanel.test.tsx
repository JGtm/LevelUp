/**
 * Tests — l'état du panneau de joueur côté React (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LES GESTES DE LA PAGE passent par le réducteur pur : clic de fiche, bascule d'action, retour au
 *     fil, geste manuel de cadrage.
 *  2. LE JOUEUR SE RETROUVE AU ROSTER : `focus` porte le joueur et ses actions, `null` panneau fermé.
 *  3. UN AUTRE MATCH REPART À ZÉRO : l'état n'est jamais mémorisé (D-4) — pas d'un match à l'autre,
 *     alors que la route garde le même composant quand seul `$matchId` change.
 *  4. SUIVRE EST LA CAMÉRA, À PART DE LA FICHE ; LE JOUEUR SUIVI EST LE JOUEUR REGARDÉ, passé par la page.
 *  5. LES GESTES GARDENT LEUR IDENTITÉ : ils sont passés en props à tout le canvas.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { usePlayerPanel } from './usePlayerPanel'

const A: ReplayPlayer = { xuid: 'A', lives: [] }
const B: ReplayPlayer = { xuid: 'B', lives: [] }

describe('usePlayerPanel', () => {
  it('démarre fermé, sans joueur', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
  })

  it('un clic de fiche ouvre le panneau sur ce joueur ; un autre le remplace', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toEqual({ player: A, follow: false, solo: false, hide: false })
    act(() => result.current.onCard('B'))
    expect(result.current.focus?.player).toBe(B)
  })

  it('les actions suivent les exclusions, et « Retour au fil » remet tout à zéro', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
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
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('follow'))
    act(() => result.current.onToggle('solo'))
    act(() => result.current.onFollowStop())
    expect(result.current.focus).toMatchObject({ follow: false, solo: true })
  })

  it('un autre match repart de zéro, même avec le même joueur au roster', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B], null), {
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

  it('aller du match A au match B puis revenir à A ne rouvre rien : ni panneau ni actions d’avant', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B], null), {
      initialProps: { scope: 'A' },
    })
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('follow'))
    act(() => result.current.onToggle('solo'))
    rerender({ scope: 'B' })
    rerender({ scope: 'A' })
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toMatchObject({ follow: false, solo: false, hide: false })
  })

  it('les gestes gardent leur identité d’un rendu à l’autre', () => {
    const { result, rerender } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    const first = result.current
    rerender()
    expect(result.current.onCard).toBe(first.onCard)
    expect(result.current.onToggle).toBe(first.onToggle)
    expect(result.current.onClose).toBe(first.onClose)
    expect(result.current.onFollowStop).toBe(first.onFollowStop)
    expect(result.current.onFollow).toBe(first.onFollow)
  })

  it('un joueur absent du roster ferme le panneau à l’écran (jamais un joueur deviné)', () => {
    const { result, rerender } = renderHook(({ players }) => usePlayerPanel('m1', players, null), {
      initialProps: { players: [A, B] },
    })
    act(() => result.current.onCard('B'))
    rerender({ players: [A] })
    expect(result.current.focus).toBeNull()
  })
  it('Suivre survit à « Retour au fil » : la fiche ne rend rien, mais la caméra garde son joueur', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('follow'))
    expect(result.current.camera.player).toBe(A)
    act(() => result.current.onClose())
    expect(result.current.state.followXuid).toBe('A')
    expect(result.current.focus).toBeNull()
    expect(result.current.camera.player).toBe(A)
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toMatchObject({ follow: true })
  })

  it('« Suivre tel joueur » est accepté fiche fermée, et un geste manuel l’arrête', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('B'))
    expect(result.current.state).toMatchObject({ open: false, followXuid: 'B' })
    expect(result.current.camera.player).toBe(B)
    act(() => result.current.onFollowStop())
    expect(result.current.state.followXuid).toBeNull()
    expect(result.current.camera.player).toBeNull()
  })

  it('la caméra : le menu suit un joueur par le geste de la fiche (onFollow), « Aucun » l’éteint, Masquer ne bouge pas', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('hide'))
    act(() => result.current.camera.onFollow('B'))
    expect(result.current.camera.player).toBe(B)
    expect(result.current.focus).toMatchObject({ player: A, follow: false, solo: false, hide: true })
    act(() => result.current.camera.onFollow(null))
    expect(result.current.camera.player).toBeNull()
    expect(result.current.focus).toMatchObject({ hide: true })
  })

  it('Seul et la caméra : suivre un AUTRE joueur lève Seul, suivre celui de la fiche le garde', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('solo'))
    act(() => result.current.camera.onFollow('A'))
    expect(result.current.focus).toMatchObject({ follow: true, solo: true })
    act(() => result.current.camera.onFollow('B'))
    expect(result.current.camera.player).toBe(B)
    expect(result.current.focus).toMatchObject({ player: A, follow: false, solo: false })
  })

  it('Seul sur la fiche éteint la caméra posée sur un autre joueur', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('B'))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('solo'))
    expect(result.current.camera.player).toBeNull()
    expect(result.current.focus).toMatchObject({ player: A, solo: true })
  })

  it('suivre depuis le menu le joueur masqué de la fiche lève son masquage (règle du modèle)', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('hide'))
    act(() => result.current.camera.onFollow('A'))
    expect(result.current.focus).toMatchObject({ hide: false, follow: true })
  })

  it('la caméra garde son identité tant que le joueur suivi ne change pas, même quand la fiche change', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('B'))
    const camera = result.current.camera
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggle('hide'))
    expect(result.current.camera).toBe(camera)
  })

  it('un autre match éteint aussi la caméra (rien n’est mémorisé)', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B], null), {
      initialProps: { scope: 'm1' },
    })
    act(() => result.current.onFollow('A'))
    rerender({ scope: 'm2' })
    expect(result.current.state.followXuid).toBeNull()
  })

  it('le joueur suivi est le joueur regardé passé par la page, et suit son changement sans toucher à la caméra', () => {
    const { result, rerender } = renderHook(({ watched }) => usePlayerPanel('m1', [A, B], watched), {
      initialProps: { watched: 'A' as string | null },
    })
    expect(result.current.tracked).toBe(A)
    act(() => result.current.onFollow('B'))
    rerender({ watched: 'B' })
    expect(result.current.tracked).toBe(B)
    rerender({ watched: 'A' })
    expect(result.current.tracked).toBe(A)
    expect(result.current.state.followXuid).toBe('B')
    rerender({ watched: null })
    expect(result.current.tracked).toBeNull()
  })

  it('le joueur suivi ne dépend pas de la fiche : ouvrir ou fermer une fiche ne le change pas', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], 'A'))
    act(() => result.current.onCard('B'))
    expect(result.current.tracked).toBe(A)
    act(() => result.current.onClose())
    expect(result.current.tracked).toBe(A)
  })
})
