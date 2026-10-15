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
