/**
 * replayVideoEncoder.ts — L'ENCODAGE VIDÉO DE L'EXPORT, et rien d'autre.
 *
 * # POURQUOI PAS `MediaRecorder`, QUI EXISTE DÉJÀ DANS CE DOSSIER
 *
 * `replayRecording.ts` filme la toile PENDANT qu'elle joue : c'est un enregistreur d'écran, et
 * il horodate sur l'horloge murale. On ne peut donc pas le nourrir plus vite que le temps réel
 * — pousser trente images en une seconde produirait une seconde de vidéo, pas une seconde de
 * match. C'est exactement ce que l'export refuse : il RECALCULE le film image par image, aussi
 * vite que la machine suit, et la durée du clip doit rester celle du MATCH.
 *
 * WebCodecs est le seul chemin où l'horodatage est un PARAMÈTRE et non une conséquence. Chaque
 * `VideoFrame` porte son instant (en microsecondes de match) ; le muxeur les remet dans l'ordre
 * du fichier. Le temps de calcul n'entre nulle part dans le résultat.
 *
 * # POURQUOI `mediabunny` ET PLUS `mp4-muxer` (2026-08-28)
 *
 * Les deux viennent du meme auteur, et le second est le successeur du premier. Le changement
 * n'est pas une preference : `mp4-muxer` declare `audio?:` AU SINGULIER — une piste video, une
 * piste audio, pas plus. L'utilisateur veut ses bruitages, ses voix et sa musique SEPARES pour
 * les remonter ; `mediabunny` accepte un nombre illimite de pistes audio en MP4 (verifie :
 * `Mp4OutputFormat.getSupportedTrackCounts()` rend `audio.max = 2^32-1`).
 *
 * CE QUI N'A PAS CHANGE, ET C'EST DELIBERE : l'encodage VIDEO reste le notre — `VideoEncoder`,
 * notre config, notre contre-pression, nos horodatages. `mediabunny` sait encoder lui-meme,
 * mais tout cela est deja mesure et teste, et une reecriture aurait remis en jeu la seule
 * partie du chantier verifiee de bout en bout dans un navigateur. Seule la couche de MUXAGE
 * change, par `EncodedVideoPacketSource` qui prend nos paquets tels quels.
 *
 * L'AUDIO, LUI, PASSE PAR `AudioBufferSource` : on donne un tampon rendu, la bibliotheque
 * encode. C'est ce qui supprime notre `encodeAudioInto` et, avec lui, toute la famille de
 * pieges qu'il portait (configuration AAC refusee de facon asynchrone, `flush()` sur un
 * encodeur ferme, piste declaree mais vide).
 *
 * # CE QUE CE MODULE NE FAIT PAS
 *
 * Il ne dessine rien, ne connaît ni le document du rejeu ni React, et n'écrit aucun fichier :
 * il rend un `Blob`. La boucle qui pose les images vit dans `useReplayExport.ts`, la piste
 * sonore dans `replayAudioMix.ts`, le téléchargement dans `replayCapture.ts`. Ici, seulement :
 * quel codec, à quel débit, et comment on ne noie pas l'encodeur.
 *
 * # LES TROIS DÉCISIONS QUI NE SONT PAS ARBITRAIRES
 *
 * 1. LES DIMENSIONS SONT PAIRES. H.264 échantillonne la chrominance à 4:2:0, c'est-à-dire un
 *    demi-pixel sur chaque axe : une largeur impaire n'a pas de représentation, et l'encodeur
 *    la refuse. La toile du rejeu est dimensionnée en pixels physiques (largeur CSS × DPR) et
 *    tombe donc sur des nombres impairs une fois sur deux. On arrondit VERS LE BAS — rogner une
 *    colonne de pixels est invisible, l'étirer ne l'est pas.
 * 2. LE NIVEAU H.264 SE CALCULE, il ne se choisit pas au jugé. Un niveau déclaré trop bas
 *    produit un fichier que les lecteurs stricts refusent ; trop haut, il ferme la porte aux
 *    lecteurs anciens sans rien apporter. `avcLevelFor` prend le plus BAS qui accepte la
 *    surface ET la cadence demandées.
 * 3. LA CONTRE-PRESSION N'EST PAS UNE OPTIMISATION. `encoder.encode()` ne bloque jamais : sur
 *    un match de dix minutes, une boucle qui ne regarde pas `encodeQueueSize` empile dix-huit
 *    mille images dans la file avant que l'encodeur en ait sorti la moitié, et l'onglet meurt.
 *    On attend que la file redescende (cf. `shouldDrain`).
 */
