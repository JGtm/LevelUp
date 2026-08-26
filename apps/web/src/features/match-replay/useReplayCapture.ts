/**
 * useReplayCapture — LA COUTURE REACT DE LA CAPTURE : ce que la barre de lecture commande, et
 * ce que le canvas doit lui prêter pour cela.
 *
 * POURQUOI UN HOOK PLUTÔT QU'UN HANDLER DANS LE CANVAS. `ReplayCanvas.tsx` vit sous un cliquet
 * de taille (`placementFamily.guard.test.ts`, neuf extractions imposées à ce jour) : tout ce
 * qui n'est pas du DESSIN en sort. Le canvas ne prête donc que sa TOILE, son HORLOGE et son
 * état de lecture, et reçoit en retour un objet unique qu'il repasse tel quel à la barre —
 * exactement le patron du son (`useReplaySound` -> `ReplaySound`), et pour la même raison :
 * brancher une commande de plus ne doit pas coûter une ligne de canvas.
 *
 * CE QUE CE HOOK NE DÉCIDE PAS : le nom du fichier, le téléchargement et la lecture des pixels
 * sont de la logique pure et vivent dans `replayCapture.ts` ; le choix du conteneur vidéo, dans
 * `replayRecording.ts` ; la piste audio, dans `replayAudio.ts` (le lecteur ouvre une seconde
 * sortie en parallèle des haut-parleurs). Ici, seule la couture — lire les refs au moment du
 * clic, tenir l'état de l'enregistreur, et refermer proprement.
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
import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

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
}

export function useReplayCapture(o: ReplayCaptureOptions): ReplayCapture {
  const { canvasRef, doc, frameRef, playing, play, audioTrack } = o
  const [recording, setRecording] = useState(false)
  const recorderRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])
  // Le nom est FIGÉ au démarrage : il porte l'instant où le clip commence, pas celui où il
  // s'arrête. Un clip nommé sur sa fin ne se replacerait pas dans le match.
  const filenameRef = useRef('')
  // LE DÉMONTAGE NE TÉLÉCHARGE PAS. Quitter la page pendant un enregistrement doit refermer
  // l'enregistreur, pas déposer un fichier que plus personne n'attend.
  const liveRef = useRef(true)

  // La capacité ne dépend que du navigateur : elle ne change pas d'un rendu à l'autre.
  const recordingSupported = useMemo(() => canRecordCanvas(), [])

  const captureImage = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    // LE NOM EST FIGÉ AVANT L'ENCODAGE : `toBlob` est asynchrone, et l'image continue de
    // courir pendant ce temps. Nommer après rendrait un fichier daté d'un instant qui n'est
    // pas celui qu'il montre.
    const filename = buildCaptureFilename(
      doc.matchId,
      frameToMs(frameRef.current, doc),
      'png',
      new Date(),
    )
    void captureCanvasImage(canvas).then((blob) => {
      // Pas de blob = toile vierge ou encodage refusé : rien à remettre. Un fichier vide
      // serait pire que pas de fichier (cf. `captureCanvasImage`).
      if (blob) triggerDownload(blob, filename)
    })
  }, [canvasRef, doc, frameRef])

  const stopRecording = useCallback(() => {
    const recorder = recorderRef.current
    if (!recorder || recorder.state === 'inactive') return
    // L'assemblage et le téléchargement se font dans `onstop` : `stop()` vide d'abord ce que
    // l'encodeur retient encore, et cette dernière tranche arrive APRÈS ce retour.
    recorder.stop()
  }, [])

  const startRecording = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || recorderRef.current) return
    const choice = pickVideoMimeType(isVideoTypeSupported)
    if (!choice) return
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
    chunksRef.current = []
    filenameRef.current = buildCaptureFilename(
      doc.matchId,
      frameToMs(frameRef.current, doc),
      choice.ext,
      new Date(),
    )
    recorder.ondataavailable = (e: BlobEvent) => {
      if (e.data && e.data.size > 0) chunksRef.current.push(e.data)
    }
    recorder.onstop = () => {
      const parts = chunksRef.current
      chunksRef.current = []
      recorderRef.current = null
      // LES PISTES DE LA TOILE SE COUPENT ICI, pas avant : la capture doit tourner jusqu'à la
      // dernière tranche, sans quoi le clip perdrait sa fin. LA PISTE AUDIO, elle, N'EST PAS
      // COUPÉE — elle appartient au lecteur de son, qui vit plus longtemps que le clip ; la
      // fermer ici rendrait muet tout enregistrement suivant.
      for (const track of canvasStream.getTracks()) track.stop()
      setRecording(false)
      // Rien d'enregistré (arrêt immédiat, encodeur muet) : pas de fichier vide.
      if (!liveRef.current || parts.length === 0) return
      triggerDownload(new Blob(parts, { type: choice.mime }), filenameRef.current)
    }
    recorderRef.current = recorder
    recorder.start()
    setRecording(true)
    // FILMER UNE IMAGE FIGÉE N'A AUCUN SENS (décision 3) : un rejeu en pause repart.
    if (!playing) play()
  }, [canvasRef, doc, frameRef, playing, play, audioTrack])

  const toggleRecording = useCallback(() => {
    if (recorderRef.current) stopRecording()
    else startRecording()
  }, [startRecording, stopRecording])

  // AUTO-ARRÊT SUR LA RETOMBÉE DE LA LECTURE — pause manuelle ou fin du film, c'est le même
  // signal et la même sortie. On guette la TRANSITION, jamais l'état : au démarrage depuis une
  // pause, `playing` vaut encore `false` le temps d'un rendu, et lire l'état arrêterait le
  // clip dans la seconde qui suit son ouverture.
  const wasPlayingRef = useRef(playing)
  useEffect(() => {
    const was = wasPlayingRef.current
    wasPlayingRef.current = playing
    if (was && !playing) stopRecording()
  }, [playing, stopRecording])

  // Démontage : on referme l'enregistreur sans rien déposer (cf. `liveRef`).
  useEffect(() => {
    return () => {
      liveRef.current = false
      const recorder = recorderRef.current
      if (recorder && recorder.state !== 'inactive') recorder.stop()
    }
  }, [])

  return { captureImage, recordingSupported, recording, toggleRecording }
}
