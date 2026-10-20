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
 * L'ENCRE EST CELLE DU CAMP DU VIP, vu du joueur regardé (allié / adverse), comme le drapeau :
 * avec deux VIP à l'écran, la couleur dit lequel est à protéger et lequel est à abattre. Sans
 * allégeance connue, le neutre du thème — jamais un camp supposé. L'encre est RÉSOLUE par
 * l'appelant (règle color-tokens). Le glyphe pose un LISERÉ à l'encre du fond, à la même cote que
 * le drapeau et le crâne, pour garder un bord franc sur un fond de carte chargé.
 *
 * AUCUNE VIGNETTE DU JEU : le film ne porte pas le bit VIP (la voie est le statborg), il n'y a
 * donc ni tag ni image à deviner. La forme est décrite dans `crownGlyph.ts`, partagée avec le
 * filigrane de la fiche. Aucun texte : la couronne se pose SUR le joueur déjà nommé par son
 * marqueur, AU-DESSUS de lui — elle le qualifie, elle ne le remplace pas.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES (`posOf`), comme le drapeau : sans position, la
 * couronne ne se dessine pas — elle n'a pas de position propre (elle est TOUJOURS sur le joueur).
 */
import { type XY } from '../../../lib/replay/replayLogic'

import { type CanvasView, projectTo } from '../model/replayView'
import type { ReplayVipPeriod } from '../../../lib/replay/replayNormalize'
import { carriedGlyphAlpha } from './carriedGlyphPulse'
import { covers } from '../model/replaySpans'
import { CROWN_BOTTOM, CROWN_BOX, traceCrown } from './crownGlyph'

/** Style du calque : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface VipCrownStyle {
  /** Encre de la couronne d'un VIP : la couleur de son camp vu du joueur regardé, ou le neutre. */
  inkOf: (xuid: string) => string
  /** Encre du FOND, pour le liseré (celle du drapeau et du crâne). */
  outline: string
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

/** Côté du glyphe, en pixels canvas : la largeur du crâne porté (r=7 + liseré). */
export const CROWN_SIZE_PX = 18
/**
 * Hauteur du bas de la couronne au-dessus du point du joueur : le bas du crâne porté (centre à
 * 12, rayon 7), pour que les objets portés tiennent la même place au-dessus du marqueur.
 */
const CROWN_FOOT_Y = 5
/** Débord du liseré au-delà de la forme — MÊME COTE que le drapeau (`FLAG_OUTLINE_PAD`). */
const CROWN_LISERE_PAD = 1.6

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
    drawCrownGlyph(ctx, projectTo(view, w), {
      ink: layer.style.inkOf(p.xuid),
      outline: layer.style.outline,
      alpha: carriedGlyphAlpha(p.closed, frame, layer.style.reducedMotion),
    })
  }
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'un glyphe a besoin de savoir. */
interface CrownGlyphPaint {
  ink: string
  outline: string
  alpha: number
}

/**
 * drawCrownGlyph trace la couronne AU-DESSUS du point servi : le liseré à l'encre du fond d'abord
 * (trait centré, dont le remplissage recouvre la moitié intérieure), puis la forme à l'encre du camp.
 */
export function drawCrownGlyph(
  ctx: CanvasRenderingContext2D,
  at: XY,
  paint: CrownGlyphPaint,
): void {
  const scale = CROWN_SIZE_PX / CROWN_BOX
  const origin: XY = {
    x: at.x - CROWN_SIZE_PX / 2,
    y: at.y - CROWN_FOOT_Y - CROWN_BOTTOM * scale,
  }
  ctx.globalAlpha = paint.alpha
  traceCrown(ctx, origin, scale)
  ctx.lineJoin = 'round'
  ctx.lineWidth = 2 * CROWN_LISERE_PAD
  ctx.strokeStyle = paint.outline
  ctx.stroke()
  ctx.fillStyle = paint.ink
  ctx.fill()
}
