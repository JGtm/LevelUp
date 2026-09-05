/**
 * vehicleEnginePlayer.ts — LA LECTURE TEMPS RÉEL des moteurs de véhicules, et rien d'autre.
 *
 * Le lecteur d'événements (`replayAudio.ts`) joue des one-shots enveloppés ; un moteur est un
 * ÉTAT CONTINU (cf. l'en-tête de `vehicleEngineSound.ts`, qui porte le plan et le contrat de
 * la banque). Ce module tient les sources Web Audio de cet état :
 *
 *  - ENTRÉE CONTINUE (la lecture franchit T0) : `enter` part, et son fondu croisé vers `loop`
 *    est PRÉ-PROGRAMMÉ à la construction (automation Web Audio) — aucun battement à surveiller,
 *    la jonction tombe à l'échantillon près quelle que soit la charge de la page.
 *  - SEEK AU MILIEU D'UN ÉPISODE : reprise DIRECTEMENT sur `loop` (décision de cadrage n° 6,
 *    jamais de re-enter — un démarrage de moteur qui n'a pas eu lieu serait un mensonge).
 *  - SORTIE (la lecture franchit T1) : fondu croisé vers `exit`, qui se joue ENTIER et finit à
 *    zéro tout seul (fins gravées — AUCUNE enveloppe ajoutée, contrat de la banque).
 *  - PAUSE / COUPURE / DÉMONTAGE : rampe courte (`ENGINE_STOP_RAMP_S`, 20 ms) — l'anti-clic
 *    d'un arrêt que la timeline ne raconte pas, pas une enveloppe musicale.
 *  - VITESSE DE LECTURE ≠ 1 : les BORNES suivent la timeline (T0/T1 sont détectés sur les ms
 *    du rejeu), mais les fichiers se jouent à VITESSE NATURELLE — jamais de `playbackRate`,
 *    donc jamais de moteur « chipmunk ». Au-delà de `SOUND_MAX_SPEED`, le câblage
 *    (`useReplaySound`) cesse d'appeler `sync` et appelle `stopAll` : même silence que le
 *    reste de la piste, même message « son coupé par la vitesse ».
 *
 * LES MOTEURS NE COMPTENT PAS DANS `SOUND_MAX_VOICES` : ce plafond arbitre des one-shots de
 * mêlée équivalents entre eux ; couper un moteur au profit d'un tir raconterait qu'un véhicule
 * s'est tu sans raison. Deux ou trois moteurs simultanés sont le maximum réaliste d'un match,
 * et le bus à 0,85 les tient en retrait.
 */
import {
  ENGINE_CROSSFADE_S,
  ENGINE_STOP_RAMP_S,
  enginePhaseAt,
  equalPowerCurves,
  VEHICLE_ENGINE_STEMS,
  type EngineEpisode,
  type EnginePhase,
  type EnginePlan,
} from './vehicleEngineSound'

/**
 * Saut de timeline au-delà duquel une avance n'est plus une lecture continue : même valeur et
 * même raison que `SOUND_RESYNC_JUMP_MS` (replaySoundCursor.ts) — mais ici le recalage ne
 * SAUTE pas des sons, il réconcilie un état (couper ce qui ne doit plus sonner, reprendre la
 * boucle de ce qui doit).
 */
export const ENGINE_RESYNC_JUMP_MS = 1_000

/** Ce que le lecteur moteur demande à son hôte : le contexte, le bus moteur, les tampons. */
export interface EngineAudioPort {
  ctx: BaseAudioContext
  /** Le bus moteur (gain 0,85 posé par l'hôte), branché sur le maître du lecteur. */
  out: AudioNode
  /** Tampon décodé d'un stem, `null`/`undefined` = absent ou pas encore chargé : silence. */
  bufferOf: (stem: string) => AudioBuffer | null | undefined
}

/** Une source en vol : le nœud, son gain, et de quoi l'éteindre proprement. */
interface EngineVoice {
  src: AudioBufferSourceNode
  gain: GainNode
}

/** L'état d'un épisode en train de sonner. `holdUntil` protège le fondu enter->loop programmé. */
interface EngineChain {
  phase: EnginePhase
  /** La voix TENUE (loop ou idle). L'enter et l'exit sont des one-shots qui s'éteignent seuls. */
  held: EngineVoice | null
  /** Temps de contexte avant lequel aucun swap idle/loop n'est permis (fondu d'enter en cours). */
  holdUntil: number
}

const CURVES = equalPowerCurves()

export class VehicleEnginePlayer {
  private port: EngineAudioPort
  private plans: EnginePlan[] = []
  /** clé `plan:épisode` -> chaîne en vol. */
  private chains = new Map<string, EngineChain>()
  /**
   * Les ONE-SHOTS en vol (`enter` pendant son fondu, `exit` jusqu'à sa fin gravée) : ils ne
   * sont la voix tenue de personne, mais une PAUSE doit les éteindre aussi — un moteur qui
   * finit son démarrage pendant que la lecture est arrêtée serait un son sans image.
   */
  private tails = new Set<EngineVoice>()
  /** Dernier instant servi, ou `null` = le prochain `sync` est un recalage silencieux. */
  private lastMs: number | null = null

