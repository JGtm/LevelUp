/**
 * _logger.test.ts — Le logger media doit dédupliquer par session (warn + error).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { log } from './_logger'

describe('media logger', () => {
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
    log.warn('hls:unsupported', 'msg')
    log.warn('hls:unsupported', 'msg')
    expect(warnSpy).toHaveBeenCalledTimes(1)
  })

  it('error : logge une seule fois par clé', () => {
    log.error('hls:fatal', 'boom')
    log.error('hls:fatal', 'boom')
    expect(errorSpy).toHaveBeenCalledTimes(1)
  })

  it('warn et error dédupliquent indépendamment', () => {
    log.warn('shared', 'w')
    log.error('shared', 'e')
    expect(warnSpy).toHaveBeenCalledTimes(1)
    expect(errorSpy).toHaveBeenCalledTimes(1)
  })

  it('préfixe les messages avec [media]', () => {
    log.warn('hls:unsupported', 'hello')
    log.error('hls:fatal', 'oops')
    expect(warnSpy).toHaveBeenCalledWith('[media] hello')
    expect(errorSpy).toHaveBeenCalledWith('[media] oops')
  })

  it('passe les args supplémentaires à console', () => {
    log.error('hls:fatal', 'with extra', { details: 'bufferStalledError' })
    expect(errorSpy).toHaveBeenCalledWith('[media] with extra', { details: 'bufferStalledError' })
  })

  it('_resetForTests permet de logger à nouveau la même clé', () => {
    log.error('hls:fatal', 'one')
    log._resetForTests()
    log.error('hls:fatal', 'two')
    expect(errorSpy).toHaveBeenCalledTimes(2)
  })
})
