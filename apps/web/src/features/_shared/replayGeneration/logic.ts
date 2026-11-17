/**
 * Logique pure de la génération à la demande d'un rejeu : ce que l'écran offre (action, attente, phrase) pour un
 * état servi par le serveur, et quand relire cet état. Aucun accès réseau ni React.
 *
 * Règle : une action n'existe que si elle est possible. Générer (`generatable`), réessayer (`failed` réessayable)
 * et ouvrir (`ready`) sont les trois seules ; tous les autres états n'offrent aucun bouton.
 */
import type { ReplayGenerationStatus } from './queries'

export type ReplayGenerationAction = 'generate' | 'retry' | 'open'

/** Attente affichée : en file, film en cours de téléchargement, rejeu en cours de construction. */
export type ReplayGenerationPending = 'queued' | 'downloading' | 'building'

export type ReplayGenerationMessageKey =
  | 'generatable'
  | 'queued'
  | 'queuedNoPosition'
  | 'waitingForGame'
  | 'paused'
  | 'downloading'
  | 'building'
  | 'failed'
  | 'failedFinal'
  | 'filmUnavailable'
  | 'modeUnmanaged'
  | 'matchUnknown'
  | 'noToken'
  | 'ready'

export interface ReplayGenerationView {
  action: ReplayGenerationAction | null
  pending: ReplayGenerationPending | null
  /** La phrase de l'état ; `null` pour un état ou une raison que ce client ne connaît pas. */
  messageKey: ReplayGenerationMessageKey | null
}

const NO_VIEW: ReplayGenerationView = { action: null, pending: null, messageKey: null }

function queuedView(status: ReplayGenerationStatus): ReplayGenerationView {
  let messageKey: ReplayGenerationMessageKey = 'queuedNoPosition'
  if (status.waiting_for_game) messageKey = 'waitingForGame'
  else if (status.paused) messageKey = 'paused'
  else if ((status.position ?? 0) >= 1) messageKey = 'queued'
  return { action: null, pending: 'queued', messageKey }
}

function notSupportedView(status: ReplayGenerationStatus): ReplayGenerationView {
  switch (status.reason) {
    case 'mode_unmanaged':
      return { ...NO_VIEW, messageKey: 'modeUnmanaged' }
    case 'match_unknown':
      return { ...NO_VIEW, messageKey: 'matchUnknown' }
    case 'no_token':
      return { ...NO_VIEW, messageKey: 'noToken' }
    default:
      return NO_VIEW
  }
}

/** Ce que l'écran montre pour cet état. `undefined` (état pas encore lu) : rien. */
export function generationView(status: ReplayGenerationStatus | undefined): ReplayGenerationView {
  if (!status) return NO_VIEW
  switch (status.state) {
    case 'generatable':
      return { action: 'generate', pending: null, messageKey: 'generatable' }
    case 'queued':
      return queuedView(status)
    case 'running': {
      const phase = status.phase === 'downloading' ? 'downloading' : 'building'
      return { action: null, pending: phase, messageKey: phase }
    }
    case 'failed':
      return status.retryable
        ? { action: 'retry', pending: null, messageKey: 'failed' }
        : { action: null, pending: null, messageKey: 'failedFinal' }
    case 'film_unavailable':
      return { ...NO_VIEW, messageKey: 'filmUnavailable' }
    case 'not_supported':
      return notSupportedView(status)
    case 'ready':
      return { action: 'open', pending: null, messageKey: 'ready' }
    default:
      return NO_VIEW
  }
}

/** Une demande est en file ou en cours : l'état change sans action de l'utilisateur, il se relit. */
export function generationInFlight(status: ReplayGenerationStatus | undefined): boolean {
  return status?.state === 'queued' || status?.state === 'running'
}

/** Ce que la demande elle-même (le POST) a de plus à dire que l'état relu : file pleine, ou échec sans suite. */
export type ReplayGenerationRequestError = 'queueFull' | 'requestFailed'

/**
 * Un 409 (le match ne se génère plus) et un 429 relisent l'état, dont la phrase dit déjà tout : seule la file pleine
 * garde un message propre, car l'état relu redevient « générable ». `null` : rien de plus à dire.
 */
export function requestErrorKey(err: unknown): ReplayGenerationRequestError | null {
  if (!err) return null
  const { status, code } = err as { status?: unknown; code?: unknown }
  if (code === 'replay_generation_queue_full' || status === 429) return 'queueFull'
  if (status === 409) return null
  return 'requestFailed'
}