  constructor(port: EngineAudioPort) {
    this.port = port
  }

  /** setPlans remplace le plan (changement de document). Tout ce qui sonne s'arrête. */
  setPlans(plans: EnginePlan[]): void {
    this.stopAll()
    this.plans = plans
  }

  /**
   * sync fait suivre l'état des moteurs à l'instant `ms` du rejeu. À appeler à chaque
   * battement audible ; un saut (scrub, reprise) est détecté ici et réconcilié en silence.
   */
  sync(ms: number): void {
    const continuous =
      this.lastMs !== null && ms >= this.lastMs && ms - this.lastMs <= ENGINE_RESYNC_JUMP_MS
    const last = this.lastMs
    this.lastMs = ms
    this.forEachEpisode((key, plan, ep) => {
      const chain = this.chains.get(key)
      const phase = enginePhaseAt(ep, ms)
      if (!continuous || last === null) {
        this.reconcile(key, plan, chain, phase)
        return
      }
      // ENTRÉE : la lecture FRANCHIT T0 — le seul chemin qui joue `enter`.
      if (!chain && last < ep.t0Ms && ms >= ep.t0Ms && phase !== null) {
        this.startEnter(key, plan, phase)
        return
      }
      // SORTIE : la lecture FRANCHIT T1 — fondu vers `exit`, qui se joue entier.
      if (chain && last < ep.t1Ms && ms >= ep.t1Ms) {
        this.startExit(key, plan, chain)
        return
      }
      // COURSE <-> RALENTI en cours d'épisode, une fois le fondu d'enter retombé.
      if (chain && phase !== null && chain.phase !== phase && this.port.ctx.currentTime >= chain.holdUntil) {
        this.swapHeld(plan, chain, phase)
        return
      }
      // Épisode entré par un chemin non continu couvert plus haut ? Réconciliation par défaut.
      if (!chain && phase !== null) this.startHeld(key, plan, phase)
    })
  }

  /** stopAll éteint tout par la rampe courte. Le prochain `sync` sera un recalage. */
  stopAll(): void {
    for (const chain of this.chains.values()) this.release(chain)
    this.chains.clear()
    const t0 = this.port.ctx.currentTime
    for (const tail of this.tails) this.fadeOutVoice(tail, t0, ENGINE_STOP_RAMP_S)
    this.tails.clear()
    this.lastMs = null
  }

  dispose(): void {
    this.stopAll()
  }

  private forEachEpisode(fn: (key: string, plan: EnginePlan, ep: EngineEpisode) => void): void {
    for (let p = 0; p < this.plans.length; p++) {
      const plan = this.plans[p]
      for (let e = 0; e < plan.episodes.length; e++) fn(`${p}:${e}`, plan, plan.episodes[e])
    }
  }

  /** reconcile — l'état voulu SANS histoire : boucle directe dedans, silence dehors. */
  private reconcile(
    key: string,
    plan: EnginePlan,
    chain: EngineChain | undefined,
    phase: EnginePhase,
  ): void {
    if (chain && (phase === null || chain.phase !== phase)) {
      this.release(chain)
      this.chains.delete(key)
    }
    if (phase !== null && (!chain || chain.phase !== phase)) this.startHeld(key, plan, phase)
  }

  /** startHeld démarre la voix tenue (loop/idle) SANS enter : le chemin du seek. */
  private startHeld(key: string, plan: EnginePlan, phase: Exclude<EnginePhase, null>): void {
    const held = this.spawnLoop(plan, phase, this.port.ctx.currentTime, 0)
    if (!held) return
    this.chains.set(key, { phase, held, holdUntil: 0 })
  }

  /**
   * startEnter joue `enter` et PRÉ-PROGRAMME son fondu croisé vers la voix tenue : les deux
   * automations sont posées maintenant, la jonction n'a besoin d'aucun battement.
   */
  private startEnter(key: string, plan: EnginePlan, phase: Exclude<EnginePhase, null>): void {
    const stems = VEHICLE_ENGINE_STEMS[plan.family]
    const buf = stems ? this.port.bufferOf(stems.enter) : null
    const t0 = this.port.ctx.currentTime
    if (!buf) {
      // Enter absent (chargement en retard) : la boucle directe vaut mieux qu'un silence.
      this.startHeld(key, plan, phase)
      return
    }
    const fadeAt = t0 + Math.max(buf.duration - ENGINE_CROSSFADE_S, 0)
    const voice = this.spawnVoice(buf, false, t0, 1)
    this.trackTail(voice)
    voice.gain.gain.setValueCurveAtTime(CURVES.fadeOut, fadeAt, ENGINE_CROSSFADE_S)
    voice.src.stop(fadeAt + ENGINE_CROSSFADE_S)
    const held = this.spawnLoop(plan, phase, fadeAt, ENGINE_CROSSFADE_S)
    if (!held) return
    this.chains.set(key, { phase, held, holdUntil: fadeAt + ENGINE_CROSSFADE_S })
  }

