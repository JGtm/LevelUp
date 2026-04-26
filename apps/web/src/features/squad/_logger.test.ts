/**
 * _logger.test.ts — Le logger Squad doit dédupliquer par session.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { log } from './_logger'

describe('squad logger', () => {
  let warnSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    log._resetForTests()
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

  afterEach(() => {
    warnSpy.mockRestore()
  })

  it('logge une seule fois par clé', () => {
    log.warn('k1', 'msg 1')
    log.warn('k1', 'msg 1')
    log.warn('k1', 'msg 1')
    expect(warnSpy).toHaveBeenCalledTimes(1)
  })

  it('logge des clés différentes séparément', () => {
    log.warn('k1', 'msg 1')
    log.warn('k2', 'msg 2')
    expect(warnSpy).toHaveBeenCalledTimes(2)
  })

  it('préfixe les messages avec [squad]', () => {
    log.warn('k1', 'hello')
    expect(warnSpy).toHaveBeenCalledWith('[squad] hello')
  })

  it('passe les args supplémentaires à console.warn', () => {
    log.warn('k1', 'with extra', { foo: 'bar' })
    expect(warnSpy).toHaveBeenCalledWith('[squad] with extra', { foo: 'bar' })
  })

  it('_resetForTests permet de logger à nouveau la même clé', () => {
    log.warn('k1', 'one')
    log._resetForTests()
    log.warn('k1', 'two')
    expect(warnSpy).toHaveBeenCalledTimes(2)
  })
})
