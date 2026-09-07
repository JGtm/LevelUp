/**
 * replayPreferences.test.ts — le patron localStorage partagé (né dans useReplaySound,
 * centralisé pour le tiroir de réglages) : lecture avec repli, écriture qui ne plante
 * jamais même si le storage refuse.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { persistPreference, readStoredFlag, readStoredNumber } from './replayPreferences'

describe('readStoredFlag', () => {
  it('rend le fallback quand rien n est stocké', () => {
    expect(readStoredFlag('rp-flag-absent', true)).toBe(true)
    expect(readStoredFlag('rp-flag-absent', false)).toBe(false)
  })

  it('lit exactement ce qui a été persisté', () => {
    persistPreference('rp-flag', 'true')
    expect(readStoredFlag('rp-flag', false)).toBe(true)
    persistPreference('rp-flag', 'false')
    expect(readStoredFlag('rp-flag', true)).toBe(false)
  })

  it('une valeur qui n est ni "true" ni "false" retombe sur false, jamais une erreur', () => {
    persistPreference('rp-flag-garbage', 'garbage')
    expect(readStoredFlag('rp-flag-garbage', true)).toBe(false)
  })
})

describe('readStoredNumber', () => {
  it('rend le fallback quand rien n est stocké', () => {
    expect(readStoredNumber('rp-num-absent', 1, (v) => v > 0)).toBe(1)
  })

  it('rend le fallback quand la valeur stockée échoue la validation', () => {
    persistPreference('rp-num-invalid', '-5')
    expect(readStoredNumber('rp-num-invalid', 1, (v) => v > 0)).toBe(1)
    persistPreference('rp-num-nan', 'pas-un-nombre')
    expect(readStoredNumber('rp-num-nan', 1, () => true)).toBe(1)
  })

  it('lit une valeur valide', () => {
    persistPreference('rp-num-ok', '2')
    expect(readStoredNumber('rp-num-ok', 1, (v) => [1, 2, 4].includes(v))).toBe(2)
  })
})

describe('storage indisponible (navigation privée) — fail-open, jamais une erreur', () => {
  let originalGetItem: typeof window.localStorage.getItem
  let originalSetItem: typeof window.localStorage.setItem

  beforeEach(() => {
    originalGetItem = window.localStorage.getItem
    originalSetItem = window.localStorage.setItem
    window.localStorage.getItem = vi.fn(() => {
      throw new Error('SecurityError')
    })
    window.localStorage.setItem = vi.fn(() => {
      throw new Error('QuotaExceededError')
    })
  })

  afterEach(() => {
    window.localStorage.getItem = originalGetItem
    window.localStorage.setItem = originalSetItem
  })

  it('readStoredFlag -> fallback', () => {
    expect(readStoredFlag('k', true)).toBe(true)
  })

  it('readStoredNumber -> fallback', () => {
    expect(readStoredNumber('k', 3, () => true)).toBe(3)
  })

  it('persistPreference ne lève rien', () => {
    expect(() => persistPreference('k', 'v')).not.toThrow()
  })
})