import {
  AudioBufferSource,
  BufferTarget,
  EncodedPacket,
  EncodedVideoPacketSource,
  Mp4OutputFormat,
  Output,
  QUALITY_HIGH,
} from 'mediabunny'

import { yieldToEvents } from './eventLoopYield'

/**
 * CADENCE DE L'EXPORT, en images par seconde.
 *
 * VOLONTAIREMENT DISTINCTE de `CAPTURE_FPS` (`replayRecording.ts`), qui vaut le même nombre
 * pour une raison qui n'est pas la même : là-bas c'est un DÉBIT DEMANDÉ à un flux de toile,
 * que le navigateur sert au mieux ; ici c'est la définition même de l'axe du temps du fichier —
 * une image tous les 1/30 de seconde de MATCH, ni plus ni moins. Les fusionner ferait croire
 * qu'un réglage de l'un vaut pour l'autre.
 *
 * 30 et pas 60 : ce que montre le rejeu, ce sont des points qui glissent sur une carte. Doubler
 * la cadence doublerait le poids du fichier et le temps de calcul pour un œil qui n'y verrait
 * rien.
 */
export const EXPORT_FPS = 30

/**
 * Une image-clé toutes les 2 secondes de match. C'est ce qui rend le clip NAVIGABLE : sans
 * image-clé régulière, déplacer le curseur dans le fichier oblige le lecteur à tout rejouer
 * depuis le début. Plus serré gonflerait le fichier sans servir personne.
 */
export const KEYFRAME_EVERY = EXPORT_FPS * 2

/**
 * Profondeur de file au-delà de laquelle la boucle attend. Deux images de marge : assez pour
 * que l'encodeur ne chôme jamais entre deux tracés, assez peu pour que la mémoire retenue reste
 * celle de quelques images et non celle d'un match.
 */
export const ENCODE_QUEUE_MAX = 2

/** Bits par pixel et par image. Une carte 2D à aplats se compresse bien mieux qu'un film. */
const BITS_PER_PIXEL = 0.1
/** Débit plancher et plafond, en bits par seconde : sous 2 Mb/s le texte du HUD bave. */
const BITRATE_MIN = 2_000_000
const BITRATE_MAX = 40_000_000

/**
 * Les niveaux H.264, du plus bas au plus haut, avec ce que chacun accepte : `maxFS` en
 * macroblocs par image, `maxMBps` en macroblocs par seconde (tableau A-1 de la norme, les
 * seuls niveaux qu'un export de rejeu peut atteindre). `hex` est l'octet du nom de codec.
 */
const AVC_LEVELS: readonly { hex: string; maxFS: number; maxMBps: number }[] = [
  { hex: '1e', maxFS: 1620, maxMBps: 40_500 }, // 3.0
  { hex: '1f', maxFS: 3600, maxMBps: 108_000 }, // 3.1
  { hex: '20', maxFS: 5120, maxMBps: 216_000 }, // 3.2
  { hex: '28', maxFS: 8192, maxMBps: 245_760 }, // 4.0
  { hex: '2a', maxFS: 8704, maxMBps: 522_240 }, // 4.2
  { hex: '32', maxFS: 22_080, maxMBps: 589_824 }, // 5.0
  { hex: '33', maxFS: 36_864, maxMBps: 983_040 }, // 5.1
  { hex: '34', maxFS: 36_864, maxMBps: 2_073_600 }, // 5.2
  { hex: '3c', maxFS: 139_264, maxMBps: 4_177_920 }, // 6.0
]

/** Profil HIGH (`6400`) : le profil que tout lecteur matériel de cette décennie décode. */
const AVC_PROFILE = 'avc1.6400'

