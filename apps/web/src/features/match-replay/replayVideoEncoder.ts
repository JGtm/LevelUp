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
import { ArrayBufferTarget, Muxer } from 'mp4-muxer'

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

/** Ce que la boucle d'export reçoit : trois gestes, et aucun état à tenir de son côté. */
export interface VideoExportSink {
  /** Pousse la toile TELLE QU'ELLE EST à cet instant. Attend si la file est pleine. */
  addFrame: (canvas: HTMLCanvasElement, index: number) => Promise<void>
  /** Vide la file, referme le conteneur, rend le fichier. */
  finish: () => Promise<Blob>
  /**
   * Encode et muxe la piste sonore rendue hors ligne. À n'appeler QUE si `audio` a été déclaré
   * à l'ouverture, et une seule fois. Sans piste déclarée, ne fait rien.
   */
  addAudioBuffer: (buffer: AudioBuffer) => Promise<void>
  /** Referme tout sans rien rendre (annulation, erreur). Ne lève jamais. */
  abort: () => void
}

export interface VideoExportOptions {
  width: number
  height: number
  fps?: number
  /**
   * La PISTE SONORE, déclarée à l'ouverture ou jamais : le conteneur MP4 écrit sa table des
   * pistes une fois pour toutes, et une piste ajoutée après coup n'y aurait pas de place.
   * Absente = clip muet, ce qui reste un résultat.
   */
  audio?: AudioTrackShape
}

/** La forme de la piste sonore : ce que le conteneur doit annoncer avant le premier octet. */
export interface AudioTrackShape {
  sampleRate: number
  numberOfChannels: number
}

/**
 * openVideoExport ouvre l'encodeur et le conteneur, et rend les trois gestes de la boucle.
 *
 * `null` = ce navigateur ne sait pas encoder CETTE image (configuration refusée) : l'appelant
 * bascule sur le repli. On ne lève pas — un export impossible n'est pas une panne, c'est une
 * capacité absente, et le plan (D5) veut qu'elle se traite par un autre bouton.
 */
export async function openVideoExport(o: VideoExportOptions): Promise<VideoExportSink | null> {
  if (!canExportVideo()) return null
  const fps = o.fps ?? EXPORT_FPS
  const config = videoEncoderConfig(o.width, o.height, fps)
  const support = await VideoEncoder.isConfigSupported(config)
  if (!support.supported) return null

  const muxer = new Muxer({
    target: new ArrayBufferTarget(),
    video: { codec: 'avc', width: config.width, height: config.height },
    ...(o.audio
      ? {
          audio: {
            codec: 'aac' as const,
            sampleRate: o.audio.sampleRate,
            numberOfChannels: o.audio.numberOfChannels,
          },
        }
      : {}),
    // « in-memory » : l'en-tête de lecture se pose EN TÊTE du fichier une fois tout connu. Sans
    // lui, l'index atterrit à la fin et un lecteur qui ouvre le fichier en flux ne sait pas
    // combien de temps il dure — le clip s'ouvre alors sur une durée inconnue.
    fastStart: 'in-memory',
  })
  // La première erreur gagne et sera relancée par `finish` : une erreur d'encodeur arrive de
  // façon asynchrone, hors de la pile de l'appelant, et l'avaler rendrait un fichier tronqué
  // sans que rien ne le dise.
  let failure: Error | null = null
  const encoder = new VideoEncoder({
    output: (chunk, meta) => muxer.addVideoChunk(chunk, meta),
    error: (err) => {
      failure = err instanceof Error ? err : new Error(String(err))
    },
  })
  encoder.configure(config)
  return makeSink(encoder, muxer, fps, () => failure, o.audio ?? null)
}

/**
 * makeSink assemble les trois gestes. HORS de `openVideoExport` pour tenir la règle des 80
 * lignes du dépôt, et parce que l'ouverture (asynchrone, faillible) et l'usage (synchrone,
 * répété dix-huit mille fois) n'ont pas la même nature.
 */
