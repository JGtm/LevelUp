/**
 * objectiveObjectsLayer.ts — LES OBJETS D'OBJECTIF LIBRES (schéma 21) : le crâne d'Oddball
 * dessiné là où il est quand PERSONNE ne le porte.
 *
 * POURQUOI IL VIT À CÔTÉ DE `flagCarriesLayer` ET NON DEDANS. Les deux dessinent un objet de
 * mode, mais ils ne lisent pas la même chose et ne peuvent pas mentir de la même façon :
 *
 *  - `flagCarries` publie des INTERVALLES D'ÉTAT, dont deux états PORTÉS. Le drapeau porté n'a
 *    pas de position propre — le calque le colle au marqueur de son porteur.
 *  - ce calque-ci publie des POSITIONS RÉELLEMENT ÉMISES par l'objet. Il ne connaît aucun
 *    porteur et n'en dessine aucun.
 *
 * LE SILENCE PENDANT LE PORTAGE EST LA PROPRIÉTÉ CENTRALE DE CE CALQUE, et il est VOULU. Entre
 * deux vies libres il y a un trou : quelqu'un porte le crâne, et le document ne dit pas qui —
 * l'oracle du porteur a été mesuré puis RÉFUTÉ (phase D4 : 40,6 à 66,7 % de trous à porteur
 * unique pour un seuil de 90 %, témoin hors trou à 66,7 et 71,4 %). Dessiner le crâne sur un
 * joueur pendant ces trous afficherait une certitude que la mesure refuse. On ne dessine RIEN.
 *
 * AUCUNE INTERPOLATION ENTRE DEUX VIES, ni même entre deux points d'une même vie : le crâne est
 * à la DERNIÈRE position qu'il a émise, ou nulle part. Une position interpolée serait une
 * position inventée, et c'est exactement ce que ce lot a passé six phases à ne pas faire.
 *
 * LE GLYPHE EST CELUI, PARTAGÉ, DU CRÂNE (`skullGlyph.ts`) : le MÊME dessin que le crâne porté
 * (`skullCarrierLayer`), pour qu'on reconnaisse le même objet — seule sa place change (au sol /
 * au-dessus d'un joueur). Habillé comme le drapeau depuis le 2026-08-28 (liseré à l'encre du
 * fond, taille agrandie, deux orbites) : cf. l'en-tête de `skullGlyph.ts` pour le pourquoi et la
 * limite (l'icône du jeu existe mais n'est pas cuite dans le document).
 *
 * AUCUN TEXTE, comme les calques voisins : ce qui se dit se dit dans l'infobulle.
 */
import { worldToCanvas, type XY } from './replayLogic'
import { drawSkullGlyph } from './skullGlyph'

import type { CanvasView } from './objectivesLayer'
import type { ReplayObjectiveObjectReady } from './replayNormalize'

/** Le rayon de SURVOL : plus généreux que le tracé, comme pour le drapeau. */
export const OBJECTIVE_OBJECT_HIT_RADIUS = 11

/**
 * ALPHA_AT_REST — le glyphe d'un crâne IMMOBILE (vie réduite à un point, ou dernier point d'une
 * vie terminée). Il est en jeu, il ne bouge pas : pleine encre, sans effet.
 */
const ALPHA_AT_REST = 0.95

/**
 * ALPHA_ROLLING — le glyphe pendant que l'objet BOUGE encore (la vie a un point à cette image et
 * un autre après). Légèrement plus soutenu : c'est le moment où il compte le plus.
 */
const ALPHA_ROLLING = 1

/** Ce que le calque a besoin de savoir pour se dessiner (règle des 5 paramètres). */
export interface ObjectiveObjectsInput {
  style: {
    /** L'encre sémantique de l'objet, résolue par l'appelant — jamais une couleur en dur ici. */
    ink: string
    /** L'encre du FOND : le liseré du crâne, comme celui du drapeau (`skullGlyph.ts`). */
    outline: string
  }
}

