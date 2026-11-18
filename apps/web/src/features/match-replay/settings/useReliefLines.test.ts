/**
 * Tests — useReliefLines : l'interrupteur « Traits du relief », retenu PAR FOND.
 *
 * Ce qu'ils protègent : (1) le défaut de chaque fond — allumé sur le relief, éteint sur le plan et le rendu ;
 * (2) un geste manuel se retient pour CE fond sans toucher aux deux autres ; (3) changer de fond relit la
 * valeur du nouveau fond ; (4) deux lecteurs de la même clé bougent ensemble.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import { readReliefLines, reliefLinesDefault, reliefLinesKey } from './replayPreferences'
import { useReliefLines } from './useReliefLines'

beforeEach(() => {
  window.localStorage.clear()
})

describe('useReliefLines — défauts par fond', () => {
  it('allumés d’office sur le relief, éteints sur le plan et le rendu', () => {
    expect(reliefLinesDefault('relief')).toBe(true)
    expect(reliefLinesDefault('plan')).toBe(false)
    expect(reliefLinesDefault('rendu')).toBe(false)
    expect(renderHook(() => useReliefLines('relief')).result.current[0]).toBe(true)
    expect(renderHook(() => useReliefLines('plan')).result.current[0]).toBe(false)
    expect(renderHook(() => useReliefLines('rendu')).result.current[0]).toBe(false)
  })

  it('une clé par fond, sous replay-relief-lines.<fond>', () => {
    expect(reliefLinesKey('relief')).toBe('replay-relief-lines.relief')
    expect(reliefLinesKey('plan')).toBe('replay-relief-lines.plan')
  })
})

describe('useReliefLines — un geste est retenu pour son fond', () => {
  it('éteindre les traits sur le relief ne touche ni le plan ni le rendu', () => {
    const relief = renderHook(() => useReliefLines('relief'))
    act(() => relief.result.current[1]())
    expect(relief.result.current[0]).toBe(false)
    expect(window.localStorage.getItem('replay-relief-lines.relief')).toBe('false')
    expect(readReliefLines('plan')).toBe(false)
    expect(readReliefLines('rendu')).toBe(false)
  })

  it('allumer les traits sur le plan se retient pour le plan seul, et se relit au prochain montage', () => {
    const plan = renderHook(() => useReliefLines('plan'))
    act(() => plan.result.current[1]())
    expect(plan.result.current[0]).toBe(true)
    expect(renderHook(() => useReliefLines('plan')).result.current[0]).toBe(true)
    expect(renderHook(() => useReliefLines('rendu')).result.current[0]).toBe(false)
    expect(renderHook(() => useReliefLines('relief')).result.current[0]).toBe(true)
  })

  it('une valeur écrite hors liste vaut le défaut du fond', () => {
    window.localStorage.setItem('replay-relief-lines.plan', 'peut-etre')
    expect(readReliefLines('plan')).toBe(false)
  })
})

describe('useReliefLines — changer de fond', () => {
  it('relit la valeur du nouveau fond au même rendu, sans repasser par celle de l’ancien', () => {
    window.localStorage.setItem('replay-relief-lines.plan', 'true')
    const h = renderHook(({ fond }: { fond: MapBackgroundVariant }) => useReliefLines(fond), {
      initialProps: { fond: 'rendu' as MapBackgroundVariant },
    })
    expect(h.result.current[0]).toBe(false)
    h.rerender({ fond: 'plan' })
    expect(h.result.current[0]).toBe(true)
    h.rerender({ fond: 'relief' })
    expect(h.result.current[0]).toBe(true)
    h.rerender({ fond: 'rendu' })
    expect(h.result.current[0]).toBe(false)
  })

  it('le geste suit le fond COURANT : basculer sur le rendu n’écrit pas la clé du relief', () => {
    const h = renderHook(({ fond }: { fond: MapBackgroundVariant }) => useReliefLines(fond), {
      initialProps: { fond: 'relief' as MapBackgroundVariant },
    })
    h.rerender({ fond: 'rendu' })
    act(() => h.result.current[1]())
    expect(window.localStorage.getItem('replay-relief-lines.rendu')).toBe('true')
    expect(window.localStorage.getItem('replay-relief-lines.relief')).toBeNull()
  })
})

describe('useReliefLines — deux lecteurs de la même clé', () => {
  it('bougent ensemble (le panneau écrit, le canvas lit)', () => {
    const a = renderHook(() => useReliefLines('plan'))
    const b = renderHook(() => useReliefLines('plan'))
    act(() => a.result.current[1]())
    expect(b.result.current[0]).toBe(true)
  })
})
