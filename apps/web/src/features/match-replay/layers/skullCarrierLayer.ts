/**
 * skullCarrierLayer.ts — LE PORTEUR DU CRÂNE d'Oddball (schéma 23), image par image.
 *
 * POURQUOI IL VIT À CÔTÉ DE `vipCrownLayer` ET `flagCarriesLayer`. Même partage : le marqueur suit
 * son porteur image par image (le porteur court, la période ne publie qu'un intervalle), donc il se
 * peint À CHAQUE IMAGE, sur la position RELUE du joueur — jamais à une position figée.
 *
 * CE QUE CE CALQUE DIT, ET RIEN DE PLUS : QUI porte le crâne à cette image. Un train de tics de
 * score de mode d'un même joueur EST une période de portage. UN SEUL crâne : au plus un porteur à
 * la fois (contrairement à la couronne VIP, dont chaque camp a la sienne).
 *
 * `closed` faux = RIEN ne date la fin du portage : le train de tics court jusqu'à la fin de l'axe
 * (le film s'arrête pendant le portage), une BORNE HAUTE. Le glyphe est alors ATTÉNUÉ — l'incertitude
 * est à l'écran, pas dans une note.
 *
 * LE CRÂNE LIBRE (`objectiveObjectsLayer`) DESSINE LE CRÂNE POSÉ AU SOL quand PERSONNE ne le porte ;
 * ce calque-ci le dessine SUR SON PORTEUR. Les deux ne sont jamais actifs au même instant pour le
 * même crâne : entre deux vies libres il y a un portage, et c'est exactement cet intervalle que ce
 * calque remplit. Le glyphe est le MÊME — celui, partagé, de `skullGlyph.ts` — pour qu'on
 * reconnaisse le même objet ; seule sa place change (au sol / au-dessus d'un joueur).
 *
 * SON HABILLAGE EST CELUI DU DRAPEAU depuis le 2026-08-28 (liseré à l'encre du fond, taille
 * agrandie, deux orbites) : le pourquoi, la cote et la limite (l'icône du jeu existe mais n'est
 * pas cuite dans le document) vivent dans l'en-tête de `skullGlyph.ts`. Aucun texte : le porteur
 * est déjà nommé par son marqueur.
 */
import { type XY } from '../model/replayLogic'
import { drawSkullGlyph } from './skullGlyph'

import { type CanvasView, projectTo } from '../model/replayView'
import type { ReplaySkullCarry } from '../model/replayNormalize'
import { carriedGlyphAlpha } from './carriedGlyphPulse'
import { covers } from '../model/replaySpans'

/** Style du calque : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface SkullCarrierStyle {
  /** Encre du disque du crâne (token du thème déjà résolu par l'appelant). */
  ink: string
  /** Encre du FOND : le liseré du crâne, comme celui du drapeau (`skullGlyph.ts`). */
  outline: string
  /** Mouvement réduit : la pulsation d'un portage « ouvert » devient une opacité constante. */
  reducedMotion: boolean
}

/**
 * SkullCarrierInput — ce que le calque reçoit de l'appelant (`useReplaySkullCarrier`).
 *
 * `posOf` est LA raison d'être de ce champ : le crâne se dessine sur le marqueur de son porteur à
 * l'image courante. Le calque ne sait pas relire une trajectoire — l'appelant lui passe la lecture.
 */
export interface SkullCarrierInput {
  style: SkullCarrierStyle
  /** Position monde du joueur à une image, ou `null` s'il n'est pas localisable. */
  posOf: (xuid: string, frame: number) => XY | null
}

/** Décalage vertical au-dessus du point du joueur (le marqueur occupe le point lui-même). */
const SKULL_OFFSET_Y = 12


/** Rayon de SURVOL potentiel, en pixels : exporté pour un futur survol, non consommé ici. */
export const SKULL_CARRIER_HIT_RADIUS = 10

/**
 * skullCarrierActiveAt rend les portages qui COUVRENT l'image demandée. UN SEUL crâne : au plus un,
 * mais on rend une liste (comme les calques frères) pour un tracé uniforme. Fonction PURE, testée
 * à part.
 */
export function skullCarrierActiveAt(
  carries: readonly ReplaySkullCarry[],
  frame: number,
): ReplaySkullCarry[] {
  const out: ReplaySkullCarry[] = []
  for (const c of carries) {
    if (covers(c, frame)) out.push(c)
  }
  return out
}

/** Opacité du glyphe pour un portage — la pulsation d'un portage « ouvert » comprise. */
/**
 * drawSkullCarrier peint, PAR-DESSUS les marqueurs, le crâne sur son porteur courant.
 *
 * Un porteur non localisable (vie non publiée, image hors de ses trajectoires) n'est PAS dessiné :
 * le crâne porté n'a pas de position propre, et l'inventer serait affirmer une place que le film ne
 * donne pas à cette image.
 */
export function drawSkullCarrier(
  ctx: CanvasRenderingContext2D,
  layer: SkullCarrierInput,
  carries: readonly ReplaySkullCarry[],
  view: CanvasView,
  frame: number,
): void {
  for (const c of skullCarrierActiveAt(carries, frame)) {
    const w = layer.posOf(c.xuid, frame)
    if (!w) continue
    const at = projectTo(view, w)
    // Le crâne se pose AU-DESSUS du marqueur (celui-ci occupe le point) : le décalage est appliqué
    // ICI, le glyphe partagé ne connaît que son centre.
    const center: XY = { x: at.x, y: at.y - SKULL_OFFSET_Y }
    drawSkullGlyph(ctx, center, {
      ink: layer.style.ink,
      outline: layer.style.outline,
      alpha: carriedGlyphAlpha(c.closed, frame, layer.style.reducedMotion),
    })
  }
  ctx.globalAlpha = 1
}