/** ObjectiveObjectNow — ce qu'un objet d'objectif montre à une image donnée. */
export interface ObjectiveObjectNow {
  /** La vie qui porte cette position. */
  life: ReplayObjectiveObjectReady
  /** La position en coordonnées MONDE. */
  at: XY
  /** L'objet bouge-t-il encore à cette image ? (un point plus tard dans la même vie) */
  rolling: boolean
}

/**
 * objectiveObjectAt rend la position de l'objet à l'image servie, ou `null` s'il ne réplique pas.
 *
 * LA RÈGLE EST « LE DERNIER POINT ÉMIS, ET SEULEMENT À L'INTÉRIEUR DE LA VIE » : hors de
 * [t0, t1], l'objet ne réplique rien — quelqu'un le porte, ou il n'existe pas encore. Rendre sa
 * dernière position connue au-delà de `t1` le laisserait posé au sol pendant qu'un joueur court
 * avec, ce qui est précisément le mensonge que ce calque refuse.
 */
export function objectiveObjectAt(
  life: ReplayObjectiveObjectReady,
  frame: number,
): ObjectiveObjectNow | null {
  if (frame < life.t0 || frame > life.t1 || life.pts.length === 0) return null
  let idx = -1
  for (let i = 0; i < life.pts.length; i += 1) {
    if (life.pts[i].t > frame) break
    idx = i
  }
  if (idx < 0) return null
  const p = life.pts[idx]
  return { life, at: { x: p.x, y: p.y }, rolling: idx < life.pts.length - 1 }
}

/** objectiveObjectsAt rend TOUS les objets qui répliquent à cette image. */
export function objectiveObjectsAt(
  lives: readonly ReplayObjectiveObjectReady[],
  frame: number,
): ObjectiveObjectNow[] {
  const out: ObjectiveObjectNow[] = []
  for (const life of lives) {
    const now = objectiveObjectAt(life, frame)
    if (now) out.push(now)
  }
  return out
}

/**
 * drawObjectiveObjects trace les objets d'objectif libres de l'image servie — chacun avec le
 * glyphe PARTAGÉ du crâne (`skullGlyph.ts`), le même que le crâne porté.
 */
export function drawObjectiveObjects(
  ctx: CanvasRenderingContext2D,
  layer: ObjectiveObjectsInput,
  lives: readonly ReplayObjectiveObjectReady[],
  view: CanvasView,
  frame: number,
): void {
  for (const now of objectiveObjectsAt(lives, frame)) {
    const at = worldToCanvas(now.at, view.bounds, view.width, view.height, view.pad)
    drawSkullGlyph(ctx, at, {
      ink: layer.style.ink,
      outline: layer.style.outline,
      alpha: now.rolling ? ALPHA_ROLLING : ALPHA_AT_REST,
    })
  }
  ctx.globalAlpha = 1
}

/** Ce qu'un survol a trouvé : l'objet, sa position LUE À CET INSTANT, et où poser l'infobulle. */
export interface ObjectiveObjectHit {
  now: ObjectiveObjectNow
  /** Point CANVAS du glyphe — l'infobulle se pose là, pas sous le pointeur brut. */
  at: XY
}

/**
 * objectiveObjectHitAt rend l'objet dont le glyphe se trouve sous le point CANVAS servi.
 *
 * LE SURVOL REJOUE EXACTEMENT LA GÉOMÉTRIE DU TRACÉ (même conversion, même centre) : viser une
 * forme dessinée ailleurs ne toucherait rien, et ce genre d'écart ne se voit pas à la lecture.
 */
export function objectiveObjectHitAt(
  lives: readonly ReplayObjectiveObjectReady[],
  view: CanvasView,
  frame: number,
  point: XY,
): ObjectiveObjectHit | null {
  for (const now of objectiveObjectsAt(lives, frame)) {
    const at = worldToCanvas(now.at, view.bounds, view.width, view.height, view.pad)
    const dx = at.x - point.x
    const dy = at.y - point.y
    if (dx * dx + dy * dy <= OBJECTIVE_OBJECT_HIT_RADIUS * OBJECTIVE_OBJECT_HIT_RADIUS) {
      return { now, at }
    }
  }
  return null
}
