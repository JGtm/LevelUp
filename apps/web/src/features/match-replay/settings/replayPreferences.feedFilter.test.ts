/**
 * Tests — la préférence du filtre du fil (D-4) : valeur validée à la lecture, jamais
 * d'erreur quand le stockage est refusé.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { FEED_FILTER_KEY, persistFeedFilter, persistPreference, readFeedFilter } from './replayPreferences'

describe('le filtre du fil des éliminations', () => {
  beforeEach(() => {
    window.localStorage.removeItem(FEED_FILTER_KEY)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('vaut Tout quand rien n’est retenu', () => {
    expect(readFeedFilter()).toBe('tout')
  })

  it('relit le filtre retenu', () => {
    persistFeedFilter('objectives')
    expect(readFeedFilter()).toBe('objectives')
  })

  it.each(['', 'inconnu', 'TOUT', 'kills '])(
    'une valeur hors liste (%j) retombe sur Tout',
    (brut) => {
      persistPreference(FEED_FILTER_KEY, brut)
      expect(readFeedFilter()).toBe('tout')
    },
  )

  it('stockage refusé : Tout, et persister ne lève rien', () => {
    vi.spyOn(window.localStorage.__proto__, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError')
    })
    vi.spyOn(window.localStorage.__proto__, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    expect(readFeedFilter()).toBe('tout')
    expect(() => persistFeedFilter('medals')).not.toThrow()
  })
})
