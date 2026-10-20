/**
 * mapBackground.ts — LE FOND DE CARTE DU REJEU, côté rendu.
 *
 * L'image d'une carte est cuite hors ligne (cmd/mapfond-build) et voyage avec son CALAGE :
 * mètres par pixel, et l'origine monde du coin HAUT-GAUCHE. La convention est publiée en
 * clair dans le sidecar, et c'est celle de `MapBackgroundCalibration.MondeVersPixel` côté
 * Go — elle est réécrite ici parce que le dessin se fait dans le navigateur, et à un seul
 * endroit pour que la seconde écriture ne devienne pas une seconde vérité :
 *
 *   xMonde = originX + (px + 0.5) * metersPerPixel
 *   yMonde = originY - (py + 0.5) * metersPerPixel
 *
 * L'IMAGE DESCEND EN Y QUAND LE MONDE MONTE — même sens que le canvas, ce qui rend la pose
 * du fond purement affine : deux coins suffisent, aucune rotation, aucun miroir.
 *
 * Pas de React, pas de canvas : géométrie pure, testable (mapBackground.test.ts).
 */
import type { ReplayBounds, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import { worldToCanvas } from '../../../lib/replay/replayLogic'

/** Rectangle de dessin dans le repère canvas (pixels CSS). */
export interface BackgroundRect {
  x: number
  y: number
  width: number
  height: number
}

/** L'étendue MONDE couverte par l'image, déduite du calage. */
export interface BackgroundExtent {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

/**
 * worldExtent rend l'emprise monde de l'image. `originX`/`originY` sont les coordonnées du
 * coin haut-gauche : l'image s'étend vers l'EST et vers le SUD.
 */
export function worldExtent(cal: ReplayMapBackgroundCalibration): BackgroundExtent {
  const w = cal.widthPx * cal.metersPerPixel
  const h = cal.heightPx * cal.metersPerPixel
  return {
    minX: cal.originX,
    maxX: cal.originX + w,
    maxY: cal.originY,
    minY: cal.originY - h,
  }
}

/**
 * worldToPixel applique la convention du sidecar : position monde -> pixel de l'image.
 * `inside` dit si le point tombe dans l'image — un point hors cadre garde ses coordonnées
 * (elles servent à mesurer le débord), il n'est pas ramené au bord.
 */
export function worldToPixel(
  cal: ReplayMapBackgroundCalibration,
  x: number,
  y: number,
): { px: number; py: number; inside: boolean } {
  if (cal.metersPerPixel <= 0) return { px: 0, py: 0, inside: false }
  const px = Math.floor((x - cal.originX) / cal.metersPerPixel)
  const py = Math.floor((cal.originY - y) / cal.metersPerPixel)
  const inside = px >= 0 && px < cal.widthPx && py >= 0 && py < cal.heightPx
  return { px, py, inside }
}

/**
 * backgroundRect place l'image entière dans le canvas, pour le MÊME cadrage que les
 * trajectoires (`worldToCanvas`).
 *
 * POURQUOI L'IMAGE ENTIÈRE ET PAS UN DÉCOUPAGE. La projection est affine et sans rotation :
 * poser les deux coins opposés suffit, et le canvas se charge de rogner ce qui déborde. Un
 * découpage en pixels source ajouterait une seconde arithmétique — donc un second endroit
 * où se tromper — sans rien gagner à l'écran.
 *
 * Rend null si le calage est inexploitable (m/px nul, image vide) : pas de fond plutôt qu'un
 * fond posé n'importe où.
 */
export function backgroundRect(
  cal: ReplayMapBackgroundCalibration,
  bounds: ReplayBounds,
  width: number,
  height: number,
  pad: number,
): BackgroundRect | null {
  if (cal.metersPerPixel <= 0 || cal.widthPx <= 0 || cal.heightPx <= 0) return null
  const ext = worldExtent(cal)
  // Coin haut-gauche de l'image = (minX monde, maxY monde) ; coin bas-droit = (maxX, minY).
  const tl = worldToCanvas({ x: ext.minX, y: ext.maxY }, bounds, width, height, pad)
  const br = worldToCanvas({ x: ext.maxX, y: ext.minY }, bounds, width, height, pad)
  const w = br.x - tl.x
  const h = br.y - tl.y
  if (!(w > 0) || !(h > 0)) return null
  return { x: tl.x, y: tl.y, width: w, height: h }
}

/**
 * coversPlayedArea dit si l'image contient la zone parcourue par les joueurs.
 *
 * CE QUE ÇA SERT À DÉTECTER : un fond qui ne recouvre pas le terrain joué n'est pas un
 * détail d'affichage, c'est le signe que les deux repères ne sont pas le même (mauvaise
 * carte, ou bornes de déquantification d'une autre carte). Mieux vaut alors NE PAS poser le
 * fond que d'afficher des joueurs à côté de leur terrain.
 */
export function coversPlayedArea(
  cal: ReplayMapBackgroundCalibration,
  bounds: ReplayBounds,
): boolean {
  if (cal.metersPerPixel <= 0) return false
  const ext = worldExtent(cal)
  return (
    bounds.minX >= ext.minX &&
    bounds.maxX <= ext.maxX &&
    bounds.minY >= ext.minY &&
    bounds.maxY <= ext.maxY
  )
}