/**
 * evenSize arrondit VERS LE BAS à un nombre pair, plancher à 2 (cf. décision 1 de l'en-tête).
 * Une dimension nulle ou négative n'a pas de sens pour un encodeur : elle remonte à 2, et
 * l'appelant qui lui a servi ça a un autre problème que l'arrondi.
 */
export function evenSize(n: number): number {
  const floored = Math.floor(n)
  if (!Number.isFinite(floored) || floored < 2) return 2
  return floored - (floored % 2)
}

/** Surface en macroblocs 16x16 : l'unité dans laquelle la norme H.264 compte tout. */
function frameMacroblocks(width: number, height: number): number {
  return Math.ceil(width / 16) * Math.ceil(height / 16)
}

/**
 * avcLevelFor rend le nom de codec complet pour cette image et cette cadence : le PLUS BAS
 * niveau qui accepte la surface et le débit de macroblocs (cf. décision 2 de l'en-tête).
 *
 * Au-delà du plus haut niveau du tableau (une toile démesurée), on rend quand même le dernier :
 * un fichier au niveau sous-déclaré reste lisible partout où le décodeur ne vérifie pas, là où
 * refuser l'export ne rendrait service à personne. Le cas ne se produit pas sous 8K.
 */
export function avcLevelFor(width: number, height: number, fps: number): string {
  const fs = frameMacroblocks(width, height)
  const mbps = fs * fps
  const level = AVC_LEVELS.find((l) => fs <= l.maxFS && mbps <= l.maxMBps)
  return `${AVC_PROFILE}${level?.hex ?? AVC_LEVELS[AVC_LEVELS.length - 1].hex}`
}

/** videoBitrate : le débit visé, borné (cf. `BITRATE_MIN` / `BITRATE_MAX`). */
export function videoBitrate(width: number, height: number, fps: number): number {
  const raw = Math.round(width * height * fps * BITS_PER_PIXEL)
  return Math.min(BITRATE_MAX, Math.max(BITRATE_MIN, raw))
}

/**
 * videoEncoderConfig assemble la configuration complète. PURE : c'est elle qu'on teste, et
 * c'est elle qu'on passe à `VideoEncoder.isConfigSupported` avant d'ouvrir quoi que ce soit.
 *
 * Les dimensions reçues sont celles de la TOILE ; celles qui sortent sont paires.
 */
export function videoEncoderConfig(
  width: number,
  height: number,
  fps: number = EXPORT_FPS,
): VideoEncoderConfig {
  const w = evenSize(width)
  const h = evenSize(height)
  return {
    codec: avcLevelFor(w, h, fps),
    width: w,
    height: h,
    framerate: fps,
    bitrate: videoBitrate(w, h, fps),
    // `avc` et non `annexb` : c'est la forme que le conteneur MP4 attend.
    avc: { format: 'avc' },
  }
}

/**
 * shouldDrain dit si la boucle doit attendre avant de pousser l'image suivante (décision 3).
 * Séparée de la boucle pour être testable sans encodeur.
 */
export function shouldDrain(queueSize: number): boolean {
  return queueSize > ENCODE_QUEUE_MAX
}

/** isKeyFrame : l'image `index` (0 = la première) ouvre-t-elle un nouveau point d'entrée ? */
export function isKeyFrame(index: number): boolean {
  return index % KEYFRAME_EVERY === 0
}

/**
 * canExportVideo dit si CE navigateur sait encoder hors temps réel.
 *
 * Les trois conditions sont indissociables : l'encodeur, le type d'image qu'on lui pousse, et
 * la lecture asynchrone de ses capacités. Il en manque une, il n'y a pas d'export possible — et
 * c'est le bouton d'enregistrement TEMPS RÉEL qui reprend la main (repli, décision D5 du plan).
 *
 * Rend `false` sous jsdom, où rien de tout cela n'existe : c'est ce qui permet aux tests de ce
 * module de rester des tests purs.
 */
