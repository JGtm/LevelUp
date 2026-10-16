/**
 * loginError — lecture des refus des routes d'authentification par mot de passe
 * (POST /auth/login, POST /auth/password).
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
