/**
 * captureBusy — LA RÈGLE QUI NEUTRALISE LA LECTURE PENDANT UN EXPORT, écrite une fois pour les barres de lecture
 * et le bouton d'export : l'export de CETTE capture travaille-t-il ? Faux quand le navigateur ne sait pas encoder
 * (pas d'export). Un garde-rail (`ui/exportBusy.guard.test.ts`) interdit de la réécrire à la main.
 */
import { isExportBusy } from './ReplayExportDialog'
import type { ReplayCapture } from './useReplayCapture'

export function isCaptureBusy(capture: Pick<ReplayCapture, 'videoExport'>): boolean {
  return capture.videoExport ? isExportBusy(capture.videoExport.state) : false
}
