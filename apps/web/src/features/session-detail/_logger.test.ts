import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { log } from './_logger'

describe('session-detail _logger', () => {
  beforeEach(() => log._resetForTests())
  afterEach(() => vi.restoreAllMocks())

  it('émet un console.warn préfixé', () => {
    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    log.warn('k', 'mon message', { a: 1 })
    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy.mock.calls[0][0]).toContain('[session-detail]')
    expect(spy.mock.calls[0][0]).toContain('mon message')
  })

  it('dédupe par clé (une fois par session)', () => {
    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    log.warn('mmr_missing:S1', 'a')
    log.warn('mmr_missing:S1', 'a encore') // même clé → ignoré
    log.warn('mmr_missing:S2', 'autre session') // clé différente → loggé
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('_resetForTests réarme la déduplication', () => {
    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    log.warn('k', 'a')
    log._resetForTests()
    log.warn('k', 'a')
    expect(spy).toHaveBeenCalledTimes(2)
  })
})
