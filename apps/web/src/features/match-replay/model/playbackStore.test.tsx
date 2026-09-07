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

import { createPlaybackStore, usePlaybackFrame, usePlaybackStore } from './playbackStore'

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
