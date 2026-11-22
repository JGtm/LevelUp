/**
 * relief/protocol — LES MESSAGES ENTRE LE FIL PRINCIPAL ET LE WORKER DU RELIEF.
 *
 * Le worker n'est qu'une ENVELOPPE de messages (conception §6.2) : tout ce qu'il calcule est une
 * fonction pure de cette bibliothèque (`engine.ts`), testée sans worker. Chaque demande porte un
 * `id` ; la réponse le reprend. Le worker annonce son démarrage par `{ op: 'ready' }`.
 *
 * TRANSFERT : les tableaux d'une `ReliefFrame` reviennent par transfert (`postMessage(msg,
 * [buffer])`), pas par copie. A l'aller, `load` CEDE les hauteurs et les drapeaux de l'image au
 * worker (transfert) : l'appelant ne les relit plus.
 */
import type { ReliefErrorCode } from './errors'
import type { ReliefFrame, ReliefInfo, ReliefLoadInput, ReliefRenderRequest } from './engine'

/** Une demande du fil principal au worker. */
export type ReliefRequest =
  | { id: number; op: 'load'; input: ReliefLoadInput }
  | { id: number; op: 'render'; request: ReliefRenderRequest }
  | { id: number; op: 'dispose' }

/** La réponse du worker à une demande. */
export type ReliefReply =
  | { id: number; ok: true; op: 'load'; info: ReliefInfo }
  | { id: number; ok: true; op: 'render'; frame: ReliefFrame }
  | { id: number; ok: true; op: 'dispose' }
  | { id: number; ok: false; error: { code: ReliefErrorCode; message: string } }

/** Le message de démarrage du worker, sans `id`. */
export interface ReliefReadyMessage {
  op: 'ready'
}

/** Tout ce que le worker envoie au fil principal. */
export type ReliefWorkerMessage = ReliefReply | ReliefReadyMessage

/** isReadyMessage reconnaît le message de démarrage. */
export function isReadyMessage(message: unknown): message is ReliefReadyMessage {
  return typeof message === 'object' && message !== null && (message as { op?: unknown }).op === 'ready'
}
