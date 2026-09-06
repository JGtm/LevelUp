/**
 * useReplayCapture — LA COUTURE REACT DE LA CAPTURE : ce que la barre de lecture commande, et
 * ce que le canvas doit lui prêter pour cela.
 *
 * POURQUOI UN HOOK PLUTÔT QU'UN HANDLER DANS LE CANVAS. `ReplayCanvas.tsx` vit sous un seuil
 * de taille (`max-lines` eslint, R5, neuf extractions imposées à ce jour) : tout ce
 * qui n'est pas du DESSIN en sort. Le canvas ne prête donc que sa TOILE, son HORLOGE et son
 * état de lecture, et reçoit en retour un objet unique qu'il repasse tel quel à la barre —
 * exactement le patron du son (`useReplaySound` -> `ReplaySound`), et pour la même raison :
 * brancher une commande de plus ne doit pas coûter une ligne de canvas.
 *
 * CE QUE CE FICHIER NE DÉCIDE PAS : le nom du fichier, le téléchargement et la lecture des
 * pixels sont de la logique pure et vivent dans `replayCapture.ts` ; le choix du conteneur
 * vidéo, dans `replayRecording.ts` ; la piste audio, dans `replayAudio.ts` (le lecteur ouvre
 * une seconde sortie en parallèle des haut-parleurs). Ici, seule la couture.
 *
 * DEUX SOUS-HOOKS, pour la raison écrite dans `useReplaySound.ts` avant lui (CLAUDE.md n° 5,
 * fonction ≤ 80 lignes) : l'image et la vidéo n'ont en commun que leur destination — un
 * fichier — et les mêler dans une seule fonction n'aurait fait qu'une fonction longue.
 *
 * # ON FILME L'ÉCRAN, ET C'EST ASSUMÉ
 *
 * L'enregistrement capture ce que la toile MONTRE, image par image, pendant qu'elle le montre.
 * Changer de vitesse ou déplacer le curseur en cours d'enregistrement se voit donc dans le
 * fichier (décision 4) : ce n'est pas un défaut, c'est le contrat, et l'infobulle du bouton le
 * dit. Un rendu hors écran à vitesse fixe serait un autre produit — et un autre chantier.
 *
 * # UN SEUL CHEMIN DE SORTIE
 *
 * Trois gestes arrêtent l'enregistrement, et les trois passent par `stopRecording` : le second
 * clic sur le bouton, la mise en pause, et la fin du film. C'est ce qui garantit qu'un clip est
 * toujours assemblé et remis une fois — jamais zéro (un enregistrement oublié qui tourne dans
 * le vide), jamais deux (deux téléchargements pour un seul clip). Symétriquement, un clic sur
 * un rejeu EN PAUSE lance la lecture (décision 3) : filmer une image figée n'a aucun sens.
 */
import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from 'react'

import { buildCaptureFilename, captureCanvasImage, triggerDownload } from './replayCapture'
import {
  CAPTURE_FPS,
  canRecordCanvas,
  isVideoTypeSupported,
  pickVideoMimeType,
} from './replayRecording'
import { frameToMs } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'
import { useReplayExport, type ReplayExport, type ReplayExportOptions } from './useReplayExport'
import type { ExportOutcome } from './exportOverlayPanels'
import type { ReplayWindowBounds } from '../model/replayWindow'
import type { ReplayLocale } from '../i18n/i18n'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { ReplayScoreDocument } from '@/lib/replay/scoreTimeline'

export interface ReplayCaptureOptions {
  /** La toile du rejeu : c'est elle, et elle seule, qui porte tout ce qui se voit. */
  canvasRef: RefObject<HTMLCanvasElement | null>
  /** Le document : il donne l'identifiant du match et la durée d'une image. */
  doc: ReplayDocumentReady
  /** L'image courante, partagée avec le dessin — lue au clic, jamais au rendu. */
  frameRef: RefObject<number>
  /** L'état de lecture. Sa RETOMBÉE à `false` clôt l'enregistrement (pause, fin du film). */
  playing: boolean
  /** Lance la lecture. Appelé au démarrage si le rejeu est en pause (décision 3). */
  play: () => void
  /**
   * La piste audio du rejeu, DEMANDÉE au démarrage de l'enregistrement (cf. `ReplaySound`).
   * `null` = son coupé ou lecteur pas né : le clip sort muet, ce qui est le cas nominal
   * puisque le son du rejeu est coupé par défaut.
   */
  audioTrack?: () => MediaStreamTrack | null
  /**
   * CE QUE L'EXPORT HORS TEMPS REEL DEMANDE EN PLUS (cf. `useReplayExport`). Absents, seules
   * l'image et la video temps reel restent disponibles — l'export ne se rend simplement pas.
   */
  redraw?: () => void
  playWindow?: ReplayWindowBounds | null
  scoreboard?: readonly MatchScoreboardRow[]
  xuidMeta?: XuidMeta
  /** Le verdict du backend : sans lui, pas d'ecran de fin dans le clip (parite DOM). */
  outcome?: ExportOutcome | null
  locale?: ReplayLocale
  /** La piste sonore du rejeu et son volume, pour le mixage hors ligne (`useReplaySound`). */
  soundTrack?: ReplayExportOptions['soundTrack']
  soundVolume?: number
}