function makeSink(
  encoder: VideoEncoder,
  muxer: Muxer<ArrayBufferTarget>,
  fps: number,
  failureOf: () => Error | null,
  audio: AudioTrackShape | null,
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

  const finish = async (): Promise<Blob> => {
    await encoder.flush()
    const failed = failureOf()
    if (failed) throw failed
    muxer.finalize()
    closed = true
    encoder.close()
    return new Blob([muxer.target.buffer], { type: 'video/mp4' })
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

  const addAudioBuffer = (buffer: AudioBuffer) =>
    audio ? encodeAudioInto(muxer, buffer, audio) : Promise.resolve()

  return { addFrame, addAudioBuffer, finish, abort }
}

/**
 * Débit de la piste sonore, en bits par seconde. 128 kb/s en AAC stéréo : au-dessus du seuil
 * où l'oreille distingue encore l'encodage sur des sons courts et secs (tirs, explosions),
 * et sans commune mesure avec le débit vidéo — la piste ne pèse rien dans le fichier.
 */
const AUDIO_BITRATE = 128_000

/**
 * Taille d'un paquet poussé à l'encodeur, en échantillons par canal. 1024 est la taille de
 * trame native de l'AAC : la choisir évite à l'encodeur de recouper ce qu'on lui donne.
 */
const AUDIO_CHUNK_FRAMES = 1024

/**
 * encodeAudioInto encode un tampon rendu hors ligne et le muxe.
 *
 * LE TAMPON EST DÉJÀ COMPLET quand on arrive ici : `OfflineAudioContext` a tout rendu d'un
 * coup, bien plus vite que la durée du clip. Il ne reste qu'à le découper en paquets et à les
 * horodater — comme pour l'image, l'horodatage est CALCULÉ, jamais mesuré.
 *
 * LES CANAUX SONT ENTRELACÉS EN `f32-planar` : c'est la disposition que rend `getChannelData`,
 * canal par canal, et la recopier telle quelle évite un entrelacement inutile.
 */
async function encodeAudioInto(
  muxer: Muxer<ArrayBufferTarget>,
  buffer: AudioBuffer,
  shape: AudioTrackShape,
): Promise<void> {
  if (typeof AudioEncoder === 'undefined' || typeof AudioData === 'undefined') return
  let failure: Error | null = null
  const encoder = new AudioEncoder({
    output: (chunk, meta) => muxer.addAudioChunk(chunk, meta),
    error: (err) => {
      failure = err instanceof Error ? err : new Error(String(err))
    },
  })
  encoder.configure({
    // `mp4a.40.2` = AAC-LC, le profil que tout lecteur décode.
    codec: 'mp4a.40.2',
    sampleRate: shape.sampleRate,
    numberOfChannels: shape.numberOfChannels,
    bitrate: AUDIO_BITRATE,
  })
  const channels: Float32Array[] = []
  for (let c = 0; c < shape.numberOfChannels; c++) {
    // Un tampon rendu avec moins de canaux que la piste déclarée : on recopie le dernier
    // plutôt que de laisser un canal muet — un clip à moitié silencieux s'entend, pas un
    // canal dupliqué.
    channels.push(buffer.getChannelData(Math.min(c, buffer.numberOfChannels - 1)))
  }
  for (let offset = 0; offset < buffer.length; offset += AUDIO_CHUNK_FRAMES) {
    if (failure) break
    const frames = Math.min(AUDIO_CHUNK_FRAMES, buffer.length - offset)
    const data = new Float32Array(frames * shape.numberOfChannels)
    for (let c = 0; c < shape.numberOfChannels; c++) {
      data.set(channels[c].subarray(offset, offset + frames), c * frames)
    }
    const audio = new AudioData({
      format: 'f32-planar',
      sampleRate: shape.sampleRate,
      numberOfFrames: frames,
      numberOfChannels: shape.numberOfChannels,
      timestamp: Math.round((offset / shape.sampleRate) * 1_000_000),
      data,
    })
    try {
      encoder.encode(audio)
    } finally {
      // TOUJOURS refermer : un `AudioData` non fermé retient sa mémoire, et il en passe une
      // cinquantaine par seconde de clip.
      audio.close()
    }
    if (encoder.encodeQueueSize > ENCODE_QUEUE_MAX) await yieldToEvents()
  }
  await encoder.flush()
  encoder.close()
  if (failure) throw failure
}
