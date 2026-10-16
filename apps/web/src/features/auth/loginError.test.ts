import { describe, it, expect } from 'vitest'
import {
  deviceFlowFailureKey,
  isTooManyLoginAttempts,
  oauthErrorKey,
  registerErrorKey,
  setPasswordErrorKey,
} from './loginError'

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

  it('rend la clé de la règle de longueur refusée', () => {
    expect(setPasswordErrorKey({ status: 400, code: 'password_too_short' })).toBe('common.auth.password_too_short')
    expect(setPasswordErrorKey({ status: 400, code: 'password_too_long' })).toBe('common.auth.password_too_long')
  })
})

describe('registerErrorKey', () => {
  it.each([
    ['user_exists', 'common.auth.register_err_user_exists'],
    ['invite_required', 'common.auth.register_err_invite_required'],
    ['invalid_invite', 'common.auth.register_err_invalid_invite'],
    ['registration_closed', 'common.auth.register_err_closed'],
    ['instance_locked', 'common.auth.register_instance_closed'],
    ['invalid_username', 'common.auth.register_err_invalid_username'],
    ['password_too_short', 'common.auth.password_too_short'],
    ['password_too_long', 'common.auth.password_too_long'],
  ])('%s → %s', (code, key) => {
    expect(registerErrorKey({ status: 400, code, message: 'texte du serveur' })).toBe(key)
  })

  it('code inconnu, erreur serveur ou réseau : message générique, jamais le texte du serveur', () => {
    expect(registerErrorKey({ status: 400, code: 'code_inconnu', message: 'SELECT secret' })).toBe(
      'common.auth.register_err_generic',
    )
    expect(registerErrorKey({ status: 500, code: 'create_error', message: 'internal error' })).toBe(
      'common.auth.register_err_generic',
    )
    expect(registerErrorKey(new Error('réseau'))).toBe('common.auth.register_err_generic')
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
    ['redirect_uri_not_configured', 'common.auth.oauth_failed'],
    ['state_gen_failed', 'common.auth.oauth_failed'],
    ['pkce_gen_failed', 'common.auth.oauth_failed'],
    ['session_save_failed', 'common.auth.oauth_failed'],
    ['code-inconnu', 'common.auth.oauth_failed'],
  ])('%s → %s', (code, key) => {
    expect(oauthErrorKey(code)).toBe(key)
  })

  it.each([undefined, '', 42, null])('aucun refus pour %s', (code) => {
    expect(oauthErrorKey(code)).toBeNull()
  })
})

describe('deviceFlowFailureKey', () => {
  it.each([
    ['expiré', 'expired', null, 'common.device_code.err_expired'],
    ["code expiré malgré un code d'erreur", 'expired', 'halo_exchange_error', 'common.device_code.err_expired'],
    ['échange Halo', 'failed', 'halo_exchange_error', 'common.device_code.err_halo'],
    ['attente Microsoft', 'failed', 'device_flow_acquire_error', 'common.device_code.err_ms'],
    ['identité absente', 'failed', 'identity_missing', 'common.device_code.err_identity'],
    ['instance verrouillée', 'failed', 'instance_locked', 'common.auth.register_instance_closed'],
    ['invitation périmée', 'failed', 'invite_invalid', 'common.auth.oauth_invite_invalid'],
    ['code inconnu', 'failed', 'code_inconnu', 'common.device_code.err_generic'],
    ['aucun code', 'failed', null, 'common.device_code.err_generic'],
  ])('%s', (_label, status, code, expected) => {
    expect(deviceFlowFailureKey(status, code)).toBe(expected)
  })
})
