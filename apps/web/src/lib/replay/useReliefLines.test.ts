/**
 * Tests — useReliefLines : l'interrupteur « Traits du relief », UNE valeur pour les trois fonds.
 *
 * Ce qu'ils protègent : (1) le défaut, allumé ; (2) un geste se retient dans la clé unique et se relit ; (3) deux
 * lecteurs de la même clé bougent ensemble ; (4) la reprise des clés d'avant (une par fond) : le choix du fond
 * retenu prime, puis Relief, Plan, Rendu ; sans choix d'avant, rien n'est écrit ; une valeur étrangère est écartée ;
 * la reprise est idempotente.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { adoptLegacyReliefLines, readReliefLines, RELIEF_LINES_KEY, useReliefLines } from './useReliefLines'

const set = (key: string, value: string) => window.localStorage.setItem(key, value)
const get = (key: string) => window.localStorage.getItem(key)

beforeEach(() => {
  window.localStorage.clear()
})

describe('useReliefLines — la valeur unique', () => {
  it('allumés d’office, sous la clé replay-relief-lines', () => {
    expect(RELIEF_LINES_KEY).toBe('replay-relief-lines')
    expect(renderHook(() => useReliefLines()).result.current[0]).toBe(true)
    expect(get(RELIEF_LINES_KEY)).toBeNull()
  })

  it('éteindre se retient pour tous les fonds et se relit au prochain montage', () => {
    const h = renderHook(() => useReliefLines())
    act(() => h.result.current[1]())
    expect(h.result.current[0]).toBe(false)
    expect(get(RELIEF_LINES_KEY)).toBe('false')
    expect(renderHook(() => useReliefLines()).result.current[0]).toBe(false)
  })

  it('une valeur écrite hors liste vaut le défaut', () => {
    set(RELIEF_LINES_KEY, 'peut-etre')
    expect(readReliefLines()).toBe(false)
  })

  it('deux lecteurs de la même clé bougent ensemble (le panneau écrit, la Tactique lit)', () => {
    const a = renderHook(() => useReliefLines())
    const b = renderHook(() => useReliefLines())
    act(() => a.result.current[1]())
    expect(b.result.current[0]).toBe(false)
  })
})

describe('useReliefLines — la reprise des clés d’avant', () => {
  it('reprend le choix du fond retenu, avant celui des autres', () => {
    set('replay-background-style', 'plan')
    set('replay-relief-lines.plan', 'true')
    set('replay-relief-lines.relief', 'false')
    expect(readReliefLines()).toBe(true)
    expect(get(RELIEF_LINES_KEY)).toBe('true')
  })

  it('sans choix pour le fond retenu : le Relief, puis le Plan, puis le Rendu', () => {
    set('replay-background-style', 'rendu')
    set('replay-relief-lines.relief', 'false')
    set('replay-relief-lines.plan', 'true')
    expect(readReliefLines()).toBe(false)
    window.localStorage.clear()
    set('replay-relief-lines.plan', 'false')
    set('replay-relief-lines.rendu', 'true')
    expect(readReliefLines()).toBe(false)
  })

  it('efface les trois clés d’avant une fois reprises', () => {
    set('replay-relief-lines.relief', 'false')
    set('replay-relief-lines.plan', 'true')
    set('replay-relief-lines.rendu', 'true')
    adoptLegacyReliefLines()
    expect(get('replay-relief-lines.relief')).toBeNull()
    expect(get('replay-relief-lines.plan')).toBeNull()
    expect(get('replay-relief-lines.rendu')).toBeNull()
  })

  it('sans clé d’avant : rien n’est écrit ; avec la clé unique déjà posée : elle prime', () => {
    adoptLegacyReliefLines()
    expect(get(RELIEF_LINES_KEY)).toBeNull()
    set(RELIEF_LINES_KEY, 'false')
    set('replay-relief-lines.relief', 'true')
    adoptLegacyReliefLines()
    expect(get(RELIEF_LINES_KEY)).toBe('false')
    expect(get('replay-relief-lines.relief')).toBeNull()
  })

  it('une valeur d’avant étrangère est écartée sans rien écrire (les traits restent allumés)', () => {
    set('replay-relief-lines.relief', 'peut-etre')
    expect(readReliefLines()).toBe(true)
    expect(get(RELIEF_LINES_KEY)).toBeNull()
    expect(get('replay-relief-lines.relief')).toBeNull()
  })

  it('est idempotente : un second passage ne change rien', () => {
    set('replay-relief-lines.plan', 'false')
    adoptLegacyReliefLines()
    const après = get(RELIEF_LINES_KEY)
    adoptLegacyReliefLines()
    expect(get(RELIEF_LINES_KEY)).toBe(après)
    expect(après).toBe('false')
  })

  it('le hook reprend aussi à son premier rendu', () => {
    set('replay-relief-lines.relief', 'false')
    expect(renderHook(() => useReliefLines()).result.current[0]).toBe(false)
  })
})
