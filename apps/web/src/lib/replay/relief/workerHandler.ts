/**
 * relief/workerHandler — LE TRAITEMENT D'UNE DEMANDE, indépendant du worker.
 *
 * `relief.worker.ts` n'est que `onmessage` autour de cette fonction ; le repli synchrone du client
 * l'appelle directement. Une erreur est RENDUE dans la réponse (`ok: false`), jamais avalée : le
 * client la reconstruit en `ReliefError` et rejette la demande.
 */
import type { ReliefEngine, ReliefFrame } from './engine'
import { isReliefError } from './errors'
import { lineSetBuffers } from './lineSet'
import type { ReliefRequest, ReliefResponse } from './protocol'

/** Une réponse et les buffers à transférer avec elle. */
export interface HandledRequest {
  response: ReliefResponse
  transfer: ArrayBuffer[]
}

/** Les buffers d'une frame : à transférer, pas à copier. */
export function frameBuffers(frame: ReliefFrame): ArrayBuffer[] {
  const out: ArrayBuffer[] = []
  if (frame.image) out.push(frame.image.buffer as ArrayBuffer)
  if (frame.alpha) out.push(frame.alpha.buffer as ArrayBuffer)
  if (frame.planAlpha) out.push(frame.planAlpha.buffer as ArrayBuffer)
  if (frame.lines) {
    out.push(...lineSetBuffers(frame.lines.steps), ...lineSetBuffers(frame.lines.contours), ...lineSetBuffers(frame.lines.walls))
  }
  return out
}

/** handleReliefRequest exécute une demande sur le moteur et rend la réponse à renvoyer. */
export function handleReliefRequest(engine: ReliefEngine, req: ReliefRequest): HandledRequest {
  try {
    switch (req.op) {
      case 'load':
        return { response: { id: req.id, ok: true, op: 'load', info: engine.load(req.input) }, transfer: [] }
      case 'render': {
        const frame = engine.render(req.request)
        return { response: { id: req.id, ok: true, op: 'render', frame }, transfer: frameBuffers(frame) }
      }
      case 'dispose':
        engine.dispose()
        return { response: { id: req.id, ok: true, op: 'dispose' }, transfer: [] }
    }
  } catch (err) {
    const code = isReliefError(err) ? err.code : 'worker-failed'
    const message = err instanceof Error ? err.message : String(err)
    return { response: { id: req.id, ok: false, error: { code, message } }, transfer: [] }
  }
}