export function canExportVideo(): boolean {
  if (typeof VideoEncoder === 'undefined') return false
  if (typeof VideoFrame === 'undefined') return false
  return typeof VideoEncoder.isConfigSupported === 'function'
}

/**
 * canExportAudio dit si ce navigateur sait encoder une piste sonore.
 *
 * SE TESTE AVANT D'OUVRIR LE CONTENEUR, jamais après : le MP4 annonce ses pistes une fois pour
 * toutes. Déclarer une piste AAC puis découvrir qu'`AudioEncoder` manque produisait un fichier
 * portant une piste VIDE — selon le lecteur : muette, avertissement, ou refus d'ouvrir.
 */
export function canExportAudio(): boolean {
  return typeof AudioEncoder !== 'undefined' && typeof AudioData !== 'undefined'
}

/**
 * Débit de la piste sonore, en bits par seconde. 128 kb/s en AAC stéréo : au-dessus du seuil
 * où l'oreille distingue encore l'encodage sur des sons courts et secs (tirs, explosions),
 * et sans commune mesure avec le débit vidéo — la piste ne pèse rien dans le fichier.
 */
const AUDIO_BITRATE = 128_000

/** La configuration AAC visee : AAC-LC, le profil que tout lecteur decode. */
export function audioEncoderConfig(shape: AudioTrackShape): AudioEncoderConfig {
  return {
    codec: 'mp4a.40.2',
    sampleRate: shape.sampleRate,
    numberOfChannels: shape.numberOfChannels,
    bitrate: AUDIO_BITRATE,
  }
}

/**
 * audioTrackUsable dit si ce navigateur sait REELLEMENT encoder cette piste.
 *
 * `canExportAudio` ne repond qu'a « l'API existe-t-elle ». Elle peut exister et REFUSER la
 * configuration — et ce refus est ASYNCHRONE : `configure()` ne leve pas, il passe l'encodeur
 * en erreur et le ferme, si bien que la panne n'apparait qu'au `flush()`, sous la forme
 * trompeuse « Encoder must be configured first ». C'est exactement ce qui s'est produit en
 * recette le 2026-08-28.
 *
 * La question se pose donc AVANT d'ouvrir le conteneur, qui declare ses pistes une fois pour
 * toutes : mieux vaut un clip muet annonce qu'un export perdu.
 */
export async function audioTrackUsable(shape: AudioTrackShape): Promise<boolean> {
  if (!canExportAudio()) return false
  if (typeof AudioEncoder.isConfigSupported !== 'function') return false
  try {
    const probe = await AudioEncoder.isConfigSupported(audioEncoderConfig(shape))
    return probe.supported === true
  } catch {
    // Une configuration REFUSEE leve ici (config invalide) : c'est une reponse, pas une panne.
    return false
  }
}

/** Ce que la boucle d'export reçoit : trois gestes, et aucun état à tenir de son côté. */
export interface VideoExportSink {
  /** Pousse la toile TELLE QU'ELLE EST à cet instant. Attend si la file est pleine. */
  addFrame: (canvas: HTMLCanvasElement, index: number) => Promise<void>
  /** Vide la file, referme le conteneur, rend le fichier. */
  finish: () => Promise<Blob>
  /**
   * `false` = les pistes sonores demandees n'ont PAS pu etre declarees (encodeur AAC absent ou
   * configuration refusee). Le clip sortira muet, et l'appelant doit le DIRE.
   */
  audioEnabled: boolean
  /**
   * Remet les tampons rendus, DANS L'ORDRE des noms declares a l'ouverture. A n'appeler QUE si
   * `audioEnabled`, et une seule fois.
   */
  addAudioTracks: (tracks: readonly AudioTrackInput[]) => Promise<void>
  /** Referme tout sans rien rendre (annulation, erreur). Ne lève jamais. */
  abort: () => void
}

export interface VideoExportOptions {
  width: number
  height: number
  fps?: number
  /**
   * COMBIEN de pistes sonores le clip portera. Declare a l'ouverture ou jamais : le conteneur
   * ecrit sa table des pistes une fois pour toutes. `0` = clip muet, ce qui reste un resultat.
   *
   * Les noms sont donnes ici parce qu'ils font partie de la DECLARATION ; les tampons, eux,
   * arrivent plus tard (`addAudioTracks`), une fois le mixage rendu.
   */
  audioTracks?: readonly string[]
}

