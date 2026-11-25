/**
 * chatterVoiceCache.ts — LES VOIX DES RÉPLIQUES, CHARGÉES À LA DEMANDE et gardées en mémoire à taille fixe.
 *
 * POURQUOI PAS UN PRÉCHARGEMENT EN BLOC. Un match porte jusqu'à environ 150 voix différentes ; décodées, elles pèsent
 * 0,5 Mo chacune en moyenne (mesure sur les 519 voix versées : durée moyenne 1,32 s × 48 kHz × 2 canaux × 4 octets =
 * 505 Ko, la plus longue 2,75 s = 1,05 Mo). Tout charger ferait payer 75 Mo à qui n'écoute rien : le son du rejeu est
 * coupé par défaut. La piste ne demande donc que les voix des quelques secondes à venir (`voicesInWindow`), au fil de la
 * lecture, et rien du tout tant que le son est coupé, que la voix est éteinte ou que la langue n'a pas de voix.
 *
 * LA BORNE : 16 voix décodées, soit 8 Mo en moyenne et 17 Mo au pire (seize voix de 2,75 s). La fenêtre d'avance de
 * quatre secondes porte rarement plus de huit répliques (huit joueurs, une réplique chacun) : seize laisse le double,
 * plus de quoi garder les dernières jouées si une phrase se répète. Au-delà, la voix la moins récemment demandée ou
 * jouée est libérée.
 *
 * UNE VOIX PAS ENCORE PRÊTE À L'INSTANT DE SA RÉPLIQUE EST SAUTÉE (`take` rend `undefined`) : un son décalé par rapport
 * au texte serait pire qu'un son manquant. Un échec (404, décodage) est mémorisé : pas de nouvelle tentative.
 *
 * TOUT SE COUPE AU DÉMONTAGE : `clear` annule les requêtes en cours (signal d'annulation) ; une réponse qui arrive après
 * ne se range pas, et une annulation n'est ni un échec ni un journal.
 *
 * Aucun réseau ici : le chargeur est injecté (`ReplayAudioPlayer.fetchDecoded`), ce qui le rend testable.
 */
import type { ReplaySoundEvent } from './replaySoundVariants'

/** Combien de voix décodées au plus restent en mémoire (cf. l'en-tête pour la mesure). */
export const CHATTER_VOICE_CACHE_MAX = 16

/** Combien de temps de film à l'avance on demande les voix : quatre secondes, soit deux secondes réelles à 2×. */
export const CHATTER_VOICE_AHEAD_MS = 4000

/** Charge et décode une voix ; rejette à l'annulation. */
export type VoiceLoader = (url: string, signal: AbortSignal) => Promise<AudioBuffer>

export class ChatterVoiceCache {
  /** Voix prêtes ; l'ordre d'insertion est l'ordre de récence (la première est la plus ancienne). */
  private ready = new Map<string, AudioBuffer>()
  private pending = new Map<string, AbortController>()
  private failed = new Set<string>()

  private readonly loader: VoiceLoader
  readonly max: number

  constructor(loader: VoiceLoader, max: number = CHATTER_VOICE_CACHE_MAX) {
    this.loader = loader
    this.max = max
  }

  /** Le nombre de voix décodées en mémoire. */
  get size(): number {
    return this.ready.size
  }

  /** Demande une voix, une seule fois : prête, en cours ou en échec, rien ne repart. */
  request(url: string): void {
    if (this.ready.has(url) || this.pending.has(url) || this.failed.has(url)) return
    const controller = new AbortController()
    this.pending.set(url, controller)
    this.loader(url, controller.signal).then(
      (buffer) => {
        if (controller.signal.aborted) return
        this.pending.delete(url)
        this.ready.set(url, buffer)
        while (this.ready.size > this.max) {
          const oldest = this.ready.keys().next().value
          if (oldest === undefined) break
          this.ready.delete(oldest)
        }
      },
      (err: unknown) => {
        if (controller.signal.aborted) return
        this.pending.delete(url)
        this.failed.add(url)
        console.warn('[replay-audio] voix indisponible, silence :', url, err)
      },
    )
  }

  /** La voix prête, ou `undefined` (jamais demandée, pas encore arrivée, ou en échec) : ne lance aucune requête. */
  take(url: string): AudioBuffer | undefined {
    const buffer = this.ready.get(url)
    if (buffer) {
      this.ready.delete(url)
      this.ready.set(url, buffer)
    }
    return buffer
  }

  /** Annule les requêtes en cours et libère tout. */
  clear(): void {
    for (const controller of this.pending.values()) controller.abort()
    this.pending.clear()
    this.ready.clear()
    this.failed.clear()
  }
}

/**
 * voicesInWindow — les stems de voix de réplique dont l'instant tombe dans ]from, from + ahead], sans doublon, dans
 * l'ordre du temps. `events` est la piste TRIÉE par instant : la recherche de départ est dichotomique.
 */
export function voicesInWindow(events: readonly ReplaySoundEvent[], from: number, ahead: number): string[] {
  let lo = 0
  let hi = events.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (events[mid].ms <= from) lo = mid + 1
    else hi = mid
  }
  const out: string[] = []
  for (let i = lo; i < events.length && events[i].ms <= from + ahead; i++) {
    const e = events[i]
    if (e.chatter && !out.includes(e.stem)) out.push(e.stem)
  }
  return out
}
