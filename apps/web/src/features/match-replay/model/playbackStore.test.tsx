/**
 * playbackStore.test.tsx — LA POSITION PUBLIÉE : un seul écrivain, des lecteurs abonnés.
 *
 * CE QUE CES CAS PROTÈGENT (registre 2026-09-05, W1) : la page tenait une COPIE React de la
 * position, poussée par une prop de rappel. Rien ne disait qui l'écrivait, ni à quel rythme,
 * ni ce qui arrivait quand deux montages coexistaient. Le magasin le dit, et ces cas le
 * vérifient — notamment le point qui rend la copie nécessaire : un instantané STABLE entre
 * deux notifications, sans quoi React signale une boucle.
 */
import { describe, expect, it, vi } from 'vitest'
import { act, render, renderHook } from '@testing-library/react'

import type { KillFocusTarget } from './killFocusLogic'
import {
  createPlaybackStore,
  usePlaybackFocus,
  usePlaybackFrame,
  usePlaybackStore,
} from './playbackStore'

describe('createPlaybackStore — la publication', () => {
  it('part de l’image zéro', () => {
    expect(createPlaybackStore().published()).toBe(0)
  })

  it('retient la dernière image publiée et réveille ses abonnés', () => {
    const store = createPlaybackStore()
    const vu = vi.fn()
    store.subscribe(vu)
    store.publish(42)
    expect(store.published()).toBe(42)
    expect(vu).toHaveBeenCalledTimes(1)
  })

  it('NE RÉVEILLE PERSONNE quand l’image ne change pas', () => {
    // La boucle de lecture est souvent plus lente que l'écran : le canvas repeint et
    // republie la même image plusieurs fois de suite, et rien ne doit se re-rendre.
    const store = createPlaybackStore()
    const vu = vi.fn()
    store.subscribe(vu)
    store.publish(7)
    store.publish(7)
    store.publish(7)
    expect(vu).toHaveBeenCalledTimes(1)
  })

  it('rend un INSTANTANÉ STABLE entre deux publications — c’est ce que React exige', () => {
    const store = createPlaybackStore()
    store.publish(12)
    expect(store.published()).toBe(store.published())
    expect(store.published()).toBe(12)
  })

  it('réveille TOUS les abonnés, et plus aucun après désabonnement', () => {
    const store = createPlaybackStore()
    const a = vi.fn()
    const b = vi.fn()
    store.subscribe(a)
    const stopB = store.subscribe(b)
    store.publish(1)
    expect([a, b].map((f) => f.mock.calls.length)).toEqual([1, 1])
    stopB()
    store.publish(2)
    expect([a, b].map((f) => f.mock.calls.length)).toEqual([2, 1])
  })

  it('DEUX MAGASINS SONT INDÉPENDANTS : pas de singleton de module', () => {
    const a = createPlaybackStore()
    const b = createPlaybackStore()
    a.publish(99)
    expect(b.published()).toBe(0)
  })
})

describe('usePlaybackFrame — le lecteur abonné', () => {
  it('rend l’image publiée, et se re-rend quand elle change', () => {
    const store = createPlaybackStore()
    const { result } = renderHook(() => usePlaybackFrame(store))
    expect(result.current).toBe(0)
    act(() => store.publish(150))
    expect(result.current).toBe(150)
  })

  it('NE SE RE-REND PAS sur une publication identique', () => {
    const store = createPlaybackStore()
    const rendus = vi.fn()
    function Lecteur() {
      rendus()
      return <span>{usePlaybackFrame(store)}</span>
    }
    render(<Lecteur />)
    const auMontage = rendus.mock.calls.length
    act(() => store.publish(3))
    const apresChangement = rendus.mock.calls.length
    expect(apresChangement).toBeGreaterThan(auMontage)
    act(() => store.publish(3))
    expect(rendus.mock.calls.length).toBe(apresChangement)
  })
})

describe('usePlaybackStore — un magasin par page', () => {
  it('rend le MÊME magasin d’un rendu à l’autre', () => {
    const { result, rerender } = renderHook(() => usePlaybackStore())
    const premier = result.current
    rerender()
    expect(result.current).toBe(premier)
  })

  it('rend un magasin DIFFÉRENT à un second montage', () => {
    const a = renderHook(() => usePlaybackStore()).result.current
    const b = renderHook(() => usePlaybackStore()).result.current
    expect(a).not.toBe(b)
  })
})

