/**
 * relief/client — LE CALCULATEUR DU RELIEF côté fil principal : un worker, ou son repli synchrone.
 *
 * `createReliefComputer()` essaie d'abord le worker (premier worker de l'application) ; il ne le
 * garde que s'il a ANNONCE son démarrage dans le délai. Sans `Worker` (jsdom, vieux navigateur),
 * quand sa construction échoue, quand son script ne charge pas (erreur avant `ready`) ou quand il
 * ne répond pas à temps, le client se rabat sur le MEME moteur exécuté sur place (`ReliefEngine`) :
 * même code, même résultat, mais le fil principal le subit. Le repli est journalisé, jamais silencieux.
 *
 * UNE SEULE COUPE EN CALCUL, LA DERNIERE DEMANDE GAGNE (conception §6.2) : pendant qu'un rendu est
 * en cours, les demandes suivantes se remplacent dans UNE case d'attente ; une demande remplacée
 * rend `null` (rien à afficher, une plus récente arrive). Le curseur peut donc envoyer à chaque
 * pixel de glissement sans empiler de calculs.
 *
 * `load` cède les hauteurs et les drapeaux de l'image au worker (transfert) : ne plus les relire.
 */
import { ReliefEngine, type ReliefFrame, type ReliefInfo, type ReliefLoadInput, type ReliefRenderRequest } from './engine'
import { ReliefError } from './errors'
import { isReadyMessage, type ReliefRequest, type ReliefResponse, type ReliefWorkerMessage } from './protocol'
import { handleReliefRequest } from './workerHandler'

/** Où tourne le calcul. */
export type ReliefMode = 'worker' | 'inline'

/** Ce que le client utilise d'un `Worker` (le vrai, ou un faux dans les tests). */
export interface ReliefWorkerLike {
  postMessage(message: ReliefRequest, transfer: ArrayBuffer[]): void
  terminate(): void
  onmessage: ((event: MessageEvent<ReliefWorkerMessage>) => void) | null
  onerror: ((event: ErrorEvent) => void) | null
  onmessageerror: ((event: MessageEvent) => void) | null
}

/** Les options du calculateur. */
export interface ReliefComputerOptions {
  /** Fabrique du worker ; défaut : le worker du relief. Un échec bascule sur le repli. */
  workerFactory?: () => ReliefWorkerLike
  /** Ne pas essayer le worker. */
  forceInline?: boolean
  /** Délai de démarrage du worker avant repli (ms). */
  startTimeoutMs?: number
}

const DEFAULT_START_TIMEOUT_MS = 5000

/** Un moyen d'exécuter les demandes : le worker ou le moteur sur place. */
interface Transport {
  mode: ReliefMode
  call(build: (id: number) => ReliefRequest, transfer: ArrayBuffer[]): Promise<ReliefResponse>
  close(): void
}

/** defaultWorkerFactory crée le worker du relief ; lève quand le navigateur n'a pas de `Worker`. */
export function defaultWorkerFactory(): ReliefWorkerLike {
  if (typeof Worker === 'undefined') throw new Error('Worker indisponible')
  return new Worker(new URL('./relief.worker.ts', import.meta.url), { type: 'module' })
}

class InlineTransport implements Transport {
  readonly mode: ReliefMode = 'inline'
  private engine = new ReliefEngine()
  private nextId = 1

  call(build: (id: number) => ReliefRequest): Promise<ReliefResponse> {
    return Promise.resolve().then(() => handleReliefRequest(this.engine, build(this.nextId++)).response)
  }

  close(): void {
    this.engine.dispose()
  }
}

interface Waiter {
  resolve: (r: ReliefResponse) => void
  reject: (e: Error) => void
}

class WorkerTransport implements Transport {
  readonly mode: ReliefMode = 'worker'
  private worker: ReliefWorkerLike
  private nextId = 1
  private waiting = new Map<number, Waiter>()
  private dead: Error | null = null

  private constructor(worker: ReliefWorkerLike) {
    this.worker = worker
  }

  /** open attend l'annonce de démarrage du worker ; rejette sur erreur de chargement ou délai dépassé. */
  static open(worker: ReliefWorkerLike, timeoutMs: number): Promise<WorkerTransport> {
    return new Promise((resolve, reject) => {
      const transport = new WorkerTransport(worker)
      const fail = (reason: string): void => {
        clearTimeout(timer)
        worker.terminate()
        reject(new ReliefError('worker-failed', reason))
      }
      const timer = setTimeout(() => fail('worker : démarrage non annoncé dans le délai'), timeoutMs)
      worker.onerror = () => fail('worker : erreur au chargement du script')
      worker.onmessageerror = () => fail('worker : message invalide au démarrage')
      worker.onmessage = (event) => {
        if (!isReadyMessage(event.data)) return
        clearTimeout(timer)
        transport.arm()
        resolve(transport)
      }
    })
  }

