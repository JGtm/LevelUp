/**
 * replayAudio.ts — LA LECTURE Web Audio des sons du rejeu, et rien d'autre.
 *
 * LA DURÉE JOUÉE EST CELLE DU FICHIER, et c'est le fichier qui porte la règle de durée
 * par catégorie (décision utilisateur du 2026-08-16, lot R2.1) : une arme, un lancer et
 * la mêlée sont livrés à 1,2 s ; une explosion de grenade et un équipement vont jusqu'à
 * 4 s, parce que leur source dure 1,8 à 4,8 s et que les couper à la seconde les rendait
 * « écourtés » à l'oreille ; les répliques et les fanfares de FIN DE PARTIE (lot C,
 * 2026-08-27) sont livrées entières, jusqu'à 11,67 s — une fanfare coupée n'est plus une
 * fanfare. Le lecteur n'impose donc plus SA seconde à tout le monde : il joue ce qu'on lui
 * livre, et `SOUND_CUT_MAX_S` n'est plus qu'un PLAFOND DE SÛRETÉ contre un fichier livré trop
 * long.
 *
 * LA COUPE RESTE UNE ENVELOPPE DE GAIN : tenue pleine, puis fondu de sortie sur les
 * dernières 0,25 s. Un `stop()` sec au milieu d'une onde claquerait (discontinuité) — le
 * fondu est la coupe propre, y compris à la fin exacte d'un fichier dont la source a été
 * tronquée au plafond (c'est le cas de l'explosion plasma, source 4,07 s).
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

import {
  gainFromDb,
  type DistanceChain,
  type SoundDraw,
} from './weaponSoundLogic'

/**
 * PLAFOND de durée jouée, en secondes — pas la coupe : la durée d'un son est celle de son
 * FICHIER (règle par catégorie, appliquée à la livraison des assets : armes/lancers/mêlée
 * 1,2 s, explosions et équipements jusqu'à 4 s, répliques et fanfares de FIN DE PARTIE
 * entières). Ce plafond ne mord donc sur AUCUN fichier livré aujourd'hui — il existe pour
 * qu'un asset livré par erreur en pleine longueur (une source de 30 s) ne tienne pas une voix
 * pendant tout un échange.
 *
 * LA RÈGLE EST « PLAFOND = PLUS LONG FICHIER LIVRÉ », arrondi au-dessus : un nombre plus bas
 * tronquerait un son SANS que rien ne le dise, un nombre bien plus haut ne protégerait plus de
 * rien. Historique des relevés (fusion du 2026-08-27, deux lots convergents) : 4,0 s tant que
 * les explosions et équipements étaient les plus longs ; 4 -> 6 s quand la reconstitution des
 * gestes Wwise a révélé qu'un conteneur déclare COMBIEN de fois il se joue ET À QUEL RYTHME
 * (`play_004_mod_mp_ctf_flag_taken_team` 4,588 s, `objective_zone_new` 5,15 s) ; 6 -> 12 s
 * quand les fanfares de FIN DE PARTIE entrent au catalogue — la plus longue, l'égalité, fait
 * 11,67 s. Le plafond n'est pas là pour raccourcir un geste — le tronquer en silence est
 * exactement ce que ce commentaire interdit — mais pour qu'un asset livré par erreur en pleine
 * longueur (une source de 30 s) ne tienne pas une voix pendant tout un échange. C'est le
 * garde-rail `replaySoundAssets.guard.test.ts` qui pose la question à chaque livraison.
 *
 * CE QUE ÇA NE CHANGE PAS : ce qui sature les voix est le TIR, et un tir tient toujours 1,2 s.
 * Les gestes longs (vol de drapeau, déplacement de colline, fanfare de fin) sont les plus
 * rares de la piste.
 */
export const SOUND_CUT_MAX_S = 12.0

/** Durée du fondu de sortie, en secondes (borné à la moitié du son pour les très courts). */
export const SOUND_FADE_S = 0.25

/**
 * Voix simultanées maximum : au-delà, les sons supplémentaires sont sautés. Sur un échange
 * nourri, empiler vingt sources ne raconte rien de plus qu'un mur de bruit.
 *
 * C'EST LE SEUL PLAFOND DE TOUTE LA CHAÎNE SONORE depuis que TOUS les tirs sonnent
 * (décision utilisateur du 2026-08-15 : aucun filtrage éditorial). Ce qu'il coûte, mesuré
 * le même jour par simulation à 1× (une voix tenue 1 s, la durée jouée d'alors) : sur le
 * film témoin 000d5950, 46 sources refusées pour 483 tirs sonores ; sur les 23 artefacts
 * locaux, 4 897 refus pour 17 068 sources, soit 28,7 %. Le relever est un changement d'UN
 * chiffre — à faire si l'écoute le demande, pas avant : c'est une décision d'oreille, pas
 * de code.
 *
 * CE QUE L'ALLONGEMENT DU 2026-08-16 CHANGE POUR CE PLAFOND : rien sur ce qui le fait
 * mordre. Ce qui le sature est le TIR (17 904 tirs au corpus mesuré), et un tir tient
 * toujours 1,2 s. Seules les explosions et les équipements tiennent désormais jusqu'à 4 s,
 * et ce sont les événements les plus rares de la piste — un kill à la grenade, un épisode
 * de camouflage. Le plafond reste inchangé (décision du lot R2.1).
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
 *
 * Le son joue jusqu'au BOUT DE SON FICHIER (c'est la livraison qui règle la durée par
 * catégorie), sauf au-delà du plafond de sûreté. Le fondu est borné à la moitié du son
 * pour les très courts : sans cela, un son de 0,1 s serait un fondu et rien d'autre.
 */
