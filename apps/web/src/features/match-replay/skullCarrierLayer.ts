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
 * calque remplit. Le glyphe est le MÊME disque, pour qu'on reconnaisse le même objet — seule sa
 * place change (au sol / au-dessus d'un joueur).
 *
 * AUCUNE VIGNETTE DU JEU : le crâne n'est pas un `weap`, il n'a pas d'icône extraite. Le glyphe est
 * tracé au canvas — un disque cerné — à l'encre sémantique résolue par l'appelant (règle
 * color-tokens). Aucun texte : le porteur est déjà nommé par son marqueur.
 */
import { worldToCanvas, type XY } from './replayLogic'

import type { CanvasView } from './objectivesLayer'
import type { ReplaySkullCarry } from './replayNormalize'

/** Style du calque : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface SkullCarrierStyle {
  /** Encre du disque du crâne (token du thème déjà résolu par l'appelant). */
  ink: string
  /** Encre du liseré, pour détacher le glyphe du marqueur et du fond. */
  edge: string
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

// Réglages du glyphe. Le crâne se pose AU-DESSUS du marqueur du joueur, un peu plus petit que lui.
const SKULL_RADIUS = 5
/** Décalage vertical au-dessus du point du joueur (le marqueur occupe le point lui-même). */
const SKULL_OFFSET_Y = 12
const SKULL_STROKE_WIDTH = 1.5

/** Opacité pleine : un fait a fermé le portage avant la fin de l'axe. */
const ALPHA_SOLID = 0.95
/** Bornes de la pulsation d'un portage « ouvert » (`closed` faux), et sa période en images. */
const PULSE_MIN = 0.42
const PULSE_MAX = 0.78
const PULSE_PERIOD_FRAMES = 26

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
    if (frame >= c.t0 && frame <= c.t1) out.push(c)
  }
  return out
}

/** Opacité du glyphe pour un portage — la pulsation d'un portage « ouvert » comprise. */
function alphaOf(closed: boolean, frame: number, reducedMotion: boolean): number {
  if (closed) return ALPHA_SOLID
  if (reducedMotion) return (PULSE_MIN + PULSE_MAX) / 2
  const phase = (2 * Math.PI * frame) / PULSE_PERIOD_FRAMES
  return PULSE_MIN + (PULSE_MAX - PULSE_MIN) * (0.5 + 0.5 * Math.sin(phase))
}

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
    const at = worldToCanvas(w, view.bounds, view.width, view.height, view.pad)
    drawSkullGlyph(ctx, at, {
      ink: layer.style.ink,
      edge: layer.style.edge,
      alpha: alphaOf(c.closed, frame, layer.style.reducedMotion),
    })
  }
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'un glyphe a besoin de savoir. */
interface SkullGlyphPaint {
  ink: string
  edge: string
  alpha: number
}

/**
 * drawSkullGlyph trace le crâne porté : un disque cerné, posé AU-DESSUS du marqueur du porteur. Le
 * MÊME disque que le crâne libre (`objectiveObjectsLayer`), pour qu'on reconnaisse le même objet ;
 * seule sa place change.
 */
export function drawSkullGlyph(
  ctx: CanvasRenderingContext2D,
  at: XY,
  paint: SkullGlyphPaint,
): void {
  ctx.globalAlpha = paint.alpha
  ctx.beginPath()
  ctx.arc(at.x, at.y - SKULL_OFFSET_Y, SKULL_RADIUS, 0, Math.PI * 2)
  ctx.fillStyle = paint.ink
  ctx.fill()
  ctx.lineWidth = SKULL_STROKE_WIDTH
  ctx.strokeStyle = paint.edge
  ctx.stroke()
}
