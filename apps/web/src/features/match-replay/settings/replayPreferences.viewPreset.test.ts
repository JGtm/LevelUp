/**
 * Tests — le préréglage de vue mémorisé (lot « bouton Vue ») : valeur validée à la lecture,
 * jamais d'erreur quand le stockage est refusé. Il ne sert qu'à départager deux préréglages
 * égaux sur un match ; l'état des calques reste la vérité.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  VIEW_PRESET_KEY,
  persistPreference,
  persistViewPreset,
  readViewPreset,
} from './replayPreferences'

describe('le préréglage de vue mémorisé', () => {
  beforeEach(() => {
    window.localStorage.removeItem(VIEW_PRESET_KEY)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('est absent quand rien n’est retenu', () => {
    expect(readViewPreset()).toBeUndefined()
  })

  it('relit le préréglage retenu', () => {
    persistViewPreset('combat')
    expect(readViewPreset()).toBe('combat')
  })

  it.each(['', 'inconnu', 'COMPLET', 'personnalise', 'combat '])(
    'une valeur hors liste (%j) est ignorée',
    (brut) => {
      persistPreference(VIEW_PRESET_KEY, brut)
      expect(readViewPreset()).toBeUndefined()
    },
  )

  it('stockage refusé : absent, et persister ne lève rien', () => {
    vi.spyOn(window.localStorage.__proto__, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError')
    })
    vi.spyOn(window.localStorage.__proto__, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    expect(readViewPreset()).toBeUndefined()
    expect(() => persistViewPreset('objectif')).not.toThrow()
  })
})
