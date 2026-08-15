/**
 * replayAudio.ts — LA LECTURE Web Audio des sons du rejeu, et rien d'autre.
 *
 * LA COUPE SE FAIT À LA LECTURE, par une enveloppe de gain : tenue pleine, puis fondu de
 * sortie, arrêt à ~1 s. Un `stop()` sec au milieu d'une onde claquerait (discontinuité) —
 * le fondu est la coupe propre. Les fichiers livrés sont eux-mêmes tronqués à 1,2 s (item
 * 5.1 : 16,9 Mo de rafales et de queues de réverbération pour une seconde jouée), ce qui
 * laisse 0,2 s de marge au-delà du `stop` : la coupe reste celle de l'enveloppe, jamais
 * une fin de fichier.
 *
 * L'AudioContext ne naît QUE dans le geste utilisateur qui active le son (politique
 * d'autoplay des navigateurs : un contexte créé hors geste démarre suspendu). Le son du
 * rejeu étant COUPÉ PAR DÉFAUT, rien ne se charge ni ne sonne tant que l'utilisateur n'a
 * pas cliqué — c'est aussi ce qui respecte `prefers-reduced-motion` par construction :
 * aucune stimulation non demandée.
 *
 * Un fichier introuvable ou indécodable est mémorisé comme ABSENT (silence propre, pas de
 * re-tentative à chaque kill) — le manifeste (replaySound.ts) rend le cas exceptionnel,
 * le garde-rail d'assets le rend visible en CI.
 */

/** Durée maximale jouée d'un son, en secondes (coupe du lot 5 : « ~1 s »). */
export const SOUND_CUT_S = 1.0

/** Durée du fondu de sortie, en secondes (borné à la moitié du son pour les très courts). */
export const SOUND_FADE_S = 0.25

/**
 * Voix simultanées maximum : au-delà, les sons supplémentaires sont sautés. Sur un échange
 * nourri, empiler vingt sources ne raconte rien de plus qu'un mur de bruit.
 *
 * C'EST LE SEUL PLAFOND DE TOUTE LA CHAÎNE SONORE depuis que TOUS les tirs sonnent
 * (décision utilisateur du 2026-08-15 : aucun filtrage éditorial). Ce qu'il coûte, mesuré
 * le même jour par simulation à 1× (une voix tenue `SOUND_CUT_S`) : sur le film témoin
 * 000d5950, 46 sources refusées pour 483 tirs sonores ; sur les 23 artefacts locaux,
 * 4 897 refus pour 17 068 sources, soit 28,7 %. Le relever est un changement d'UN chiffre —
 * à faire si l'écoute le demande, pas avant : c'est une décision d'oreille, pas de code.
 */
export const SOUND_MAX_VOICES = 8

/**
 * Durée de la rampe de volume, en secondes. Poser `gain.value` d'un coup pendant qu'un son
 * joue fait un CLIC (discontinuité) — au curseur de volume, qui émet des dizaines de
 * valeurs par seconde, ce serait un crépitement. 20 ms suffisent et ne s'entendent pas.
 */
export const VOLUME_RAMP_S = 0.02

/**
 * soundEnvelope calcule l'enveloppe d'une source de `durationS` secondes : l'instant où
 * le fondu commence et celui où tout s'arrête. Pure, testée (replaySound.test.ts).
 */
export function soundEnvelope(durationS: number): { fadeStartS: number; stopS: number } {
  const stopS = Math.min(Math.max(durationS, 0), SOUND_CUT_S)
  const fade = Math.min(SOUND_FADE_S, stopS / 2)
  return { fadeStartS: stopS - fade, stopS }
}

/**
 * ReplayAudioPlayer — contexte, volume maître, cache de buffers, lecture enveloppée.
 *
 * Le constructeur DOIT être appelé dans un geste utilisateur (clic sur le bouton son).
 */
export class ReplayAudioPlayer {
  private ctx: AudioContext
  private master: GainNode
  /** URL -> buffer décodé, ou null = absent/indécodable (silence mémorisé). */
  private buffers = new Map<string, AudioBuffer | null>()
  private pending = new Set<string>()
  private voices = 0

  /** `volume` est posé À LA CONSTRUCTION : un gain par défaut à 1 ferait passer le premier
   *  son à plein régime avant que la préférence de l'utilisateur ne soit appliquée. */
  constructor(volume: number) {
    this.ctx = new AudioContext()
    this.master = this.ctx.createGain()
    this.master.gain.value = Math.min(Math.max(volume, 0), 1)
    this.master.connect(this.ctx.destination)
  }

  /**
   * setVolume règle le volume maître (borné 0..1) par une RAMPE COURTE : c'est ce qui
   * distingue un réglage d'un clic. Volume 0 = coupure — les sons déjà en vol s'éteignent
   * avec le maître, aucun n'est laissé en fond.
   */
  setVolume(v: number): void {
    const t0 = this.ctx.currentTime
    const g = this.master.gain
    g.cancelScheduledValues(t0)
    g.setValueAtTime(g.value, t0)
    g.linearRampToValueAtTime(Math.min(Math.max(v, 0), 1), t0 + VOLUME_RAMP_S)
  }

  /** resume relance le contexte (à appeler dans le geste d'activation). */
  resume(): void {
    void this.ctx.resume()
  }

  /** dispose ferme le contexte (démontage du composant). */
  dispose(): void {
    void this.ctx.close()
  }

  /**
   * preload charge et décode les URLs pas encore connues, en tâche de fond. Un échec
   * (404, décodage) est mémorisé comme absent : jamais de re-tentative par événement.
   */
  preload(urls: Iterable<string>): void {
    for (const url of urls) {
      if (this.buffers.has(url) || this.pending.has(url)) continue
      this.pending.add(url)
      void this.load(url)
    }
  }

  private async load(url: string): Promise<void> {
    try {
      const res = await fetch(url)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const raw = await res.arrayBuffer()
      this.buffers.set(url, await this.ctx.decodeAudioData(raw))
    } catch (err) {
      // Silence propre ET dit : un asset du manifeste qui manque au runtime est une
      // anomalie de déploiement, pas un cas nominal — une trace, pas une par kill.
      console.warn('[replay-audio] son indisponible, silence :', url, err)
      this.buffers.set(url, null)
    } finally {
      this.pending.delete(url)
    }
  }

  /**
   * play joue une URL déjà chargée : tenue pleine puis fondu (soundEnvelope). Pas encore
   * chargée (scrub avant la fin du preload) ou absente : silence, jamais d'attente — un
   * son en retard sur son image est pire qu'un son manqué.
   */
  play(url: string): void {
    const buf = this.buffers.get(url)
    if (!buf || this.voices >= SOUND_MAX_VOICES) {
      if (buf === undefined) this.preload([url])
      return
    }
    const t0 = this.ctx.currentTime
    const { fadeStartS, stopS } = soundEnvelope(buf.duration)
    const gain = this.ctx.createGain()
    gain.gain.setValueAtTime(1, t0)
    gain.gain.setValueAtTime(1, t0 + fadeStartS)
    gain.gain.linearRampToValueAtTime(0, t0 + stopS)
    const src = this.ctx.createBufferSource()
    src.buffer = buf
    src.connect(gain)
    gain.connect(this.master)
    this.voices++
    src.onended = () => {
      this.voices--
      src.disconnect()
      gain.disconnect()
    }
    src.start(t0)
    src.stop(t0 + stopS)
  }
}
