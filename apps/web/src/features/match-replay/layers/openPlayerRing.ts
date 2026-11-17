/**
 * openPlayerRing.ts — LE PION CERCLÉ DU JOUEUR DONT LA FICHE EST OUVERTE, peint sur le plan.
 *
 * Un anneau clair, à `COTES_PION_CERCLE.rayonPx` du centre du pion, cerné de part et d'autre d'un liseré à
 * l'encre du fond (comme le liseré des glyphes portés) : le repère se lit sur un fond de carte clair comme
 * sombre. Il se lit sur la position du joueur À L'IMAGE (le pion avance, l'anneau le suit) et ne se peint que
 * pour un joueur VIVANT à cette image : aucune vie de lui qui couvre l'image, pas d'anneau — pas de position
 * inventée, pas d'anneau sur une croix de mort. Embarqué, l'anneau suit le véhicule qui le porte
 * (`positionOfCarrierAt`, la règle des objets portés), puisque le pion est alors retiré de la carte.
 *
 * Les encres sont RÉSOLUES par l'appelant (règle color-tokens) ; ce module ne lit ni thème ni DOM.
 */
import { isAliveAt, positionAt, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { buildEmbarkedPosAt, positionOfCarrierAt } from '../model/carrierPosition'
import { COTES_PION_CERCLE } from '../model/cotesFenetreFiche'
import { projectTo, type CanvasView } from '../model/replayView'

/** Ce que le peintre lit : la position du joueur à une image (null = pas vivant), et les deux encres. */
export interface OpenPlayerRing {
  posAt: (frame: number) => XY | null
  /** L'encre de l'anneau : la plus claire du thème. */
  ink: string
  /** L'encre du liseré : celle du fond. */
  outline: string
}

/**
 * openPlayerPosAt rend la relecture de position du joueur : vivant à l'image -> sa position (celle du
 * véhicule s'il est embarqué), sinon null. L'index des épisodes d'occupation se bâtit UNE fois.
 */
export function openPlayerPosAt(doc: ReplayDocumentReady, player: ReplayPlayer): (frame: number) => XY | null {
  const embarkedAt = buildEmbarkedPosAt(doc.vehicles)
  const bipedAt = (_xuid: string, frame: number): XY | null => {
    const life = player.lives.find((l) => isAliveAt(l, frame))
    return life ? positionAt(life.points, frame) : null
  }
  return (frame) => {
    if (!player.lives.some((l) => isAliveAt(l, frame))) return null
    return positionOfCarrierAt(embarkedAt, bipedAt, player.xuid, frame)
  }
}

/**
 * drawOpenPlayerRing peint l'anneau sur le pion du joueur ; ne fait rien s'il n'est pas vivant à l'image.
 * `k` est l'échelle de la toile : les cotes sont des pixels d'écran.
 */
export function drawOpenPlayerRing(
  ctx: CanvasRenderingContext2D,
  ring: OpenPlayerRing,
  view: CanvasView,
  frame: number,
  k: number,
): void {
  const world = ring.posAt(frame)
  if (!world) return
  const c = projectTo(view, world)
  const { rayonPx, anneauPx, lisereExtraPx } = COTES_PION_CERCLE
  ctx.globalAlpha = 1
  ctx.lineWidth = (anneauPx + 2 * lisereExtraPx) * k
  ctx.strokeStyle = ring.outline
  ctx.beginPath()
  ctx.arc(c.x, c.y, rayonPx * k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.lineWidth = anneauPx * k
  ctx.strokeStyle = ring.ink
  ctx.beginPath()
  ctx.arc(c.x, c.y, rayonPx * k, 0, Math.PI * 2)
  ctx.stroke()
}
