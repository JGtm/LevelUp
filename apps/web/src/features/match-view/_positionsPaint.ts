/**
 * _positionsPaint.ts — LE TRACÉ DE « OCCUPATION DU TERRAIN » : le fond de carte, le plan des murs et la
 * chaleur, sur la fenêtre visible, la carte tournée comme dans le rejeu 2D.
 *
 * EXTRAIT DE `MatchPositionsHeatmap.tsx` : le composant gardait le tracé dans un effet, donc hors de
 * portée d'un test sans navigateur. Ce module est le tracé pur, sur un contexte 2D donné.
 *
 * LA CARTE TOURNÉE, SANS TOUCHER AU TRACÉ D'ORIGINE. La fenêtre visible (`bounds`) est dans le repère de
 * la carte TOURNÉE (celui du zoom et des gestes). On défait la rotation de cette fenêtre — c'est un
 * rectangle du monde, de largeur et hauteur échangées pour un quart de tour —, on trace exactement
 * comme avant dans une toile « virtuelle » de ce rectangle, puis on pose cette toile sur la vraie, tournée
 * de `turns` quarts de tour horaires autour du centre : `translate(centre)`, `rotate`, `translate(-centre
 * virtuel)`. Le fond, le plan et la chaleur — des objets du monde — tournent ensemble, sans calcul de
 * pixel de plus. À 0 quart de tour, aucune transformation n'est posée.
 */
import type { ReplayBounds } from '@/lib/api/types'
import { flouDeLaChaleur, peindreLaChaleurLissee } from '@/lib/replay/chaleurLissee'
import type { TacticalGrid } from '@/lib/replay/heatPaint'
import { normalizeTurns, orientBoundsAbout, type QuarterTurns } from '@/lib/replay/orientation'
import type { XY } from '@/lib/replay/replayLogic'
import type { WallPlanReady } from '@/lib/replay/wallPlan'
import { strokeWallPlan } from '@/lib/replay/wallPlanPaint'

import type { MapFrame } from './_positionsHeat'

/** La fenêtre visible dans le repère du MONDE : la rotation de la carte défaite autour de `pivot`. */
export function fenetreDuMonde(bounds: ReplayBounds, turns: QuarterTurns, pivot: XY): ReplayBounds {
  return orientBoundsAbout(bounds, normalizeTurns(4 - turns), pivot)
}

/** Ce que le tracé a besoin de savoir, hors contexte et taille. */
export interface PeintureDesPositions {
  frame: MapFrame
  /** L'image du fond (dimensions naturelles lues pour passer des mètres aux pixels de l'image). */
  image: CanvasImageSource & { width: number }
  /** La grille LISSÉE à peindre, ou `null` (le fond se trace alors seul). */
  lissee: TacticalGrid | null
  /** Le côté d'une cellule de la grille, en mètres. */
  cell: number
  ramp: readonly string[]
  /** La fenêtre visible, dans le repère de la carte tournée. */
  bounds: ReplayBounds
  /** Les quarts de tour horaires de la carte et son pivot (le centre du cadre du fond, dans le monde). */
  turns: QuarterTurns
  pivot: XY
  /** Le plan des murs, ou `null` (carte sans plan, interrupteur éteint). */
  plan: WallPlanReady | null
  /** L'encre du plan des murs, déjà résolue. */
  ink: string
}

/** peindreLesPositions trace le fond, le plan des murs puis la chaleur sur une toile de `width` x `height` px. */
export function peindreLesPositions(
  ctx: CanvasRenderingContext2D,
  width: number,
  height: number,
  p: PeintureDesPositions,
): void {
  const { frame, image, lissee, bounds: fenetre } = p
  const bounds = fenetreDuMonde(fenetre, p.turns, p.pivot)
  // La toile virtuelle : celle de la fenêtre du monde, de dimensions échangées pour un quart de tour.
  const impair = p.turns % 2 === 1
  const w0 = impair ? height : width
  const h0 = impair ? width : height
  if (p.turns !== 0) {
    ctx.save()
    ctx.translate(width / 2, height / 2)
    ctx.rotate((p.turns * Math.PI) / 2)
    ctx.translate(-w0 / 2, -h0 / 2)
  }
  // LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE, ET C'EST LA FENÊTRE VISIBLE. À 1x elle vaut la scène
  // et l'on retrouve exactement le tracé d'avant le zoom ; au-delà, l'image est découpée à la source (la
  // portion monde visible, convertie en pixels d'image) et le calque reçoit la MÊME origine et la MÊME
  // échelle. Les deux ne peuvent donc pas se désaligner.
  const k = w0 / Math.max(bounds.maxX - bounds.minX, 1e-6)
  const pxParM = image.width / Math.max(frame.widthM, 1e-6)
  ctx.drawImage(
    image,
    (bounds.minX - frame.originX) * pxParM,
    (frame.originY - bounds.maxY) * pxParM,
    (bounds.maxX - bounds.minX) * pxParM,
    (bounds.maxY - bounds.minY) * pxParM,
    0,
    0,
    w0,
    h0,
  )
  // LE PLAN DES MURS, entre le fond et la chaleur : un trait de dessin sur la carte, sous la mesure.
  strokeWallPlan(ctx, p.plan, (q) => ({ x: (q.x - bounds.minX) * k, y: (bounds.maxY - q.y) * k }), p.ink)
  if (lissee) {
    peindreLaChaleurLissee(
      ctx,
      lissee,
      {
        topLeftWorld: {
          x: (frame.originX - bounds.minX) * k,
          y: (bounds.maxY - frame.originY) * k,
        },
        scale: k,
      },
      p.ramp,
      flouDeLaChaleur(p.cell, k),
    )
  }
  if (p.turns !== 0) ctx.restore()
}
