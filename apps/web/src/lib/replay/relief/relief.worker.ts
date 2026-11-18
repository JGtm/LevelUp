/**
 * relief.worker — LE WORKER DU RELIEF : une enveloppe de messages autour du moteur.
 *
 * Premier worker de l'application. Chargé par `client.ts` :
 * `new Worker(new URL('./relief.worker.ts', import.meta.url), { type: 'module' })`. Il n'importe
 * ni le DOM ni React : `engine.ts` et ses dépendances seulement. Toute la logique est dans
 * `workerHandler.ts` (testée sans worker).
 */
import { ReliefEngine } from './engine'
import type { ReliefRequest, ReliefWorkerMessage } from './protocol'
import { handleReliefRequest } from './workerHandler'

/** Ce que ce fichier utilise de la portée d'un worker (le lib « DOM » du projet ne porte pas celle-ci). */
interface WorkerScope {
  onmessage: ((event: MessageEvent<ReliefRequest>) => void) | null
  postMessage(message: ReliefWorkerMessage, transfer: ArrayBuffer[]): void
}

const scope = self as unknown as WorkerScope
const engine = new ReliefEngine()

scope.onmessage = (event) => {
  const { response, transfer } = handleReliefRequest(engine, event.data)
  scope.postMessage(response, transfer)
}

scope.postMessage({ op: 'ready' }, [])