  /** Après le démarrage : les réponses et les pannes passent par le transport. */
  private arm(): void {
    this.worker.onmessage = (event) => {
      const message = event.data
      if (isReadyMessage(message)) return
      const waiter = this.waiting.get(message.id)
      if (!waiter) return
      this.waiting.delete(message.id)
      waiter.resolve(message)
    }
    this.worker.onerror = () => this.die('worker : erreur à l\'exécution')
    this.worker.onmessageerror = () => this.die('worker : message invalide')
  }

  private die(reason: string): void {
    this.dead = new ReliefError('worker-failed', reason)
    for (const waiter of this.waiting.values()) waiter.reject(this.dead)
    this.waiting.clear()
    this.worker.terminate()
  }

  call(build: (id: number) => ReliefRequest, transfer: ArrayBuffer[]): Promise<ReliefResponse> {
    if (this.dead) return Promise.reject(this.dead)
    const id = this.nextId++
    return new Promise((resolve, reject) => {
      this.waiting.set(id, { resolve, reject })
      try {
        this.worker.postMessage(build(id), transfer)
      } catch (err) {
        this.waiting.delete(id)
        reject(new ReliefError('worker-failed', `postMessage refusé : ${err instanceof Error ? err.message : String(err)}`))
      }
    })
  }

  close(): void {
    this.dead = new ReliefError('worker-failed', 'calculateur fermé')
    for (const waiter of this.waiting.values()) waiter.reject(this.dead)
    this.waiting.clear()
    this.worker.terminate()
  }
}

async function openTransport(options: ReliefComputerOptions): Promise<Transport> {
  if (options.forceInline) return new InlineTransport()
  try {
    const worker = (options.workerFactory ?? defaultWorkerFactory)()
    return await WorkerTransport.open(worker, options.startTimeoutMs ?? DEFAULT_START_TIMEOUT_MS)
  } catch (err) {
    console.warn(`[replay-relief] worker indisponible, calcul sur le fil principal : ${err instanceof Error ? err.message : String(err)}`)
    return new InlineTransport()
  }
}

/** Une demande de rendu en attente derrière celle qui calcule. */
interface QueuedRender {
  request: ReliefRenderRequest
  resolve: (frame: ReliefFrame | null) => void
  reject: (err: Error) => void
}

function toError(response: ReliefResponse): Error {
  if (response.ok) return new ReliefError('worker-failed', 'réponse inattendue')
  return new ReliefError(response.error.code, response.error.message)
}

/** Le calculateur d'une carte à la fois. */
export class ReliefComputer {
  private transport: Promise<Transport>
  private computing = false
  private queued: QueuedRender | null = null

  constructor(options: ReliefComputerOptions = {}) {
    this.transport = openTransport(options)
  }

  /** Où le calcul tourne : `'worker'`, ou `'inline'` quand le repli est en service. */
  mode(): Promise<ReliefMode> {
    return this.transport.then((t) => t.mode)
  }

  /**
   * load pose une carte (et remplace la précédente). Cède `image.heights` et `image.flags` au worker.
   * Une demande de rendu en attente est abandonnée (elle rend `null`).
   */
  async load(input: ReliefLoadInput): Promise<ReliefInfo> {
    this.supersedeQueued()
    const transport = await this.transport
    const transfer = transport.mode === 'worker' ? [input.image.heights.buffer as ArrayBuffer, input.image.flags.buffer as ArrayBuffer] : []
    const response = await transport.call((id) => ({ id, op: 'load', input }), transfer)
    if (!response.ok || response.op !== 'load') throw toError(response)
    return response.info
  }

  /**
   * render demande un rendu. Une seule demande calcule à la fois, la dernière gagne : une demande
   * remplacée avant son départ rend `null`.
   */
  render(request: ReliefRenderRequest): Promise<ReliefFrame | null> {
    return new Promise((resolve, reject) => {
      this.supersedeQueued()
      this.queued = { request, resolve, reject }
      void this.pump()
    })
  }

  /** Libère le worker et l'état du moteur. */
  dispose(): void {
    this.supersedeQueued()
    void this.transport.then((t) => t.close())
  }

  private supersedeQueued(): void {
    this.queued?.resolve(null)
    this.queued = null
  }

  private async pump(): Promise<void> {
    if (this.computing || this.queued === null) return
    const job = this.queued
    this.queued = null
    this.computing = true
    try {
      const transport = await this.transport
      const response = await transport.call((id) => ({ id, op: 'render', request: job.request }), [])
      if (!response.ok || response.op !== 'render') job.reject(toError(response))
      else job.resolve(response.frame)
    } catch (err) {
      job.reject(err instanceof Error ? err : new ReliefError('worker-failed', String(err)))
    } finally {
      this.computing = false
      void this.pump()
    }
  }
}

/** createReliefComputer ouvre un calculateur : le worker du relief, ou son repli synchrone. */
export function createReliefComputer(options: ReliefComputerOptions = {}): ReliefComputer {
  return new ReliefComputer(options)
}
