/**
 * useReplayCapture — LA COUTURE REACT DE LA CAPTURE : ce que la barre de lecture commande, et
 * ce que le canvas doit lui prêter pour cela.
 *
 * POURQUOI UN HOOK PLUTÔT QU'UN HANDLER DANS LE CANVAS. `ReplayCanvas.tsx` vit sous un cliquet
 * de taille (`placementFamily.guard.test.ts`, neuf extractions imposées à ce jour) : tout ce
 * qui n'est pas du DESSIN en sort. Le canvas ne prête donc que sa TOILE et son HORLOGE, et
 * reçoit en retour un objet unique qu'il repasse tel quel à la barre — exactement le patron du
 * son (`useReplaySound` -> `ReplaySound`), et pour la même raison : brancher une commande de
 * plus ne doit pas coûter une ligne de canvas.
 *
 * CE QUE CE HOOK NE DÉCIDE PAS : le nom du fichier, le téléchargement et la lecture des pixels
 * sont de la logique pure et vivent dans `replayCapture.ts`. Ici, seule la couture — lire les
 * refs au moment du clic, et jamais pendant le rendu.
 */
import { useCallback, type RefObject } from 'react'

import { buildCaptureFilename, captureCanvasImage, triggerDownload } from './replayCapture'
import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

export interface ReplayCaptureOptions {
  /** La toile du rejeu : c'est elle, et elle seule, qui porte tout ce qui se voit. */
  canvasRef: RefObject<HTMLCanvasElement | null>
  /** Le document : il donne l'identifiant du match et la durée d'une image. */
  doc: ReplayDocumentReady
  /** L'image courante, partagée avec le dessin — lue au clic, jamais au rendu. */
  frameRef: RefObject<number>
}

/** Ce que la barre de lecture reçoit (même forme que `ReplaySound` : un objet, pas des props). */
export interface ReplayCapture {
  /** Télécharge la scène courante en PNG. Sans toile lisible : ne fait rien, ne casse rien. */
  captureImage: () => void
}

export function useReplayCapture({ canvasRef, doc, frameRef }: ReplayCaptureOptions): ReplayCapture {
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

  return { captureImage }
}
