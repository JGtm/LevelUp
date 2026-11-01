/**
 * Attente d'un serveur qui démarre — décision de rejeu du /bootstrap et libellé d'étape.
 *
 * Le serveur Go écoute avant d'avoir ouvert ses bases : pendant son initialisation, il
 * répond 503 `server_starting` (étape en cours dans `details.step`). Avant même qu'il
 * écoute (compilation par air, redémarrage), la requête échoue sans réponse (erreur
 * réseau) ou sur un 502 du proxy (Vite en dev, nginx en prod). Ces trois cas sont « le
 * serveur démarre » : la page réinterroge chaque seconde, sans compteur, jusqu'à
 * SERVER_STARTUP_MAX_WAIT_MS ; au-delà, l'écran « API injoignable ». Toute autre erreur
 * du /bootstrap garde quelques rejeux espacés (BOOTSTRAP_OTHER_ERROR_RETRIES). Cf.
 * createBootstrapRetryPolicy.
 *
 * Un démarrage qui ne peut pas aboutir (migration en échec, base tenue par un autre
 * processus, schéma plus récent que le binaire) est annoncé par un 503 `server_start_failed`
 * NON réessayable (`retryable: false`, état dans `details.state`) : la page cesse
 * d'interroger et affiche l'état (serverStartFailure). Tant que le serveur annonce l'étape
 * `migrations`, l'attente maximale repart à zéro à chaque réponse : une migration longue
 * progresse, elle n'est pas un serveur injoignable.
 */
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

/** Intervalle de réinterrogation pendant le démarrage du serveur. */
export const SERVER_STARTUP_POLL_MS = 1000

/**
 * Attente maximale d'un serveur qui démarre (couvre une compilation par air), mesurée en
 * temps depuis le premier échec « en démarrage » de la série de tentatives.
 */
export const SERVER_STARTUP_MAX_WAIT_MS = 120_000

/** Code d'erreur de la réponse 503 d'un serveur en cours d'initialisation. */
export const SERVER_STARTING_CODE = 'server_starting'

/** Code d'erreur de la réponse 503 d'un démarrage qui a échoué (non réessayable). */
export const SERVER_START_FAILED_CODE = 'server_start_failed'

/**
 * Code d'erreur du 401 de l'application desktop : la requête ne porte pas le cookie posé par
 * l'entrée de la coquille (ADR 0045 D-3), par exemple après son expiration. Rejouer n'y change rien.
 */
export const DESKTOP_UNAUTHORIZED_CODE = 'desktop_unauthorized'

/** Statut du 401 de l'entrée de la coquille. */
const DESKTOP_UNAUTHORIZED_STATUS = 401

/** Étape annoncée pendant les migrations : la progression prolonge l'attente maximale. */
const MIGRATIONS_STEP = 'migrations'

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

