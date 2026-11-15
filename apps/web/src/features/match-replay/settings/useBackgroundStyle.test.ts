/**
 * useBackgroundStyle.test.ts — LE STYLE DU FOND (rendu ou plan) : défaut, lecture tolérante,
 * persistance, et partage entre deux composants qui lisent la même clé.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { BACKGROUND_STYLE_KEY, readBackgroundStyle } from './replayPreferences'
import { useBackgroundStyle } from './useBackgroundStyle'

beforeEach(() => window.localStorage.clear())

describe('useBackgroundStyle', () => {
  it('par défaut : rendu', () => {
    const { result } = renderHook(() => useBackgroundStyle())
    expect(result.current[0]).toBe('rendu')
    expect(readBackgroundStyle()).toBe('rendu')
  })

  it('la clé est replay-background-style', () => {
    expect(BACKGROUND_STYLE_KEY).toBe('replay-background-style')
  })

  it('lit un choix plan déjà retenu', () => {
    window.localStorage.setItem(BACKGROUND_STYLE_KEY, 'plan')
    expect(renderHook(() => useBackgroundStyle()).result.current[0]).toBe('plan')
  })

  it('une valeur inconnue vaut rendu, sans rien effacer', () => {
    window.localStorage.setItem(BACKGROUND_STYLE_KEY, 'n-importe-quoi')
    expect(renderHook(() => useBackgroundStyle()).result.current[0]).toBe('rendu')
    expect(readBackgroundStyle()).toBe('rendu')
    expect(window.localStorage.getItem(BACKGROUND_STYLE_KEY)).toBe('n-importe-quoi')
  })

  it('choisir un style le retient dans le navigateur', () => {
    const { result } = renderHook(() => useBackgroundStyle())
    act(() => result.current[1]('plan'))
    expect(result.current[0]).toBe('plan')
    expect(window.localStorage.getItem(BACKGROUND_STYLE_KEY)).toBe('plan')
    act(() => result.current[1]('rendu'))
    expect(window.localStorage.getItem(BACKGROUND_STYLE_KEY)).toBe('rendu')
  })

  it('deux lecteurs de la clé bougent ensemble (le panneau écrit, la page lit)', () => {
    const ecrivain = renderHook(() => useBackgroundStyle())
    const lecteur = renderHook(() => useBackgroundStyle())
    act(() => ecrivain.result.current[1]('plan'))
    expect(lecteur.result.current[0]).toBe('plan')
  })

  it('stockage refusé : le choix vaut pour la session, sans erreur', () => {
    const { result } = renderHook(() => useBackgroundStyle())
    const refus = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('refusé')
    })
    try {
      act(() => result.current[1]('plan'))
    } finally {
      refus.mockRestore()
    }
    expect(result.current[0]).toBe('plan')
  })
})
