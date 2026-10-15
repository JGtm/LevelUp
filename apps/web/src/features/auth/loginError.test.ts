import { describe, it, expect } from 'vitest'
import { isTooManyLoginAttempts } from './loginError'

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
