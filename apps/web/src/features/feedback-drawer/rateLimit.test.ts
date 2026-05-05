import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import {
  MAX_SUBMITS_PER_HOUR,
  _resetSubmitsForTests,
  getRemainingSubmits,
  recordSubmit,
} from './rateLimit'

beforeEach(() => {
  _resetSubmitsForTests()
})

describe('rateLimit — happy path', () => {
  it('5 submits dans la fenêtre = autorisés, le 6e refusé', () => {
    const now = 1_000_000
    for (let i = 0; i < MAX_SUBMITS_PER_HOUR; i++) {
      expect(recordSubmit(now + i)).toBe(true)
    }
    expect(recordSubmit(now + MAX_SUBMITS_PER_HOUR)).toBe(false)
  })

  it('getRemainingSubmits décroît à chaque submit', () => {
    const now = 1_000_000
    expect(getRemainingSubmits(now)).toBe(5)
    recordSubmit(now)
    expect(getRemainingSubmits(now)).toBe(4)
    recordSubmit(now + 1)
    expect(getRemainingSubmits(now + 1)).toBe(3)
  })

  it('expire les submits > 1h', () => {
    const now = 1_000_000
    recordSubmit(now)
    recordSubmit(now)
    expect(getRemainingSubmits(now)).toBe(3)
    // 1h05 plus tard
    const later = now + 65 * 60 * 1000
    expect(getRemainingSubmits(later)).toBe(5)
    expect(recordSubmit(later)).toBe(true)
  })
})

describe('rateLimit — fail-open localStorage indisponible', () => {
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

  it('getRemainingSubmits → MAX (fail-open)', () => {
    expect(getRemainingSubmits()).toBe(MAX_SUBMITS_PER_HOUR)
  })

  it('recordSubmit autorise (fail-open)', () => {
    expect(recordSubmit()).toBe(true)
  })
})

describe('rateLimit — données corrompues en localStorage', () => {
  it('payload non-JSON → traité comme vide', () => {
    window.localStorage.setItem('levelup-feedback-submits', '<garbage>')
    expect(getRemainingSubmits()).toBe(MAX_SUBMITS_PER_HOUR)
  })

  it('payload non-array → traité comme vide', () => {
    window.localStorage.setItem('levelup-feedback-submits', '{"foo":1}')
    expect(getRemainingSubmits()).toBe(MAX_SUBMITS_PER_HOUR)
  })

  it('payload mixte → ne garde que les nombres', () => {
    window.localStorage.setItem(
      'levelup-feedback-submits',
      JSON.stringify([Date.now(), 'oops', null, Date.now()]),
    )
    expect(getRemainingSubmits()).toBe(MAX_SUBMITS_PER_HOUR - 2)
  })
})