  /** startExit fond la voix tenue vers `exit`, qui se joue ENTIER (fins gravées). */
  private startExit(key: string, plan: EnginePlan, chain: EngineChain): void {
    const stems = VEHICLE_ENGINE_STEMS[plan.family]
    const buf = stems ? this.port.bufferOf(stems.exit) : null
    const t0 = this.port.ctx.currentTime
    this.fadeOutHeld(chain, t0, ENGINE_CROSSFADE_S)
    this.chains.delete(key)
    if (!buf) return
    const voice = this.spawnVoice(buf, false, t0, 0)
    this.trackTail(voice)
    voice.gain.gain.setValueCurveAtTime(CURVES.fadeIn, t0, ENGINE_CROSSFADE_S)
    // Pas de `stop` posé : la source s'arrête à la fin de son fichier, et `onended` nettoie.
  }

  /** trackTail suit un one-shot en vol pour que `stopAll` puisse l'éteindre lui aussi. */
  private trackTail(voice: EngineVoice): void {
    this.tails.add(voice)
    const inner = voice.src.onended
    voice.src.onended = (ev) => {
      this.tails.delete(voice)
      if (inner) (inner as (this: AudioScheduledSourceNode, ev: Event) => void).call(voice.src, ev)
    }
  }

  /** fadeOutVoice — rampe courte puis arrêt d'une voix quelconque. */
  private fadeOutVoice(voice: EngineVoice, at: number, fadeS: number): void {
    const g = voice.gain.gain
    g.cancelScheduledValues(at)
    g.setValueCurveAtTime(scaledFadeOut(g.value), at, fadeS)
    voice.src.stop(at + fadeS)
  }

  /** swapHeld — fondu croisé loop <-> idle, 150 ms à puissance constante. */
  private swapHeld(plan: EnginePlan, chain: EngineChain, phase: Exclude<EnginePhase, null>): void {
    const t0 = this.port.ctx.currentTime
    const next = this.spawnLoop(plan, phase, t0, ENGINE_CROSSFADE_S)
    if (!next) return
    this.fadeOutHeld(chain, t0, ENGINE_CROSSFADE_S)
    chain.held = next
    chain.phase = phase
  }

  /** spawnLoop crée la voix TENUE d'une phase : source bouclée, fondu d'entrée optionnel. */
  private spawnLoop(
    plan: EnginePlan,
    phase: Exclude<EnginePhase, null>,
    at: number,
    fadeS: number,
  ): EngineVoice | null {
    const stems = VEHICLE_ENGINE_STEMS[plan.family]
    const stem = phase === 'idle' ? stems?.idle : stems?.loop
    const buf = stem ? this.port.bufferOf(stem) : null
    if (!buf) return null
    const voice = this.spawnVoice(buf, true, at, fadeS > 0 ? 0 : 1)
    if (fadeS > 0) voice.gain.gain.setValueCurveAtTime(CURVES.fadeIn, at, fadeS)
    return voice
  }

  /** spawnVoice — source + gain branchés sur le bus, nettoyés à la fin de la source. */
  private spawnVoice(buf: AudioBuffer, loop: boolean, at: number, gain0: number): EngineVoice {
    const ctx = this.port.ctx
    const gain = ctx.createGain()
    gain.gain.setValueAtTime(gain0, at)
    const src = ctx.createBufferSource()
    src.buffer = buf
    // LE FONDU DE REBOUCLAGE EST DANS LE FICHIER (contrat de la banque) : `loop = true` suffit.
    src.loop = loop
    src.connect(gain)
    gain.connect(this.port.out)
    src.onended = () => {
      src.disconnect()
      gain.disconnect()
    }
    src.start(Math.max(at, ctx.currentTime))
    return { src, gain }
  }

  /** fadeOutHeld éteint la voix tenue par un fondu, puis l'arrête. */
  private fadeOutHeld(chain: EngineChain, at: number, fadeS: number): void {
    if (!chain.held) return
    this.fadeOutVoice(chain.held, at, fadeS)
    chain.held = null
  }

  /** release — l'arrêt d'urgence (pause, coupure) : rampe de 20 ms, jamais un clic. */
  private release(chain: EngineChain): void {
    const t0 = this.port.ctx.currentTime
    this.fadeOutHeld(chain, t0, ENGINE_STOP_RAMP_S)
  }
}

/** La courbe de sortie, remise à l'échelle du gain courant (un fondu qui part d'où on est). */
function scaledFadeOut(from: number): Float32Array {
  const base = CURVES.fadeOut
  const out = new Float32Array(base.length)
  for (let i = 0; i < base.length; i++) out[i] = base[i] * from
  return out
}
