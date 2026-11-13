/**
 * mapBackgroundDraw — LE DESSIN DU FOND DE CARTE DANS LA TOILE DU REJEU.
 *
 * EXTRAIT DE `ui/ReplayCanvas.tsx` (qui composait la pose et le `drawImage` en ligne) : la
 * géométrie pure reste dans `mapBackground.ts`, ce module la branche sur le cadrage partagé
 * (`CanvasView`) et sur un contexte de dessin. C'est aussi le seul endroit qui sait POSER
 * l'image dans la toile — le canvas ne fait que demander « où ? » puis « peins ».
 *
 * La pose se lit sur `canvasView` (bornes de la fenêtre, taille de la toile, marge) : le fond
 * suit la fenêtre, pas la scène, sinon il resterait cadré large pendant que les joueurs zooment.
 */
import type { ReplayMapBackgroundCalibration } from '@/lib/api/types'

import type { CanvasView } from '../model/replayView'
import { backgroundRect, type BackgroundRect } from './mapBackground'

/** Le fond retenu : l'image cuite et le calage qui la pose dans le repère monde. */
export interface MapBackgroundSource {
  calibration: ReplayMapBackgroundCalibration
  image: CanvasImageSource
}

/**
 * placeMapBackground rend le rectangle de la toile que l'image occupe pour ce cadrage, ou null
 * sans fond ou quand le calage est inexploitable (pas de fond plutôt qu'un fond posé n'importe où).
 */
export function placeMapBackground(
  background: Pick<MapBackgroundSource, 'calibration'> | null,
  view: CanvasView,
): BackgroundRect | null {
  if (!background) return null
  return backgroundRect(background.calibration, view.bounds, view.width, view.height, view.pad)
}

/**
 * drawMapBackground peint l'image ENTIÈRE sur son rectangle ; la toile rogne le débord. Sans
 * fond ou sans rectangle, il ne peint rien.
 */
export function drawMapBackground(
  ctx: CanvasRenderingContext2D,
  background: MapBackgroundSource | null,
  rect: BackgroundRect | null,
): void {
  if (!background || !rect) return
  ctx.drawImage(background.image, rect.x, rect.y, rect.width, rect.height)
}
