/**
 * bombCarrierLayer.ts — LA BOMBE D'ASSAUT (schéma 30), image par image : sur son PORTEUR
 * pendant les périodes de portage, AU SOL entre un lâcher et la prise suivante.
 *
 * POURQUOI IL VIT À CÔTÉ DE `skullCarrierLayer`. Même partage : le marqueur suit son porteur
 * image par image (le porteur court, la période ne publie qu'un intervalle), donc il se peint À
 * CHAQUE IMAGE, sur la position RELUE du joueur — jamais à une position figée.
 *
 * LA BOMBE AU SOL EST DÉRIVÉE, PAS PUBLIÉE, et c'est écrit des deux côtés (cf.
 * document_bomb_carries.go) : l'objet bombe n'a pas de canal mesuré côté Go. Entre la FIN d'une
 * période FERMÉE et la prise suivante, la bombe est immobile au DERNIER POINT de son lâcheur —
 * la règle du drapeau `dropped`, la position venant ici de la piste déjà publiée du lâcheur à
 * l'instant du lâcher. Le segment se coupe au premier de : la prise suivante, l'EXPLOSION
 * (`bomb_detonations` du calque des actions — après elle, la bombe n'existe plus), la fin de
 * l'axe. SANS HORLOGE DE CONFIANCE (`filmClockTrusted` faux), les explosions ne se posent pas
 * sur la grille : le sol ne se dessine PAS (l'appelant passe `explosions: null`) — une bombe
 * posée qui survivrait à sa propre explosion affirmerait ce que le film ne dit pas ; muet vaut
 * mieux que faux, la règle de toute la chaîne.
 *
 * `closed` faux = RIEN ne date la fin du portage (le film s'arrête pendant) : le glyphe porté
 * PULSE, comme le crâne — l'incertitude est à l'écran, pas dans une note. Un portage ouvert ne
 * produit JAMAIS de bombe au sol : personne ne l'a lâchée.
 *
 * Le glyphe est UNIQUE (`bombGlyph.ts`) pour les deux états — on reconnaît le même objet ;
 * seule sa place change, exactement comme le crâne libre et le crâne porté.
 */
import { drawBombGlyph } from './bombGlyph'
import { type XY } from '../../../lib/replay/replayLogic'

import { type CanvasView, projectTo } from '../model/replayView'
import type { ReplayBombCarry } from '../../../lib/replay/replayNormalize'
import { carriedGlyphAlpha } from './carriedGlyphPulse'
import { covers } from '../model/replaySpans'

/** Style du calque : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface BombCarrierStyle {
  /** Encre de la bombe (token du thème déjà résolu par l'appelant). */
  ink: string
  /** Encre du FOND : le liseré, comme celui du crâne et du drapeau. */
  outline: string
  /** Mouvement réduit : la pulsation d'un portage « ouvert » devient une opacité constante. */
  reducedMotion: boolean
}

/**
 * BombCarrierInput — ce que le calque reçoit de l'appelant (`useReplayBombCarrier`).
 *
 * `posOf` est LA raison d'être du champ : la bombe se dessine sur le marqueur de son porteur à
 * l'image courante, et AU SOL sur le dernier point de son lâcheur. Le calque ne sait pas relire
 * une trajectoire — l'appelant lui passe la lecture.
 */
export interface BombCarrierInput {
  style: BombCarrierStyle
  /** Position monde du joueur à une image, ou `null` s'il n'est pas localisable. */
  posOf: (xuid: string, frame: number) => XY | null
}

/** Décalage vertical au-dessus du point du porteur (le marqueur occupe le point lui-même). */
const BOMB_OFFSET_Y = 12

/**
 * Opacité de la bombe AU SOL : légèrement sous le porté, parce que la position est DÉRIVÉE
 * (dernier point du lâcheur) et non publiée — la nuance dit la nature de la donnée sans
 * la faire disparaître.
 */
const ALPHA_GROUND = 0.85

/**
 * bombCarrierActiveAt rend les portages qui COUVRENT l'image demandée. UNE SEULE bombe : au
 * plus un, mais on rend une liste (comme les calques frères) pour un tracé uniforme. Pure.
 */
export function bombCarrierActiveAt(
  carries: readonly ReplayBombCarry[],
  frame: number,
): ReplayBombCarry[] {
  const out: ReplayBombCarry[] = []
  for (const c of carries) {
    if (covers(c, frame)) out.push(c)
  }
  return out
}

/**
 * bombGroundAt rend la période FERMÉE dont le lâcher met la bombe au sol à cette image, ou
 * `null` : personne ne la porte, la dernière période fermée est à t1 < frame, aucune période
 * suivante n'a commencé, et aucune explosion n'est tombée dans ]t1, frame].
 *
 * `explosions` : les frames des `bomb_detonations`, triées — `null` quand l'horloge du film
 * n'est pas de confiance : le sol ne se dessine pas (cf. l'en-tête). Pure, testée à part.
 */
export function bombGroundAt(
  carries: readonly ReplayBombCarry[],
  explosions: readonly number[] | null,
  frame: number,
): ReplayBombCarry | null {
  if (explosions === null) return null
  let last: ReplayBombCarry | null = null
  for (const c of carries) {
    if (covers(c, frame)) return null // portée : jamais au sol en même temps
    if (c.t1 < frame && c.closed && (last === null || c.t1 > last.t1)) last = c
  }
  if (last === null) return null
  for (const e of explosions) {
    if (e > last.t1 && e <= frame) return null // la bombe a sauté : elle n'existe plus
  }
  return last
}

/** Opacité du glyphe porté — la pulsation d'un portage « ouvert » comprise. */
/**
 * drawBombCarrier peint la bombe de l'image : sur son porteur courant (par-dessus le marqueur),
 * ou au sol sur le dernier point de son lâcheur. Un joueur non localisable n'est PAS dessiné :
 * la bombe n'a pas de position propre, et l'inventer serait affirmer une place que le film ne
 * donne pas à cette image.
 */
export function drawBombCarrier(
  ctx: CanvasRenderingContext2D,
  layer: BombCarrierInput,
  carries: readonly ReplayBombCarry[],
  explosions: readonly number[] | null,
  view: CanvasView,
  frame: number,
): void {
  for (const c of bombCarrierActiveAt(carries, frame)) {
    const w = layer.posOf(c.xuid, frame)
    if (!w) continue
    const at = projectTo(view, w)
    // La bombe se pose AU-DESSUS du marqueur (celui-ci occupe le point) : le décalage est
    // appliqué ICI, le glyphe partagé ne connaît que son centre.
    drawBombGlyph(ctx, { x: at.x, y: at.y - BOMB_OFFSET_Y }, {
      ink: layer.style.ink,
      outline: layer.style.outline,
      alpha: carriedGlyphAlpha(c.closed, frame, layer.style.reducedMotion),
    })
    ctx.globalAlpha = 1
    return // une seule bombe : portée, donc jamais au sol à la même image
  }
  const ground = bombGroundAt(carries, explosions, frame)
  if (ground) {
    // AU SOL : au dernier point du lâcheur, à l'INSTANT du lâcher (t1) — la position ne se
    // relit pas à l'image courante, la bombe ne suit pas un joueur qui ne la porte plus.
    const w = layer.posOf(ground.xuid, ground.t1)
    if (w) {
      const at = projectTo(view, w)
      drawBombGlyph(ctx, at, {
        ink: layer.style.ink,
        outline: layer.style.outline,
        alpha: ALPHA_GROUND,
      })
    }
  }
  ctx.globalAlpha = 1
}
