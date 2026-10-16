/**
 * loginError — lecture des refus des routes d'authentification par mot de passe
 * (POST /auth/login, POST /auth/password) et des refus du callback de connexion Xbox
 * (/login?oauth_error=<code>).
 */

/**
 * Statut d'un refus pour excès de tentatives. Deux couches le rendent : le limiteur
 * d'échecs de connexion de l'API (corps JSON, code `too_many_login_attempts`) et la
 * limite de débit de nginx (corps HTML, donc sans code exploitable).
 */
const TOO_MANY_REQUESTS = 429

/**
 * Vrai quand la requête a été refusée pour excès de tentatives, quelle que soit la
 * couche qui refuse : l'écran affiche alors son propre message localisé, jamais le
 * texte du serveur.
 */
export function isTooManyLoginAttempts(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    'status' in err &&
    (err as { status?: unknown }).status === TOO_MANY_REQUESTS
  )
}

/** Clés i18n des messages de refus de POST /auth/password. */
export type SetPasswordErrorKey =
  | 'common.auth.too_many_attempts'
  | 'common.auth.current_password_invalid'
  | 'common.auth.current_password_required'
  | 'common.auth.set_password_failed'

/**
 * Message localisé d'un refus de changement de mot de passe : excès de tentatives,
 * mot de passe actuel faux ou manquant, sinon un échec générique (jamais le texte
 * brut du serveur).
 */
export function setPasswordErrorKey(err: unknown): SetPasswordErrorKey {
  if (isTooManyLoginAttempts(err)) return 'common.auth.too_many_attempts'
  const code =
    typeof err === 'object' && err !== null && 'code' in err ? (err as { code?: unknown }).code : undefined
  if (code === 'invalid_current_password') return 'common.auth.current_password_invalid'
  if (code === 'current_password_required') return 'common.auth.current_password_required'
  return 'common.auth.set_password_failed'
}

/** Clés i18n des échecs du device-flow (statuts `failed` / `expired` de la tentative). */
export type DeviceFlowFailureKey =
  | 'common.device_code.err_expired'
  | 'common.device_code.err_halo'
  | 'common.device_code.err_ms'
  | 'common.device_code.err_identity'
  | 'common.device_code.err_generic'
  | OAuthErrorKey

/** Code d'échec posé par le serveur (`error_code`) → message de l'écran. */
const DEVICE_FLOW_FAILURE_KEYS: Readonly<Record<string, DeviceFlowFailureKey>> = {
  halo_exchange_error: 'common.device_code.err_halo',
  device_flow_acquire_error: 'common.device_code.err_ms',
  identity_missing: 'common.device_code.err_identity',
}

/**
 * Message localisé d'un échec du device-flow. Le serveur ne renvoie que le statut et un code
 * machine : le texte de l'écran vient toujours du manifeste, jamais d'un champ du serveur.
 * Un code inconnu reçoit le message générique.
 */
export function deviceFlowFailureKey(status: string, errorCode: string | null | undefined): DeviceFlowFailureKey {
  if (status === 'expired') return 'common.device_code.err_expired'
  if (errorCode === 'instance_locked' || errorCode === 'invite_invalid') {
    return oauthErrorKey(errorCode) ?? 'common.auth.oauth_failed'
  }
  if (errorCode && Object.hasOwn(DEVICE_FLOW_FAILURE_KEYS, errorCode)) return DEVICE_FLOW_FAILURE_KEYS[errorCode]
  return 'common.device_code.err_generic'
}

/** Clés i18n des refus du callback de connexion Xbox. */
export type OAuthErrorKey =
  | 'common.auth.register_instance_closed'
  | 'common.auth.oauth_invite_invalid'
  | 'common.auth.oauth_denied'
  | 'common.auth.oauth_interrupted'
  | 'common.auth.oauth_exchange_failed'
  | 'common.auth.oauth_failed'

/** Code de refus posé par le serveur dans `?oauth_error=` → message affiché. */
const OAUTH_ERROR_KEYS: Readonly<Record<string, OAuthErrorKey>> = {
  instance_locked: 'common.auth.register_instance_closed',
  invite_invalid: 'common.auth.oauth_invite_invalid',
  oauth_denied: 'common.auth.oauth_denied',
  state_mismatch: 'common.auth.oauth_interrupted',
  missing_params: 'common.auth.oauth_interrupted',
  no_session: 'common.auth.oauth_interrupted',
  code_exchange_failed: 'common.auth.oauth_exchange_failed',
  halo_exchange_failed: 'common.auth.oauth_exchange_failed',
}

/**
 * Message localisé d'un refus du callback de connexion Xbox, `null` sans refus. Un code
 * inconnu reçoit le message générique, jamais le code brut.
 */
export function oauthErrorKey(code: unknown): OAuthErrorKey | null {
  if (typeof code !== 'string' || code === '') return null
  return Object.hasOwn(OAUTH_ERROR_KEYS, code) ? OAUTH_ERROR_KEYS[code] : 'common.auth.oauth_failed'
}
