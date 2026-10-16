import { describe, it, expect } from 'vitest'
import { isTooManyLoginAttempts, setPasswordErrorKey } from './loginError'

describe('isTooManyLoginAttempts', () => {
  it('reconnaît le refus du limiteur de l’API', () => {
    expect(
      isTooManyLoginAttempts({ code: 'too_many_login_attempts', message: 'x', retryable: false, status: 429 }),
    ).toBe(true)
  })

  it('reconnaît le refus de nginx (corps HTML, sans code)', () => {
    expect(
      isTooManyLoginAttempts({ code: 'unknown_error', message: 'Erreur HTTP 429', retryable: false, status: 429 }),
    ).toBe(true)
  })

  it.each([
    ['identifiants incorrects', { code: 'invalid_credentials', status: 401 }],
    ['erreur serveur', { code: 'auth_error', status: 500 }],
    ['erreur réseau', new Error('Failed to fetch')],
    ['valeur nulle', null],
    ['statut en chaîne', { status: '429' }],
  ])('ignore %s', (_label, err) => {
    expect(isTooManyLoginAttempts(err)).toBe(false)
  })
})

describe('setPasswordErrorKey', () => {
  it('rend la clé du refus, jamais le texte du serveur', () => {
    expect(setPasswordErrorKey({ status: 429 })).toBe('common.auth.too_many_attempts')
    expect(setPasswordErrorKey({ status: 403, code: 'invalid_current_password' })).toBe('common.auth.current_password_invalid')
    expect(setPasswordErrorKey({ status: 400, code: 'current_password_required' })).toBe('common.auth.current_password_required')
    expect(setPasswordErrorKey({ status: 500, code: 'set_password_error', message: 'internal error' })).toBe(
      'common.auth.set_password_failed',
    )
    expect(setPasswordErrorKey(new Error('réseau'))).toBe('common.auth.set_password_failed')
  })
})