export function soundEnvelope(durationS: number): { fadeStartS: number; stopS: number } {
  const stopS = Math.min(Math.max(durationS, 0), SOUND_CUT_MAX_S)
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
  /** Chaîne de distance (réglage d'instance, page admin). Nulle = AUCUN nœud ajouté :
   *  le chemin du signal reste celui d'origine, source -> enveloppe -> maître. */
  private distGain: GainNode | null = null
  private distFilter: BiquadFilterNode | null = null
  /**
   * LE ROBINET D'ENREGISTREMENT (2026-08-26) : une seconde sortie, branchée EN PARALLÈLE de
   * `ctx.destination`, dont la piste va au `MediaRecorder` de la capture vidéo.
   *
   * Il n'existe que si on l'a demandé (`recordingTrack`) : un rejeu qu'on ne filme pas ne
   * paie pas un nœud de plus. Une fois né il RESTE — un enregistrement suivant reprend la
   * même piste, et le recréer à chaque clip laisserait des nœuds derrière lui.
   */
  private recordDest: MediaStreamAudioDestinationNode | null = null
  /**
   * Le dernier nœud AVANT la sortie : `master` sans chaîne de distance, `distFilter` avec.
   * Le suivre est ce qui permet au robinet de se brancher au BON endroit — brancher `master`
   * alors que la chaîne de distance est posée enregistrerait un son que personne n'entend.
   */
  private out: AudioNode

  /** `volume` est posé À LA CONSTRUCTION : un gain par défaut à 1 ferait passer le premier
   *  son à plein régime avant que la préférence de l'utilisateur ne soit appliquée. */
  constructor(volume: number) {
    this.ctx = new AudioContext()
    this.master = this.ctx.createGain()
    this.master.gain.value = Math.min(Math.max(volume, 0), 1)
    this.out = this.master
    this.connectOut(this.master)
  }

  /**
   * connectOut branche un nœud sur la ou les SORTIES : les haut-parleurs toujours, et le
   * robinet d'enregistrement quand il existe. Les trois points de la classe qui atteignaient
   * `ctx.destination` passent par ici — sans quoi poser la chaîne de distance en cours
   * d'enregistrement couperait le son du clip sans couper celui des haut-parleurs.
   */
  private connectOut(node: AudioNode): void {
    this.out = node
    node.connect(this.ctx.destination)
    if (this.recordDest) node.connect(this.recordDest)
  }

  /**
   * recordingTrack rend la piste audio à joindre à une vidéo, ou `null` si ce navigateur ne
   * sait pas en fabriquer. Le robinet est créé À LA PREMIÈRE DEMANDE, puis conservé.
   */
  recordingTrack(): MediaStreamTrack | null {
    if (typeof this.ctx.createMediaStreamDestination !== 'function') return null
    if (!this.recordDest) {
      this.recordDest = this.ctx.createMediaStreamDestination()
      // Le robinet naît APRÈS le câblage : il faut donc le raccrocher à la sortie courante,
      // celle-là même que les haut-parleurs reçoivent.
      this.out.connect(this.recordDest)
    }
    return this.recordDest.stream.getAudioTracks()[0] ?? null
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

  /**
   * setDistance pose ou retire la chaîne de distance (atténuation + passe-bas) ENTRE le
   * maître et la sortie. Réglage d'instance (admin) : à 0 %, `chain` est nul et les deux
   * nœuds sont déconnectés — le fichier extrait se joue tel quel, exigence « sons purs ».
   */
  setDistance(chain: DistanceChain | null): void {
    this.master.disconnect()
    if (!chain) {
      this.connectOut(this.master)
      return
    }
    if (!this.distGain || !this.distFilter) {
      this.distGain = this.ctx.createGain()
      this.distFilter = this.ctx.createBiquadFilter()
      this.distFilter.type = 'lowpass'
      this.distGain.connect(this.distFilter)
    }
    this.distGain.gain.value = gainFromDb(chain.gainDb)
    this.distFilter.frequency.value = chain.cutoffHz
    this.distFilter.disconnect()
    this.connectOut(this.distFilter)
    this.master.connect(this.distGain)
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
  play(url: string, draw?: SoundDraw): void {
    const buf = this.buffers.get(url)
    if (!buf || this.voices >= SOUND_MAX_VOICES) {
      if (buf === undefined) this.preload([url])
      return
    }
    const t0 = this.ctx.currentTime
    const { fadeStartS, stopS } = soundEnvelope(buf.duration)
    // La VARIATION de cette lecture (fourchettes RANGED du jeu, tirées en amont) : un gain
    // de départ et une vitesse de lecture. Sans tirage, la tenue est à 1 — inchangé.
    const tenue = draw ? gainFromDb(draw.gainDb) : 1
    const gain = this.ctx.createGain()
    gain.gain.setValueAtTime(tenue, t0)
    gain.gain.setValueAtTime(tenue, t0 + fadeStartS)
    gain.gain.linearRampToValueAtTime(0, t0 + stopS)
    const src = this.ctx.createBufferSource()
    src.buffer = buf
    if (draw && draw.playbackRate !== 1) src.playbackRate.value = draw.playbackRate
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
