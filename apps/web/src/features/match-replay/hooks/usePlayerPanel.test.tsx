/**
 * Tests — l'état du panneau de joueur côté React.
 *
 * Ce qu'ils protègent :
 *  1. LES GESTES DE LA PAGE passent par le réducteur pur : clic de fiche, œil d'une tuile, Seul, « Tout afficher »,
 *     retour au fil, geste manuel de cadrage.
 *  2. LE JOUEUR SE RETROUVE AU ROSTER : `focus` porte le joueur, son « Seul » et son masquage, `null` panneau
 *     fermé ; `hiddenPlayers` porte les joueurs masqués, indépendamment de la fiche.
 *  3. UN AUTRE MATCH REPART À ZÉRO : l'état — fiche, caméra, joueurs masqués — n'est jamais mémorisé (D-4), alors
 *     que la route garde le même composant quand seul `$matchId` change.
 *  4. SUIVRE EST LA CAMÉRA, À PART DE LA FICHE ; LE JOUEUR SUIVI EST LE JOUEUR REGARDÉ, passé par la page.
 *  5. LES GESTES GARDENT LEUR IDENTITÉ : ils sont passés en props à tout le canvas.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { usePlayerPanel } from './usePlayerPanel'

const A: ReplayPlayer = { xuid: 'A', lives: [] }
const B: ReplayPlayer = { xuid: 'B', lives: [] }
const C: ReplayPlayer = { xuid: 'C', lives: [] }

describe('usePlayerPanel', () => {
  it('démarre fermé, sans joueur ni masqué', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
    expect(result.current.hiddenPlayers).toBeNull()
    expect(result.current.tileActions.hidden.size).toBe(0)
  })

  it('un clic de fiche ouvre le panneau sur ce joueur ; un autre le remplace', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toEqual({ player: A, solo: false, hidden: false })
    act(() => result.current.onCard('B'))
    expect(result.current.focus?.player).toBe(B)
  })

  it('masquer deux joueurs : tous deux sont masqués, la fiche ne change pas', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onToggleHidden('B'))
    act(() => result.current.onToggleHidden('C'))
    expect(result.current.hiddenPlayers).toEqual([B, C])
    expect(result.current.focus).toBeNull()
    expect(result.current.state.open).toBe(false)
  })

  it('l’œil rebasculé réaffiche, « Tout afficher » réaffiche tous les joueurs', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onToggleHidden('B'))
    act(() => result.current.onToggleHidden('C'))
    act(() => result.current.onToggleHidden('B'))
    expect(result.current.hiddenPlayers).toEqual([C])
    act(() => result.current.onShowAll())
    expect(result.current.hiddenPlayers).toBeNull()
  })

  it('Seul masque les autres joueurs DU ROSTER ; la fiche lit « seul » ; refait, il réaffiche tout', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onSolo('A'))
    expect(result.current.hiddenPlayers).toEqual([B, C])
    expect(result.current.focus).toEqual({ player: A, solo: true, hidden: false })
    act(() => result.current.onSolo('A'))
    expect(result.current.hiddenPlayers).toBeNull()
    expect(result.current.focus).toMatchObject({ solo: false })
  })

  it('après Seul, l’œil d’une tuile réaffiche un joueur : la fiche n’est plus « seul »', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onSolo('A'))
    act(() => result.current.onToggleHidden('B'))
    expect(result.current.hiddenPlayers).toEqual([C])
    expect(result.current.focus).toMatchObject({ solo: false })
  })

  it('les joueurs masqués ne dépendent pas de la fiche : ouvrir, changer, fermer la fiche ne les touche pas', () => {
    const roster = [A, B, C]
    const { result } = renderHook(() => usePlayerPanel('m1', roster, null))
    act(() => result.current.onToggleHidden('B'))
    const masques = result.current.hiddenPlayers
    act(() => result.current.onCard('A'))
    act(() => result.current.onCard('C'))
    act(() => result.current.onClose())
    expect(result.current.hiddenPlayers).toBe(masques)
    expect(result.current.focus).toBeNull()
  })

  it('« Retour au fil » ferme la fiche et laisse les joueurs masqués', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggleHidden('B'))
    act(() => result.current.onClose())
    expect(result.current.focus).toBeNull()
    expect(result.current.hiddenPlayers).toEqual([B])
  })

  it('un geste manuel de cadrage arrête Suivre et rien d’autre', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('A'))
    act(() => result.current.onToggleHidden('B'))
    act(() => result.current.onFollowStop())
    expect(result.current.state.followXuid).toBeNull()
    expect(result.current.hiddenPlayers).toEqual([B])
  })

  it('un autre match repart de zéro, même avec le même joueur au roster : plus de fiche ni de joueur masqué', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B], null), {
      initialProps: { scope: 'm1' },
    })
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggleHidden('A'))
    act(() => result.current.onToggleHidden('B'))
    expect(result.current.focus).toMatchObject({ hidden: true })
    rerender({ scope: 'm2' })
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
    expect(result.current.hiddenPlayers).toBeNull()
    act(() => result.current.onCard('A'))
    expect(result.current.focus).toMatchObject({ hidden: false })
  })

  it('aller du match A au match B puis revenir à A ne rouvre rien : ni panneau, ni caméra, ni joueurs masqués d’avant', () => {
    const { result, rerender } = renderHook(({ scope }) => usePlayerPanel(scope, [A, B], null), {
      initialProps: { scope: 'A' },
    })
    act(() => result.current.onCard('A'))
    act(() => result.current.onFollow('A'))
    act(() => result.current.onSolo('A'))
    rerender({ scope: 'B' })
    rerender({ scope: 'A' })
    expect(result.current.state.open).toBe(false)
    expect(result.current.focus).toBeNull()
    expect(result.current.hiddenPlayers).toBeNull()
    expect(result.current.camera.player).toBeNull()
  })

  it('les gestes gardent leur identité d’un rendu à l’autre (le roster de la page est stable)', () => {
    const roster = [A, B]
    const { result, rerender } = renderHook(() => usePlayerPanel('m1', roster, null))
    const first = result.current
    rerender()
    expect(result.current.onCard).toBe(first.onCard)
    expect(result.current.onSolo).toBe(first.onSolo)
    expect(result.current.onToggleHidden).toBe(first.onToggleHidden)
    expect(result.current.onShowAll).toBe(first.onShowAll)
    expect(result.current.onClose).toBe(first.onClose)
    expect(result.current.onFollowStop).toBe(first.onFollowStop)
    expect(result.current.onFollow).toBe(first.onFollow)
    expect(result.current.tileActions).toBe(first.tileActions)
  })

  it('un joueur absent du roster ferme le panneau à l’écran (jamais un joueur deviné)', () => {
    const { result, rerender } = renderHook(({ players }) => usePlayerPanel('m1', players, null), {
      initialProps: { players: [A, B] },
    })
    act(() => result.current.onCard('B'))
    rerender({ players: [A] })
    expect(result.current.focus).toBeNull()
  })

  it('un joueur masqué qui quitte le roster n’est plus un joueur masqué à l’écran', () => {
    const { result, rerender } = renderHook(({ players }) => usePlayerPanel('m1', players, null), {
      initialProps: { players: [A, B] },
    })
    act(() => result.current.onToggleHidden('B'))
    rerender({ players: [A] })
    expect(result.current.hiddenPlayers).toBeNull()
  })

  it('Suivre survit à « Retour au fil » : la fiche ne rend rien, mais la caméra garde son joueur', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onFollow('A'))
    expect(result.current.camera.player).toBe(A)
    act(() => result.current.onClose())
    expect(result.current.state.followXuid).toBe('A')
    expect(result.current.focus).toBeNull()
    expect(result.current.camera.player).toBe(A)
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

  it('la caméra : le menu suit un joueur par le geste (onFollow), « Aucun » l’éteint, sans toucher aux masqués', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggleHidden('A'))
    act(() => result.current.camera.onFollow('B'))
    expect(result.current.camera.player).toBe(B)
    expect(result.current.focus).toMatchObject({ player: A, hidden: true })
    act(() => result.current.camera.onFollow(null))
    expect(result.current.camera.player).toBeNull()
    expect(result.current.focus).toMatchObject({ hidden: true })
  })

  it('suivre un joueur masqué le réaffiche (D-15)', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onToggleHidden('B'))
    act(() => result.current.camera.onFollow('B'))
    expect(result.current.hiddenPlayers).toBeNull()
    expect(result.current.camera.player).toBe(B)
  })

  it('masquer le joueur suivi arrête le suivi (D-15)', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('B'))
    act(() => result.current.onToggleHidden('B'))
    expect(result.current.camera.player).toBeNull()
    expect(result.current.hiddenPlayers).toEqual([B])
  })

  it('Seul sur un joueur éteint la caméra posée sur un autre joueur, et la garde sur lui-même (D-15)', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B], null))
    act(() => result.current.onFollow('B'))
    act(() => result.current.onSolo('A'))
    expect(result.current.camera.player).toBeNull()
    act(() => result.current.onShowAll())
    act(() => result.current.onFollow('A'))
    act(() => result.current.onSolo('A'))
    expect(result.current.camera.player).toBe(A)
  })

  it('la caméra garde son identité tant que le joueur suivi ne change pas, même quand la fiche ou les masqués changent', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onFollow('B'))
    const camera = result.current.camera
    act(() => result.current.onCard('A'))
    act(() => result.current.onToggleHidden('C'))
    expect(result.current.camera).toBe(camera)
  })

  it('les actions des tuiles disent qui est suivi et qui est masqué', () => {
    const { result } = renderHook(() => usePlayerPanel('m1', [A, B, C], null))
    act(() => result.current.onFollow('A'))
    act(() => result.current.onToggleHidden('C'))
    expect(result.current.tileActions.followedXuid).toBe('A')
    expect([...result.current.tileActions.hidden]).toEqual(['C'])
    act(() => result.current.tileActions.onToggleHidden('B'))
    act(() => result.current.tileActions.onFollow(null))
    expect(result.current.tileActions.followedXuid).toBeNull()
    expect([...result.current.tileActions.hidden].sort()).toEqual(['B', 'C'])
    act(() => result.current.tileActions.onShowAll())
    expect(result.current.tileActions.hidden.size).toBe(0)
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
