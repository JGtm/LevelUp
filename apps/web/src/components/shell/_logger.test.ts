/**
 * _logger.test.ts — Le logger shell-nav doit dédupliquer par session (warn + error).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { log } from './_logger'

describe('shell-nav logger', () => {
  let warnSpy: ReturnType<typeof vi.spyOn>
  let errorSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    log._resetForTests()
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    warnSpy.mockRestore()
    errorSpy.mockRestore()
  })

  it('warn : logge une seule fois par clé', () => {
    log.warn('k1', 'msg 1')
    log.warn('k1', 'msg 1')
    expect(warnSpy).toHaveBeenCalledTimes(1)
  })

  it('error : logge une seule fois par clé', () => {
    log.error('e1', 'boom')
    log.error('e1', 'boom')
    expect(errorSpy).toHaveBeenCalledTimes(1)
  })

  it('warn et error dédupliquent indépendamment', () => {
    log.warn('shared', 'w')
    log.error('shared', 'e')
    expect(warnSpy).toHaveBeenCalledTimes(1)
    expect(errorSpy).toHaveBeenCalledTimes(1)
  })

  it('préfixe les messages avec [shell-nav]', () => {
    log.warn('k1', 'hello')
    log.error('e1', 'oops')
    expect(warnSpy).toHaveBeenCalledWith('[shell-nav] hello')
    expect(errorSpy).toHaveBeenCalledWith('[shell-nav] oops')
  })

  it('passe les args supplémentaires à console', () => {
    log.error('e1', 'with extra', { foo: 'bar' })
    expect(errorSpy).toHaveBeenCalledWith('[shell-nav] with extra', { foo: 'bar' })
  })

  it('_resetForTests permet de logger à nouveau la même clé', () => {
    log.error('e1', 'one')
    log._resetForTests()
    log.error('e1', 'two')
    expect(errorSpy).toHaveBeenCalledTimes(2)
  })
})