/** La forme d'une piste sonore : ce que le conteneur annonce avant le premier octet. */
export interface AudioTrackShape {
  sampleRate: number
  numberOfChannels: number
}

/**
 * Une piste sonore a ecrire dans le clip : son nom, et le tampon deja rendu.
 *
 * L'ORDRE DES PISTES N'EST PAS DECORATIF. La PREMIERE est celle que tout lecteur ordinaire
 * joue — un navigateur n'expose pas les autres. C'est donc le MIXAGE COMPLET qui doit venir en
 * premier ; les familles separees suivent, pour qui ouvre le fichier dans un montage. Livrer
 * les familles SANS le mixage ferait entendre les seuls bruitages, sans musique ni voix : une
 * regression pour tout le monde sauf le monteur.
 */
export interface AudioTrackInput {
  /** Nom lisible de la piste (« Mixage », « Bruitages », « Voix », « Musique »). */
  name: string
  buffer: AudioBuffer
}

/**
 * openVideoExport ouvre l'encodeur et le conteneur, et rend les trois gestes de la boucle.
 *
 * `null` = ce navigateur ne sait pas encoder CETTE image (configuration refusée) : l'appelant
 * bascule sur le repli. On ne lève pas — un export impossible n'est pas une panne, c'est une
 * capacité absente, et le plan (D5) veut qu'elle se traite par un autre bouton.
 */
/**
 * La forme de reference des pistes sonores : celle que rend le mixage hors ligne
 * (`replayAudioMix.ts`). Elle ne sert qu'a PROUVER que l'encodeur AAC l'accepte, AVANT de
 * declarer quoi que ce soit dans le conteneur — une piste declaree puis refusee perd tout
 * l'export.
 */
const MIX_TRACK_SHAPE: AudioTrackShape = { sampleRate: 48_000, numberOfChannels: 2 }

export async function openVideoExport(o: VideoExportOptions): Promise<VideoExportSink | null> {
  if (!canExportVideo()) return null
  const fps = o.fps ?? EXPORT_FPS
  const config = videoEncoderConfig(o.width, o.height, fps)
  const support = await VideoEncoder.isConfigSupported(config)
  if (!support.supported) return null
  const noms = o.audioTracks ?? []
  // LES PISTES SONORES SE PROUVENT AVANT D'ETRE DECLAREES (cf. `audioTrackUsable`) : le
  // conteneur ecrit sa table des pistes une fois pour toutes, et une piste declaree que
  // l'encodeur refusera ensuite perd TOUT l'export.
  const audioOk = noms.length > 0 && (await audioTrackUsable(MIX_TRACK_SHAPE))

  const output = new Output({ format: new Mp4OutputFormat({ fastStart: 'in-memory' }), target: new BufferTarget() })
  const videoSource = new EncodedVideoPacketSource('avc')
  output.addVideoTrack(videoSource, { frameRate: fps })
  // Les pistes AUDIO sont ajoutees ICI, vides : leur EXISTENCE et leur ORDRE se declarent avant
  // le premier octet, leur contenu arrive plus tard (`addAudioTracks`).
  const audioSources = audioOk
    ? noms.map((name) => {
        const src = new AudioBufferSource({ codec: 'aac', quality: QUALITY_HIGH })
        output.addAudioTrack(src, { name })
        return src
      })
    : []
  await output.start()

  // La première erreur gagne et sera relancée par `finish` : une erreur d'encodeur arrive de
  // façon asynchrone, hors de la pile de l'appelant, et l'avaler rendrait un fichier tronqué
  // sans que rien ne le dise.
  let failure: Error | null = null
  const encoder = new VideoEncoder({
    output: (chunk, meta) => {
      void videoSource.add(EncodedPacket.fromEncodedChunk(chunk), meta)
    },
    error: (err) => {
      failure = err instanceof Error ? err : new Error(String(err))
    },
  })
  encoder.configure(config)
  return makeSink(encoder, output, videoSource, audioSources, fps, () => failure, {
    width: config.width,
    height: config.height,
  })
}