/** Libellé de chaque état d'échec annoncé par le serveur (details.state). */
const FAILURE_STATE_KEYS: Readonly<Record<string, CommonManifestKey>> = {
  locked: 'common.root.server_failed.locked',
  migration_failed: 'common.root.server_failed.migration_failed',
  schema_newer: 'common.root.server_failed.schema_newer',
  port: 'common.root.server_failed.port',
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

/**
 * Vrai si l'erreur est le 401 `desktop_unauthorized` : l'application desktop n'a plus (ou n'a pas
 * encore) son cookie d'entrée. Seule la coquille sait le reposer, en rouvrant l'application.
 */
export function isDesktopEntryRequiredError(error: unknown): boolean {
  const e = asErrorShape(error)
  return e?.status === DESKTOP_UNAUTHORIZED_STATUS && e?.code === DESKTOP_UNAUTHORIZED_CODE
}

/** Lit un champ texte de `details` (étape, état) ; undefined si absent ou non textuel. */
function detailString(error: unknown, field: 'step' | 'state'): string | undefined {
  const details = asErrorShape(error)?.details
  if (details == null || typeof details !== 'object') return undefined
  const value = (details as Record<string, unknown>)[field]
  return typeof value === 'string' ? value : undefined
}

/** Clé du libellé de l'étape annoncée par le serveur ; undefined si absente ou inconnue. */
export function serverStartingStepKey(error: unknown): CommonManifestKey | undefined {
  const step = detailString(error, 'step')
  if (step === undefined || !Object.hasOwn(STEP_MESSAGE_KEYS, step)) return undefined
  return STEP_MESSAGE_KEYS[step]
}

/**
 * Vrai si l'erreur est la réponse 503 `server_start_failed` : le démarrage a échoué et le
 * serveur le dit (`retryable: false`). Aucun rejeu n'y changera rien.
 */
export function isServerStartFailedError(error: unknown): boolean {
  const e = asErrorShape(error)
  return (
    e?.status === SERVER_STARTING_STATUS &&
    e?.code === SERVER_START_FAILED_CODE &&
    (e as { retryable?: unknown }).retryable !== true
  )
}

/**
 * Écran d'un démarrage échoué : null si l'erreur n'en est pas un ; sinon `stateKey`, le
 * libellé de l'état annoncé (undefined si l'état est absent ou inconnu).
 */
export function serverStartFailure(error: unknown): { stateKey: CommonManifestKey | undefined } | null {
  if (!isServerStartFailedError(error)) return null
  const state = detailString(error, 'state')
  const stateKey =
    state !== undefined && Object.hasOwn(FAILURE_STATE_KEYS, state) ? FAILURE_STATE_KEYS[state] : undefined
  return { stateKey }
}

/** Vrai si le serveur répond « en démarrage » à l'étape des migrations (progression). */
function isMigrationProgress(error: unknown): boolean {
  return isServerStartingError(error) && detailString(error, 'step') === MIGRATIONS_STEP
}

/** Politique de rejeu du /bootstrap, à brancher sur `retry` et `retryDelay` de useQuery. */
export interface BootstrapRetryPolicy {
  retry: (failureCount: number, error: unknown) => boolean
  retryDelay: (failureCount: number, error: unknown) => number
}

/**
 * Crée la politique de rejeu d'UNE requête /bootstrap (état propre à l'instance).
 *
 * TanStack Query passe un compteur d'échecs CUMULÉ sur toute une série de tentatives, et
 * appelle retryDelay puis retry pour chaque échec. La politique en tire :
 *   - serveur qui démarre : réinterrogation chaque seconde tant que moins de
 *     SERVER_STARTUP_MAX_WAIT_MS se sont écoulées depuis le premier échec « en
 *     démarrage » de la série (plafond mesuré en temps) ;
 *   - démarrage échoué (503 server_start_failed, non réessayable) ou entrée de la coquille
 *     manquante (401 desktop_unauthorized) : aucun rejeu ;
 *   - autre erreur : BOOTSTRAP_OTHER_ERROR_RETRIES rejeux espacés de 0,5 s à 4 s, comptés
 *     à partir de la dernière erreur « en démarrage » (pas sur le compteur cumulé) : un
 *     serveur enfin prêt qui rend une erreur ordinaire obtient tous ses rejeux.
 * Un compteur qui repart en arrière (ou un même rang avec une autre erreur) ouvre une
 * nouvelle série.
 */
export function createBootstrapRetryPolicy(now: () => number = Date.now): BootstrapRetryPolicy {
  let last: { failureCount: number; error: unknown } | null = null
  let startupSince: number | null = null
  // Rang (compteur cumulé) de la première erreur ordinaire qui suit la dernière attente.
  let ordinaryBase = 0

  function observe(failureCount: number, error: unknown): void {
    if (last && last.failureCount === failureCount && last.error === error) return
    if (!last || failureCount <= last.failureCount) {
      startupSince = null
      ordinaryBase = 0
    }
    last = { failureCount, error }
    if (isServerStartingError(error)) {
      startupSince ??= now()
      // Migrations en cours : l'attente repart de cette réponse (progression, pas blocage).
      if (isMigrationProgress(error)) startupSince = now()
      ordinaryBase = failureCount + 1
    }
  }

  return {
    retry(failureCount, error) {
      observe(failureCount, error)
      // Démarrage échoué, ou entrée de la coquille manquante : un rejeu ne changera rien.
      if (isServerStartFailedError(error) || isDesktopEntryRequiredError(error)) return false
      if (isServerStartingError(error)) {
        return now() - (startupSince ?? now()) < SERVER_STARTUP_MAX_WAIT_MS
      }
      return failureCount - ordinaryBase < BOOTSTRAP_OTHER_ERROR_RETRIES
    },
    retryDelay(failureCount, error) {
      observe(failureCount, error)
      if (isServerStartingError(error)) return SERVER_STARTUP_POLL_MS
      return Math.min(
        BOOTSTRAP_OTHER_ERROR_BASE_DELAY_MS * 2 ** (failureCount - ordinaryBase),
        BOOTSTRAP_OTHER_ERROR_MAX_DELAY_MS,
      )
    },
  }
}
