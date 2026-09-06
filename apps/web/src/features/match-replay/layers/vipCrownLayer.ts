/**
 * vipCrownLayer.ts — LA COURONNE VIP (schéma 22), image par image.
 *
 * POURQUOI IL VIT À CÔTÉ DES AUTRES CALQUES D'ÉTAT. Même partage que `flagCarriesLayer.ts` : le
 * marqueur suit son porteur image par image (le VIP court, la période ne publie qu'un intervalle),
 * donc il se peint À CHAQUE IMAGE, sur la position RELUE du joueur — jamais à une position figée.
 *
 * CE QUE LA COURONNE DIT, ET RIEN DE PLUS : QUI porte la couronne VIP à cette image. Une période
 * s'ouvre à la sélection du VIP et se ferme à sa mort (ou à la sélection suivante). Deux camps ont
 * chacun leur VIP : deux couronnes peuvent donc être à l'écran en même temps, et c'est juste.
 *
 * `closed` faux = RIEN ne date la fin du port : l'intervalle court jusqu'à la fin de l'axe, une
 * BORNE HAUTE. La couronne est alors ATTÉNUÉE — l'incertitude est à l'écran, pas dans une note.
 *
 * AUCUNE VIGNETTE DU JEU : le film ne porte pas le bit VIP (la voie est le statborg), il n'y a
 * donc ni tag ni image à deviner. Le glyphe est tracé au canvas — une petite couronne — à l'encre
 * résolue par l'appelant (règle color-tokens). Aucun texte : la couronne se pose SUR le joueur
 * déjà nommé par son marqueur.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES (`posOf`), comme le drapeau : sans position, la
 * couronne ne se dessine pas — elle n'a pas de position propre (elle est TOUJOURS sur le joueur).
 */
import { type XY } from '../../../lib/replay/replayLogic'

import { type CanvasView, projectTo } from '../model/replayView'
import type { ReplayVipPeriod } from '../../../lib/replay/replayNormalize'
import { carriedGlyphAlpha } from './carriedGlyphPulse'
import { covers } from '../model/replaySpans'

/** Style du calque : l'encre de la couronne est RÉSOLUE par l'appelant (règle color-tokens). */
export interface VipCrownStyle {
  /** Encre de la couronne (token du thème déjà résolu par l'appelant). */
  ink: string
  /** Mouvement réduit : la pulsation de la couronne « ouverte » devient une opacité constante. */
  reducedMotion: boolean
}

/**
 * VipCrownInput — ce que le calque reçoit de l'appelant (`useReplayVipCrown`).
 *
 * `posOf` est LA raison d'être de ce champ : la couronne se dessine sur le marqueur du VIP à
 * l'image courante. Le calque ne sait pas relire une trajectoire — l'appelant lui passe la lecture.
 */
export interface VipCrownInput {
  style: VipCrownStyle
  /** Position monde du joueur à une image, ou `null` s'il n'est pas localisable. */
  posOf: (xuid: string, frame: number) => XY | null
}

// Réglages du glyphe. La couronne se pose AU-DESSUS du marqueur du joueur, plus petite que lui :
// elle le qualifie, elle ne le remplace pas.
const CROWN_W = 12
const CROWN_H = 8
/** Décalage vertical au-dessus du point du joueur (le marqueur occupe le point lui-même). */
const CROWN_OFFSET_Y = 12
const CROWN_STROKE_WIDTH = 1.4
/** Rayon de SURVOL potentiel, en pixels : exporté pour un futur survol, non consommé ici. */
export const VIP_CROWN_HIT_RADIUS = 10


/**
 * vipActiveAt rend les périodes qui COUVRENT l'image demandée — au plus une par VIP courant, mais
 * plusieurs camps peuvent avoir chacun le leur. Fonction PURE, testée à part.
 */
export function vipActiveAt(
  periods: readonly ReplayVipPeriod[],
  frame: number,
): ReplayVipPeriod[] {
  const out: ReplayVipPeriod[] = []
  for (const p of periods) {
    if (covers(p, frame)) out.push(p)
  }
  return out
}

/**
 * drawVipCrown peint, PAR-DESSUS les marqueurs, la couronne sur le VIP courant de chaque camp.
 *
 * Un VIP non localisable (vie non publiée, image hors de ses trajectoires) n'est PAS dessiné : la
 * couronne n'a pas de position propre, et l'inventer serait affirmer une place que le film ne
 * donne pas à cette image.
 */
export function drawVipCrown(
  ctx: CanvasRenderingContext2D,
  layer: VipCrownInput,
  periods: readonly ReplayVipPeriod[],
  view: CanvasView,
  frame: number,
): void {
  for (const p of vipActiveAt(periods, frame)) {
    const w = layer.posOf(p.xuid, frame)
    if (!w) continue
    const at = projectTo(view, w)
    drawCrownGlyph(ctx, at, {
      ink: layer.style.ink,
      alpha: carriedGlyphAlpha(p.closed, frame, layer.style.reducedMotion),
    })
  }
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'un glyphe a besoin de savoir. */
interface CrownGlyphPaint {
  ink: string
  alpha: number
}

/**
 * drawCrownGlyph trace une petite COURONNE au-dessus du point servi : une base et trois pointes,
 * la centrale plus haute. Tracé au canvas, sans image (cf. l'en-tête du fichier).
 */
export function drawCrownGlyph(
  ctx: CanvasRenderingContext2D,
  at: XY,
  paint: CrownGlyphPaint,
): void {
  const cx = at.x
  const baseY = at.y - CROWN_OFFSET_Y
  const topY = baseY - CROWN_H
  const midY = baseY - CROWN_H * 0.45
  const hw = CROWN_W / 2
  ctx.globalAlpha = paint.alpha
  ctx.fillStyle = paint.ink
  ctx.strokeStyle = paint.ink
  ctx.lineWidth = CROWN_STROKE_WIDTH
  ctx.lineJoin = 'round'
  ctx.beginPath()
  // Bord bas gauche -> pointe gauche -> creux -> pointe centrale -> creux -> pointe droite -> bas droit.
  ctx.moveTo(cx - hw, baseY)
  ctx.lineTo(cx - hw, midY)
  ctx.lineTo(cx - hw * 0.45, baseY - CROWN_H * 0.7)
  ctx.lineTo(cx, topY)
  ctx.lineTo(cx + hw * 0.45, baseY - CROWN_H * 0.7)
  ctx.lineTo(cx + hw, midY)
  ctx.lineTo(cx + hw, baseY)
  ctx.closePath()
  ctx.fill()
}