/** Ce que la barre de lecture reçoit (même forme que `ReplaySound` : un objet, pas des props). */
export interface ReplayCapture {
  /** Télécharge la scène courante en PNG. Sans toile lisible : ne fait rien, ne casse rien. */
  captureImage: () => void
  /**
   * `false` = ce navigateur ne sait pas filmer une toile (pas de `MediaRecorder`, pas de
   * `captureStream`, ou aucun conteneur accepté). Le bouton ne se rend alors PAS du tout
   * (décision 7) : une commande grisée laisserait croire à une panne réparable.
   */
  recordingSupported: boolean
  recording: boolean
  /** Démarre, ou arrête ET télécharge. Le même bouton, les deux sens. */
  toggleRecording: () => void
  /**
   * L'EXPORT HORS TEMPS RÉEL. `null` quand la page ne lui a pas donné de quoi peindre les
   * surimpressions — la barre de lecture n'affiche alors pas la commande.
   */
  videoExport: ReplayExport | null
}

/** Nomme un fichier de capture sur l'instant de match COURANT, dans l'extension demandée. */
type FilenameFor = (ext: string) => string

/**
 * openRecording assemble le flux, ouvre l'enregistreur et le démarre. HORS REACT : rien ici
 * n'est un état de composant, seulement le graphe média — c'est ce qui garde le hook court.
 *
 * `onClosed` reçoit le clip assemblé (ou `null` si rien n'a été enregistré) et son nom. Il est
 * appelé une seule fois, à la fermeture, et c'est le seul endroit d'où un fichier peut sortir.
 */
function openRecording(
  canvas: HTMLCanvasElement,
  filenameFor: FilenameFor,
  audioTrack: (() => MediaStreamTrack | null) | undefined,
  onClosed: (clip: Blob | null, filename: string) => void,
): MediaRecorder | null {
  const choice = pickVideoMimeType(isVideoTypeSupported)
  if (!choice) return null
  const canvasStream = canvas.captureStream(CAPTURE_FPS)
  // LE SON REJOINT L'IMAGE ICI, ET SEULEMENT ICI (décision 6) : la piste est demandée au
  // DÉMARRAGE, donc activer le son ensuite n'ajoute rien au clip en cours. Sans piste — son
  // coupé, ce qui est le cas par défaut — on enregistre le flux de la toile tel quel.
  const audio = audioTrack?.() ?? null
  const stream =
    audio && typeof MediaStream === 'function'
      ? new MediaStream([...canvasStream.getTracks(), audio])
      : canvasStream
  const recorder = new MediaRecorder(stream, { mimeType: choice.mime })
  const chunks: Blob[] = []
  // Le nom est FIGÉ au démarrage : il porte l'instant où le clip commence, pas celui où il
  // s'arrête. Un clip nommé sur sa fin ne se replacerait pas dans le match.
  const filename = filenameFor(choice.ext)
  recorder.ondataavailable = (e: BlobEvent) => {
    if (e.data && e.data.size > 0) chunks.push(e.data)
  }
  recorder.onstop = () => {
    // LES PISTES DE LA TOILE SE COUPENT ICI, pas avant : la capture doit tourner jusqu'à la
    // dernière tranche, sans quoi le clip perdrait sa fin. LA PISTE AUDIO, elle, N'EST PAS
    // COUPÉE — elle appartient au lecteur de son, qui vit plus longtemps que le clip ; la
    // fermer ici rendrait muet tout enregistrement suivant.
    for (const track of canvasStream.getTracks()) track.stop()
    onClosed(chunks.length > 0 ? new Blob(chunks, { type: choice.mime }) : null, filename)
  }
  recorder.start()
  return recorder
}

/** Sous-hook : la capture d'image. Rien d'asynchrone à tenir, donc pas d'état. */
function useImageCapture(
  canvasRef: RefObject<HTMLCanvasElement | null>,
  filenameFor: FilenameFor,
): () => void {
  return useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    // LE NOM EST FIGÉ AVANT L'ENCODAGE : `toBlob` est asynchrone, et l'image continue de
    // courir pendant ce temps. Nommer après rendrait un fichier daté d'un instant qui n'est
    // pas celui qu'il montre.
    const filename = filenameFor('png')
    void captureCanvasImage(canvas).then((blob) => {
      // Pas de blob = toile vierge ou encodage refusé : rien à remettre. Un fichier vide
      // serait pire que pas de fichier (cf. `captureCanvasImage`).
      if (blob) triggerDownload(blob, filename)
    })
  }, [canvasRef, filenameFor])
}

