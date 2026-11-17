/**
 * mapBackgroundDraw — LE DESSIN DU FOND DE CARTE DANS LA TOILE DU REJEU.
 *
 * EXTRAIT DE `ui/ReplayCanvas.tsx` (qui composait la pose et le `drawImage` en ligne) : la
 * géométrie pure reste dans `mapBackground.ts`, ce module la branche sur le cadrage partagé
 * (`CanvasView`) et sur un contexte de dessin. C'est aussi le seul endroit qui sait POSER
 * l'image dans la toile — le canvas ne fait que demander « où ? » puis « peins ».
 *
 * LA CARTE TOURNÉE (`CanvasView.orient`) : le rectangle de pose est l'emprise de l'image une fois
 * tournée, et l'image se dessine tournée d'autant autour du centre de ce rectangle (le contexte
 * est tourné le temps du `drawImage`, puis rendu). À 0 quart de tour le dessin est celui
 * d'avant, sans `save` ni rotation.
 *
 * La pose se lit sur `canvasView` (bornes de la fenêtre, taille de la toile, marge) : le fond
 * suit la fenêtre, pas la scène, sinon il resterait cadré large pendant que les joueurs zooment.
 */
import type { ReplayMapBackgroundCalibration } from '@/lib/api/types'
import { rotateCanvasAbout } from '@/lib/replay/orientation'
import { rognerAuPlan } from '@/lib/replay/planMask'

import { turnsOfView, type CanvasView } from '../model/replayView'
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
  return backgroundRect(background.calibration, view)
}

/**
 * drawMapBackground peint l'image ENTIÈRE sur son rectangle, tournée comme la carte du cadrage
 * (`view.orient`, quarts de tour horaires) ; la toile rogne le débord. Sans fond ou sans rectangle, il ne peint rien.
 *
 * Le rectangle est celui de l'image TOURNÉE : pour un quart ou trois quarts de tour, l'image est
 * dessinée avec les dimensions échangées, dans le repère tourné autour du centre du rectangle.
 */
export function drawMapBackground(
  ctx: CanvasRenderingContext2D,
  background: MapBackgroundSource | null,
  rect: BackgroundRect | null,
  view: Pick<CanvasView, 'orient'>,
): void {
  if (!background || !rect) return
  const turns = turnsOfView(view)
  if (turns === 0) {
    ctx.drawImage(background.image, rect.x, rect.y, rect.width, rect.height)
    return
  }
  const w = turns % 2 === 1 ? rect.height : rect.width
  const h = turns % 2 === 1 ? rect.width : rect.height
  ctx.save()
  rotateCanvasAbout(ctx, turns, rect.x + rect.width / 2, rect.y + rect.height / 2)
  ctx.drawImage(background.image, -w / 2, -h / 2, w, h)
  ctx.restore()
}

/**
 * clipToMapPlan retire de `ctx` ce qui sort de l'image plan posée pour ce cadrage (`rognerAuPlan`). C'est le
 * rognage commun des zones nommées et de la teinte de probabilité : une seule composition, le même rectangle
 * et la même rotation que le fond. Sans rectangle exploitable, rien n'est rogné (pas de fond plutôt qu'un fond
 * posé n'importe où).
 */
export function clipToMapPlan(ctx: CanvasRenderingContext2D, plan: MapBackgroundSource, view: CanvasView): void {
  const rect = placeMapBackground(plan, view)
  if (!rect) return
  rognerAuPlan(ctx, () => drawMapBackground(ctx, plan, rect, view))
}
