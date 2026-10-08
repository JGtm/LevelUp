/**
 * Attente d'un serveur qui démarre — décision de rejeu du /bootstrap et libellé d'étape.
 *
 * Le serveur Go écoute avant d'avoir ouvert ses bases : pendant son initialisation, il
 * répond 503 `server_starting` (étape en cours dans `details.step`). Avant même qu'il
 * écoute (compilation par air, redémarrage), la requête échoue sans réponse (erreur
 * réseau) ou sur un 502 du proxy (Vite en dev, nginx en prod). Ces trois cas sont « le
 * serveur démarre » : la page réinterroge chaque seconde, sans compteur, jusqu'à
 * SERVER_STARTUP_MAX_WAIT_MS ; au-delà, l'écran « API injoignable ». Toute autre erreur
 * du /bootstrap garde quelques rejeux espacés (BOOTSTRAP_OTHER_ERROR_RETRIES).
 */
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

/** Intervalle de réinterrogation pendant le démarrage du serveur. */
export const SERVER_STARTUP_POLL_MS = 1000

/** Attente maximale d'un serveur qui démarre (couvre une compilation par air). */
export const SERVER_STARTUP_MAX_WAIT_MS = 120_000

/** Nombre de réinterrogations correspondant à SERVER_STARTUP_MAX_WAIT_MS. */
export const SERVER_STARTUP_MAX_RETRIES = Math.ceil(SERVER_STARTUP_MAX_WAIT_MS / SERVER_STARTUP_POLL_MS)

/** Code d'erreur de la réponse 503 d'un serveur en cours d'initialisation. */
export const SERVER_STARTING_CODE = 'server_starting'

/**
 * Rejeux du /bootstrap sur une autre erreur (500 bootstrap_error, base occupée…) : six,
 * espacés de 0,5 s à 4 s (≈ 15 s en tout), avant l'écran « API injoignable ».
 */
export const BOOTSTRAP_OTHER_ERROR_RETRIES = 6
const BOOTSTRAP_OTHER_ERROR_BASE_DELAY_MS = 500
const BOOTSTRAP_OTHER_ERROR_MAX_DELAY_MS = 4000

/** Statut d'un proxy qui ne joint pas le serveur (Vite en dev, nginx en prod). */
const PROXY_NO_SERVER_STATUS = 502

/** Statut de la réponse server_starting. */
const SERVER_STARTING_STATUS = 503

/** Libellé de chaque étape d'initialisation annoncée par le serveur. */
const STEP_MESSAGE_KEYS: Readonly<Record<string, CommonManifestKey>> = {
  migrations: 'common.root.server_step.migrations',
  databases: 'common.root.server_step.databases',
  accounts: 'common.root.server_step.accounts',
  services: 'common.root.server_step.services',
}

interface ErrorShape {
  status?: unknown
  code?: unknown
  details?: unknown
}

function asErrorShape(error: unknown): ErrorShape | null {
  return error != null && typeof error === 'object' ? (error as ErrorShape) : null
}

/**
 * Vrai si l'erreur signifie « le serveur démarre » : échec réseau de fetch (TypeError,
 * aucune réponse), 502 du proxy, ou 503 `server_starting`. Une annulation (AbortError)
 * ou une réponse illisible (SyntaxError) n'en est pas une.
 */
export function isServerStartingError(error: unknown): boolean {
  if (error instanceof TypeError) return true
  const e = asErrorShape(error)
  if (e?.status === PROXY_NO_SERVER_STATUS) return true
  return e?.status === SERVER_STARTING_STATUS && e?.code === SERVER_STARTING_CODE
}

/** Clé du libellé de l'étape annoncée par le serveur ; undefined si absente ou inconnue. */
export function serverStartingStepKey(error: unknown): CommonManifestKey | undefined {
  const details = asErrorShape(error)?.details
  if (details == null || typeof details !== 'object') return undefined
  const step = (details as { step?: unknown }).step
  if (typeof step !== 'string' || !Object.hasOwn(STEP_MESSAGE_KEYS, step)) return undefined
  return STEP_MESSAGE_KEYS[step]
}

/** Rejeu du /bootstrap : chaque seconde tant que le serveur démarre, quelques fois sinon. */
export function shouldRetryBootstrap(failureCount: number, error: unknown): boolean {
  const limit = isServerStartingError(error) ? SERVER_STARTUP_MAX_RETRIES : BOOTSTRAP_OTHER_ERROR_RETRIES
  return failureCount < limit
}

/** Délai avant le rejeu n° attempt (0 = premier rejeu) du /bootstrap. */
export function bootstrapRetryDelay(attempt: number, error: unknown): number {
  if (isServerStartingError(error)) return SERVER_STARTUP_POLL_MS
  return Math.min(BOOTSTRAP_OTHER_ERROR_BASE_DELAY_MS * 2 ** attempt, BOOTSTRAP_OTHER_ERROR_MAX_DELAY_MS)
}