/** Sous-hook : l'enregistrement vidéo, son état, et ses trois sorties (cf. l'en-tête). */
function useVideoRecording(
  o: ReplayCaptureOptions,
  filenameFor: FilenameFor,
): { supported: boolean; recording: boolean; toggle: () => void } {
  const { canvasRef, playing, play, audioTrack } = o
  const [recording, setRecording] = useState(false)
  const recorderRef = useRef<MediaRecorder | null>(null)
  // LE DÉMONTAGE NE TÉLÉCHARGE PAS. Quitter la page pendant un enregistrement doit refermer
  // l'enregistreur, pas déposer un fichier que plus personne n'attend.
  const liveRef = useRef(true)
  // La capacité ne dépend que du navigateur : elle ne change pas d'un rendu à l'autre.
  const supported = useMemo(() => canRecordCanvas(), [])

  const stop = useCallback(() => {
    const recorder = recorderRef.current
    if (!recorder || recorder.state === 'inactive') return
    // L'assemblage et le téléchargement se font dans `onstop` : `stop()` vide d'abord ce que
    // l'encodeur retient encore, et cette dernière tranche arrive APRÈS ce retour.
    recorder.stop()
  }, [])

  const start = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || recorderRef.current) return
    const recorder = openRecording(canvas, filenameFor, audioTrack, (clip, filename) => {
      recorderRef.current = null
      setRecording(false)
      if (liveRef.current && clip) triggerDownload(clip, filename)
    })
    if (!recorder) return
    recorderRef.current = recorder
    setRecording(true)
    // FILMER UNE IMAGE FIGÉE N'A AUCUN SENS (décision 3) : un rejeu en pause repart.
    if (!playing) play()
  }, [canvasRef, filenameFor, audioTrack, playing, play])

  // AUTO-ARRÊT SUR LA RETOMBÉE DE LA LECTURE — pause manuelle ou fin du film, c'est le même
  // signal et la même sortie. On guette la TRANSITION, jamais l'état : au démarrage depuis une
  // pause, `playing` vaut encore `false` le temps d'un rendu, et lire l'état arrêterait le
  // clip dans la seconde qui suit son ouverture.
  const wasPlayingRef = useRef(playing)
  useEffect(() => {
    const was = wasPlayingRef.current
    wasPlayingRef.current = playing
    if (was && !playing) stop()
  }, [playing, stop])

  // Démontage : on referme l'enregistreur sans rien déposer (cf. `liveRef`).
  useEffect(() => {
    return () => {
      liveRef.current = false
      const recorder = recorderRef.current
      if (recorder && recorder.state !== 'inactive') recorder.stop()
    }
  }, [])

  const toggle = useCallback(() => {
    if (recorderRef.current) stop()
    else start()
  }, [start, stop])

  return { supported, recording, toggle }
}

export function useReplayCapture(o: ReplayCaptureOptions): ReplayCapture {
  const { canvasRef, doc, frameRef } = o
  // L'INSTANT SE LIT AU MOMENT DU NOMMAGE, jamais au rendu : image et vidéo partagent la même
  // règle de nom (décision 8), et c'est la seule chose qu'elles ont réellement en commun.
  const filenameFor = useCallback<FilenameFor>(
    (ext) => buildCaptureFilename(doc.matchId, frameToMs(frameRef.current, doc), ext, new Date()),
    [doc, frameRef],
  )
  const captureImage = useImageCapture(canvasRef, filenameFor)
  const video = useVideoRecording(o, filenameFor)
  const videoExport = useExportSeam(o)

  return {
    captureImage,
    recordingSupported: video.supported,
    recording: video.recording,
    toggleRecording: video.toggle,
    videoExport,
  }
}

/**
 * useExportSeam branche l'export hors temps réel — quand la page a donné de quoi peindre.
 *
 * LA PAUSE SE FABRIQUE ICI, elle n'est pas un paramètre de plus : `play` est le BASCULEUR de
 * lecture (`togglePlay`), donc l'appeler pendant une lecture met en pause. C'est exactement ce
 * dont l'export a besoin — la boucle d'animation écrit `frameRef` elle aussi, et les deux en
 * même temps se disputeraient le curseur.
 *
 * LE HOOK EST APPELÉ SANS CONDITION, et seul son RÉSULTAT est retenu ou jeté : appeler un hook
 * derrière un `if` est interdit par React, et une page sans surimpressions reste une page qui
 * peut exporter son terrain.
 */
function useExportSeam(o: ReplayCaptureOptions): ReplayExport | null {
  const { canvasRef, frameRef, doc, playing, play, redraw } = o
  const pause = useCallback(() => {
    if (playing) play()
  }, [playing, play])
  const bidon = useCallback(() => {}, [])
  const exportable = useReplayExport({
    canvasRef,
    frameRef,
    redraw: redraw ?? bidon,
    pause,
    doc: doc as ReplayDocumentReady & ReplayScoreDocument,
    playWindow: o.playWindow ?? null,
    scoreboard: o.scoreboard ?? [],
    xuidMeta: o.xuidMeta,
    outcome: o.outcome ?? null,
    titleSlug: doc.titleSlug ?? '',
    locale: o.locale ?? 'fr',
    soundTrack: o.soundTrack,
    soundVolume: o.soundVolume,
  })
  return redraw ? exportable : null
}
