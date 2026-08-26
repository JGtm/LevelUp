/**
 * fakeAudio.ts — LE double Web Audio des tests du rejeu, partagé (patron testDoc.ts).
 *
 * jsdom n'implémente ni AudioContext ni le décodage : sans double, rien de la chaîne
 * sonore n'est testable. Celui-ci n'imite pas le son — il NOTE ce qu'on lui demande
 * (sources démarrées, enveloppes programmées, arrêts), ce qui est justement ce qui casse
 * en silence. Le rendu, lui, se juge à l'écoute, jamais ici.
 *
 * Deux tests s'en servent (moteur et câblage) : il vit donc à un seul endroit dès la
 * deuxième copie, avant de diverger.
 */
import { vi } from 'vitest'

/** Un paramètre de gain qui enregistre l'enveloppe qu'on lui programme. */
export class FakeParam {
  value = 1
  calls: Array<[string, number, number]> = []
  setValueAtTime(v: number, t: number) { this.calls.push(['set', v, t]); this.value = v }
  linearRampToValueAtTime(v: number, t: number) { this.calls.push(['ramp', v, t]) }
  cancelScheduledValues(t: number) { this.calls.push(['cancel', 0, t]) }
}

/**
 * FakeGain note VERS QUOI il est branché depuis le 2026-08-26 (robinet d'enregistrement) :
 * la question « le son du clip sort-il du même endroit que celui des haut-parleurs ? » ne se
 * répond pas autrement, et elle casserait en silence — un clip muet ne lève aucune erreur.
 */
export class FakeGain {
  gain = new FakeParam()
  connections: unknown[] = []
  connect(dest?: unknown) { this.connections.push(dest) }
  disconnect() { this.connections.length = 0 }
}

/** Le passe-bas de la chaîne de distance ; il note ses branchements comme le gain. */
export class FakeFilter {
  type = ''
  frequency = new FakeParam()
  connections: unknown[] = []
  connect(dest?: unknown) { this.connections.push(dest) }
  disconnect() { this.connections.length = 0 }
}

/** Le robinet d'enregistrement : un nœud de sortie qui expose une piste audio. */
export class FakeStreamDestination {
  track = { kind: 'audio' } as unknown as MediaStreamTrack
  stream = { getAudioTracks: () => [this.track] }
}

export class FakeSource {
  buffer: AudioBuffer | null = null
  onended: (() => void) | null = null
  started: number | null = null
  stopped: number | null = null
  connect() {}
  disconnect() {}
  start(t: number) { this.started = t }
  stop(t: number) { this.stopped = t }
  /** Fin naturelle de la source : c'est le navigateur qui l'appelle, ici le test. */
  end() { this.onended?.() }
}

export class FakeContext {
  currentTime = 10
  destination = {}
  gains: FakeGain[] = []
  sources: FakeSource[] = []
  filters: FakeFilter[] = []
  streamDests: FakeStreamDestination[] = []
  resumed = 0
  closed = 0
  createGain() { const g = new FakeGain(); this.gains.push(g); return g }
  createBiquadFilter() { const f = new FakeFilter(); this.filters.push(f); return f }
  createMediaStreamDestination() {
    const d = new FakeStreamDestination()
    this.streamDests.push(d)
    return d
  }
  createBufferSource() { const s = new FakeSource(); this.sources.push(s); return s }
  decodeAudioData(raw: ArrayBuffer) {
    // La « durée » du buffer est encodée dans la taille du tampon (1 octet = 0,1 s) : le
    // test choisit ainsi des sons longs ou courts sans embarquer de vrai WAV.
    return Promise.resolve({ duration: raw.byteLength / 10 } as AudioBuffer)
  }
  close() { this.closed++; return Promise.resolve() }
  resume() { this.resumed++; return Promise.resolve() }
}

/** Une réponse `fetch` réussie portant un tampon qui « dure » `seconds`. */
export function okAudioResponse(seconds: number) {
  return { ok: true, arrayBuffer: () => Promise.resolve(new ArrayBuffer(Math.round(seconds * 10))) }
}

/**
 * installFakeAudio remplace AudioContext et fetch le temps d'un test. À appeler dans un
 * `beforeEach`, à défaire par `vi.unstubAllGlobals()`.
 */
export function installFakeAudio(seconds = 3): {
  ctx: FakeContext
  fetchMock: ReturnType<typeof vi.fn>
} {
  const ctx = new FakeContext()
  // `new AudioContext()` doit rendre NOTRE double : une fonction constructeur qui retourne
  // un objet ignore l'instance neuve et sert celui-là (vi.fn() n'est pas constructible).
  vi.stubGlobal('AudioContext', function AudioContextDouble() { return ctx })
  const fetchMock = vi.fn(() => Promise.resolve(okAudioResponse(seconds)))
  vi.stubGlobal('fetch', fetchMock)
  return { ctx, fetchMock }
}

/** Laisse les promesses de chargement/décodage se résoudre. */
export const flushAudio = () => new Promise((r) => setTimeout(r, 0))
