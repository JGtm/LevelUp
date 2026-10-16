import { describe, it, expect } from 'vitest'
import { isTooManyLoginAttempts, oauthErrorKey, setPasswordErrorKey } from './loginError'

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

describe('oauthErrorKey', () => {
  it.each([
    ['instance_locked', 'common.auth.register_instance_closed'],
    ['invite_invalid', 'common.auth.oauth_invite_invalid'],
    ['oauth_denied', 'common.auth.oauth_denied'],
    ['state_mismatch', 'common.auth.oauth_interrupted'],
    ['missing_params', 'common.auth.oauth_interrupted'],
    ['no_session', 'common.auth.oauth_interrupted'],
    ['code_exchange_failed', 'common.auth.oauth_exchange_failed'],
    ['halo_exchange_failed', 'common.auth.oauth_exchange_failed'],
    ['link_failed', 'common.auth.oauth_failed'],
    ['code-inconnu', 'common.auth.oauth_failed'],
  ])('%s → %s', (code, key) => {
    expect(oauthErrorKey(code)).toBe(key)
  })

  it.each([undefined, '', 42, null])('aucun refus pour %s', (code) => {
    expect(oauthErrorKey(code)).toBeNull()
  })
})