/**
 * makeSink assemble les trois gestes. HORS de `openVideoExport` pour tenir la règle des 80
 * lignes du dépôt, et parce que l'ouverture (asynchrone, faillible) et l'usage (synchrone,
 * répété dix-huit mille fois) n'ont pas la même nature.
 */
function makeSink(
  encoder: VideoEncoder,
  output: Output,
  videoSource: EncodedVideoPacketSource,
  audioSources: readonly AudioBufferSource[],
  fps: number,
  failureOf: () => Error | null,
  size: { width: number; height: number },
): VideoExportSink {
  let closed = false
  const frameDurationUs = Math.round(1_000_000 / fps)

  const addFrame = async (canvas: HTMLCanvasElement, index: number) => {
    const failed = failureOf()
    if (failed) throw failed
    if (closed) return
    // L'HORODATAGE EST CALCULÉ, JAMAIS MESURÉ (cf. l'en-tête) : c'est ce qui rend la durée du
    // clip indépendante du temps de calcul.
    const frame = new VideoFrame(canvas, {
      timestamp: index * frameDurationUs,
      duration: frameDurationUs,
      // LE RECADRAGE EST EXPLICITE. La toile fait souvent une taille IMPAIRE (largeur CSS x DPR
      // fractionnaire) alors que la config est paire : sans `visibleRect`, on pousse une image
      // 1919x601 dans un encodeur configuré en 1918x600, et le comportement dépend alors de la
      // version du navigateur — mise à l'échelle implicite ici, erreur d'encodeur ailleurs.
      visibleRect: { x: 0, y: 0, width: size.width, height: size.height },
    })
    try {
      encoder.encode(frame, { keyFrame: isKeyFrame(index) })
    } finally {
      // TOUJOURS, même si `encode` lève : une `VideoFrame` non fermée retient sa mémoire GPU,
      // et il en passe trente par seconde de match.
      frame.close()
    }
    while (shouldDrain(encoder.encodeQueueSize)) {
      // JAMAIS `setTimeout` ICI (cf. `eventLoopYield.ts`) : bridé à une seconde en onglet
      // caché, il rendait l'export dix fois plus lent que le temps réel qu'il encode.
      await yieldToEvents()
      const late = failureOf()
      if (late) throw late
    }
  }

  /**
   * addAudioTracks remet les tampons rendus, DANS L'ORDRE des noms declares a l'ouverture.
   *
   * `AudioBufferSource.close()` est appele apres chaque tampon : sans lui, la piste reste
   * ouverte et `output.finalize()` attendrait indefiniment de quoi la remplir.
   */
  const addAudioTracks = async (tracks: readonly AudioTrackInput[]) => {
    for (let i = 0; i < audioSources.length; i++) {
      const piste = tracks[i]
      if (piste) await audioSources[i].add(piste.buffer)
      audioSources[i].close()
    }
  }

  const finish = async (): Promise<Blob> => {
    await encoder.flush()
    const failed = failureOf()
    if (failed) throw failed
    // LES SOURCES SE FERMENT AVANT LA FINALISATION : une piste declaree mais jamais close
    // laisse `finalize()` attendre un contenu qui ne viendra pas.
    for (const src of audioSources) src.close()
    videoSource.close()
    await output.finalize()
    closed = true
    encoder.close()
    const buffer = (output.target as BufferTarget).buffer
    return new Blob([buffer ?? new ArrayBuffer(0)], { type: 'video/mp4' })
  }

  const abort = () => {
    if (closed) return
    closed = true
    // `close()` sur un encodeur déjà en erreur lève : l'annulation ne doit jamais masquer la
    // cause d'un échec par une seconde erreur.
    try {
      encoder.close()
    } catch {
      // Rien à dire : on annule, il n'y a pas de fichier à rendre ni d'appelant à prévenir.
    }
  }

  return { addFrame, addAudioTracks, audioEnabled: audioSources.length > 0, finish, abort }
}


