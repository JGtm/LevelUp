// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createReliefComputer, defaultWorkerFactory, type ReliefWorkerLike } from './client'
import { ReliefEngine, type ReliefRenderRequest } from './engine'
import { ReliefError } from './errors'
import type { ReliefRequest, ReliefWorkerMessage } from './protocol'
import { fourBlocksScene } from './reliefScene.testutil'
import { handleReliefRequest } from './workerHandler'

const request = (over: Partial<ReliefRenderRequest> = {}): ReliefRenderRequest => ({
  turns: 0,
  cut: 2.2,
  base: '#3f8c4f',
  lines: true,
  masks: true,
  ...over,
})

/** Laisse passer les microtâches et un tour de boucle. */
const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

/**
 * Un faux worker qui se comporte comme le vrai : il copie les messages (`structuredClone`, avec
 * transfert), exécute le moteur et répond de façon asynchrone. `hold` retient les réponses.
 */
class LoopbackWorker implements ReliefWorkerLike {
  onmessage: ((event: MessageEvent<ReliefWorkerMessage>) => void) | null = null
  onerror: ((event: ErrorEvent) => void) | null = null
  onmessageerror: ((event: MessageEvent) => void) | null = null
  posted: Array<{ message: ReliefRequest; transfer: ArrayBuffer[] }> = []
  terminated = false
  hold = false
  private held: Array<() => void> = []
  private engine = new ReliefEngine()

  constructor(announce = true) {
    if (announce) queueMicrotask(() => this.emit({ op: 'ready' }))
  }

  postMessage(message: ReliefRequest, transfer: ArrayBuffer[]): void {
    this.posted.push({ message, transfer })
    const received = structuredClone(message, { transfer })
    const reply = (): void => {
      const { response, transfer: back } = handleReliefRequest(this.engine, received)
      this.emit(structuredClone(response, { transfer: back }))
    }
    if (this.hold) this.held.push(reply)
    else queueMicrotask(reply)
  }

  release(): void {
    const pending = this.held
    this.held = []
    for (const reply of pending) reply()
  }

  terminate(): void {
    this.terminated = true
  }

  emit(data: ReliefWorkerMessage): void {
    this.onmessage?.({ data } as MessageEvent<ReliefWorkerMessage>)
  }
}

describe('createReliefComputer : repli synchrone', () => {
  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
  })
  afterEach(() => vi.restoreAllMocks())

  it('sans Worker dans le navigateur (jsdom, vieux navigateur), calcule sur place', async () => {
    expect(typeof Worker).toBe('undefined')
    expect(() => defaultWorkerFactory()).toThrow('Worker indisponible')
    const computer = createReliefComputer()
    const info = await computer.load(fourBlocksScene())
    expect(console.warn).toHaveBeenCalledTimes(1)
    expect(info).toEqual({ width: 40, height: 10, hasMesh: true })
    const frame = await computer.render(request())
    expect(frame?.cut).toBe(2.25)
    computer.dispose()
  })

  it('une fabrique qui lève bascule sur le repli, en le journalisant', async () => {
    const computer = createReliefComputer({
      workerFactory: () => {
        throw new Error('CSP')
      },
    })
    await computer.load(fourBlocksScene())
    expect(vi.mocked(console.warn).mock.calls[0][0]).toContain('CSP')
    expect((await computer.render(request()))?.cut).toBe(2.25)
  })

  it('un script qui ne charge pas (erreur avant « ready ») bascule sur le repli et ferme le worker', async () => {
    const worker = new LoopbackWorker(false)
    queueMicrotask(() => worker.onerror?.(({}) as ErrorEvent))
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    expect(worker.terminated).toBe(true)
    expect((await computer.render(request()))?.cut).toBe(2.25)
  })

  it('un worker qui n’annonce pas son démarrage dans le délai bascule sur le repli', async () => {
    const worker = new LoopbackWorker(false)
    const computer = createReliefComputer({ workerFactory: () => worker, startTimeoutMs: 20 })
    await computer.load(fourBlocksScene())
    expect(worker.terminated).toBe(true)
    expect(worker.posted).toHaveLength(0)
    expect(console.warn).toHaveBeenCalledTimes(1)
  })
})

describe('createReliefComputer : worker', () => {
  afterEach(() => vi.restoreAllMocks())

  it('utilise le worker quand il annonce son démarrage ; le résultat est celui du repli', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    expect(worker.posted.some((p) => p.message.op === 'load')).toBe(true)
    const viaWorker = await computer.render(request())

    vi.spyOn(console, 'warn').mockImplementation(() => {})
    const inline = createReliefComputer({
      workerFactory: () => {
        throw new Error('pas de worker')
      },
    })
    await inline.load(fourBlocksScene())
    const viaInline = await inline.render(request())

    expect(viaWorker).not.toBeNull()
    expect(Array.from(viaWorker!.image!)).toEqual(Array.from(viaInline!.image!))
    expect(Array.from(viaWorker!.alpha!)).toEqual(Array.from(viaInline!.alpha!))
    expect(Array.from(viaWorker!.lines!.steps.points)).toEqual(Array.from(viaInline!.lines!.steps.points))
  })

  it('cède les hauteurs et les drapeaux de l’image au worker (transfert, pas copie)', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    const scene = fourBlocksScene()
    await computer.load(scene)
    const load = worker.posted.find((p) => p.message.op === 'load')!
    expect(load.transfer).toHaveLength(2)
    expect(scene.image.heights.byteLength).toBe(0)
    expect(scene.image.flags.byteLength).toBe(0)
  })

  it('une erreur du moteur rejette la demande avec son code', async () => {
    const computer = createReliefComputer({ workerFactory: () => new LoopbackWorker() })
    const err = await computer.render(request()).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ReliefError)
    expect((err as ReliefError).code).toBe('not-loaded')
  })

  it('une seule coupe en calcul : la dernière demande gagne, les intermédiaires rendent null', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    worker.hold = true
    const a = computer.render(request({ cut: 0 }))
    await flush()
    const b = computer.render(request({ cut: 1 }))
    const c = computer.render(request({ cut: 2.2 }))
    await flush()
    expect(worker.posted.filter((p) => p.message.op === 'render')).toHaveLength(1)
    worker.hold = false
    worker.release()
    expect((await a)?.cut).toBe(0)
    expect(await b).toBeNull()
    expect((await c)?.cut).toBe(2.25)
    expect(worker.posted.filter((p) => p.message.op === 'render')).toHaveLength(2)
  })

  it('charger une autre carte abandonne la demande en attente', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    worker.hold = true
    const a = computer.render(request({ cut: 0 }))
    await flush()
    const queued = computer.render(request({ cut: 2.2 }))
    worker.hold = false
    const reload = computer.load(fourBlocksScene())
    expect(await queued).toBeNull()
    worker.release()
    await a
    await reload
  })

  it('un worker qui tombe après son démarrage rejette ce qui attend, et les demandes suivantes', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    worker.hold = true
    const pending = computer.render(request())
    await flush()
    worker.onerror?.(({}) as ErrorEvent)
    await expect(pending).rejects.toMatchObject({ code: 'worker-failed' })
    await expect(computer.render(request())).rejects.toMatchObject({ code: 'worker-failed' })
    expect(worker.terminated).toBe(true)
  })

  it('dispose ferme le worker', async () => {
    const worker = new LoopbackWorker()
    const computer = createReliefComputer({ workerFactory: () => worker })
    await computer.load(fourBlocksScene())
    computer.dispose()
    await flush()
    expect(worker.terminated).toBe(true)
  })
})