describe('le canal de saut — la page DEMANDE, le canvas EXÉCUTE', () => {
  it('une demande est remise à l’exécuteur abonné, avec l’image visée', () => {
    const store = createPlaybackStore()
    const executer = vi.fn()
    store.onSeekRequest(executer)
    store.requestSeek(120)
    expect(executer).toHaveBeenCalledWith(120)
  })

  it('NE DÉPLACE PAS la position publiée : seul le canvas écrit le temps', () => {
    const store = createPlaybackStore()
    store.publish(10)
    store.onSeekRequest(vi.fn())
    store.requestSeek(120)
    expect(store.published()).toBe(10)
  })

  it('une demande faite AVANT que le canvas soit monté l’attend, et lui est remise une fois', () => {
    const store = createPlaybackStore()
    store.requestSeek(77)
    const executer = vi.fn()
    store.onSeekRequest(executer)
    expect(executer).toHaveBeenCalledTimes(1)
    expect(executer).toHaveBeenCalledWith(77)
    // Une seconde inscription ne rejoue pas une demande déjà remise.
    const autre = vi.fn()
    store.onSeekRequest(autre)
    expect(autre).not.toHaveBeenCalled()
  })

  it('seule la DERNIÈRE demande en attente est remise', () => {
    const store = createPlaybackStore()
    store.requestSeek(1)
    store.requestSeek(2)
    const executer = vi.fn()
    store.onSeekRequest(executer)
    expect(executer).toHaveBeenCalledTimes(1)
    expect(executer).toHaveBeenCalledWith(2)
  })

  it('un exécuteur désabonné ne reçoit plus rien', () => {
    const store = createPlaybackStore()
    const executer = vi.fn()
    const stop = store.onSeekRequest(executer)
    stop()
    store.requestSeek(5)
    expect(executer).not.toHaveBeenCalled()
  })

  it('deux magasins ont deux canaux indépendants', () => {
    const a = createPlaybackStore()
    const b = createPlaybackStore()
    const executer = vi.fn()
    b.onSeekRequest(executer)
    a.requestSeek(9)
    expect(executer).not.toHaveBeenCalled()
  })
})

const CIBLE: KillFocusTarget = {
  entryKey: 'k1',
  frame: 120,
  killerXuid: 'killer',
  victimXuid: 'victim',
  weapon: { imageUrl: '/w.png', tinted: true },
}

describe('la mise en valeur d’une élimination — état du magasin', () => {
  it('part sans mise en valeur', () => {
    expect(createPlaybackStore().focus()).toBeNull()
  })

  it('focusKill pose la mise en valeur et réveille les abonnés', () => {
    const store = createPlaybackStore()
    const vu = vi.fn()
    store.subscribeFocus(vu)
    store.focusKill(CIBLE)
    expect(store.focus()).toMatchObject({ entryKey: 'k1', frame: 120, token: 1 })
    expect(vu).toHaveBeenCalledTimes(1)
  })

  it('NE POSE PAS la lecture : le saut est une demande à part', () => {
    const store = createPlaybackStore()
    const executer = vi.fn()
    store.onSeekRequest(executer)
    store.focusKill(CIBLE)
    expect(executer).not.toHaveBeenCalled()
  })

  it('un autre clic REMPLACE la mise en valeur, et le même clic répété la REJOUE (jeton neuf)', () => {
    const store = createPlaybackStore()
    store.focusKill(CIBLE)
    const premier = store.focus()
    store.focusKill(CIBLE)
    expect(store.focus()?.token).toBe((premier?.token ?? 0) + 1)
    store.focusKill({ ...CIBLE, entryKey: 'k2' })
    expect(store.focus()?.entryKey).toBe('k2')
  })

  it('clearFocus efface, et ne réveille personne quand il n’y a rien à effacer', () => {
    const store = createPlaybackStore()
    const vu = vi.fn()
    store.subscribeFocus(vu)
    store.clearFocus()
    expect(vu).not.toHaveBeenCalled()
    store.focusKill(CIBLE)
    store.clearFocus()
    expect(store.focus()).toBeNull()
    expect(vu).toHaveBeenCalledTimes(2)
  })

  it('l’instantané est STABLE entre deux changements', () => {
    const store = createPlaybackStore()
    store.focusKill(CIBLE)
    expect(store.focus()).toBe(store.focus())
  })

  it('un abonné désabonné n’est plus réveillé', () => {
    const store = createPlaybackStore()
    const vu = vi.fn()
    const stop = store.subscribeFocus(vu)
    stop()
    store.focusKill(CIBLE)
    expect(vu).not.toHaveBeenCalled()
  })

  it('la mise en valeur n’est PAS mémorisée : un magasin neuf repart vide', () => {
    createPlaybackStore().focusKill(CIBLE)
    expect(createPlaybackStore().focus()).toBeNull()
  })
})

describe('usePlaybackFocus — le lecteur abonné', () => {
  it('rend la mise en valeur courante et se re-rend quand elle change', () => {
    const store = createPlaybackStore()
    const { result } = renderHook(() => usePlaybackFocus(store))
    expect(result.current).toBeNull()
    act(() => store.focusKill(CIBLE))
    expect(result.current?.entryKey).toBe('k1')
    act(() => store.clearFocus())
    expect(result.current).toBeNull()
  })
})
