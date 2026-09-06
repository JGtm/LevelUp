/**
 * objectiveObjectsLayer.ts — LES POSITIONS ÉMISES du crâne d'Oddball (schéma 21) et son tracé au
 * sol. Ce fichier est le SOCLE GÉOMÉTRIQUE : le primitif `objectiveObjectAt` (dernier point émis,
 * et SEULEMENT dans [t0, t1]), le survol, et le tracé d'une présence libre. La RÈGLE de présence
 * — carried / free / absent, avec le test du prochain événement — vit dans `skullPresence.ts`,
 * qui consomme ce fichier en SENS UNIQUE (jamais l'inverse : ce socle n'importe pas la règle).
 *
 * POURQUOI IL VIT À CÔTÉ DE `flagCarriesLayer` ET NON DEDANS. Les deux dessinent un objet de
 * mode, mais ils ne lisent pas la même chose : `flagCarries` publie des INTERVALLES D'ÉTAT (le
 * drapeau porté n'a pas de position propre, le calque le colle à son porteur) ; ici on publie des
 * POSITIONS RÉELLEMENT ÉMISES par l'objet.
 *
 * L'INVARIANT, RÉÉCRIT AU SCHÉMA 23. Avant `skullCarries`, ce calque restait MUET dès qu'une vie
 * finissait — « prolonger la dernière position au-delà de t1 laisserait le crâne posé pendant
 * qu'un joueur court avec ». Cette justification est aujourd'hui INVERSÉE : le rendu SAIT désormais
 * quand le crâne est porté (`skullCarries`). Le nouvel invariant, porté par `skullPresenceAt` :
 * TENIR un repos qu'une PRISE corrobore (le porteur arrive sur la position tenue) ; rester MUET
 * pendant les portages connus ; rester ABSENT pendant un respawn ou une retombée (le prochain
 * début est alors une VIE, pas une prise — aucun fantôme au point de chute). AUCUNE INTERPOLATION,
 * jamais : une position tenue est une position RÉELLEMENT émise (le dernier repos), pas inventée.
 *
 * LE GLYPHE EST CELUI, PARTAGÉ, DU CRÂNE (`skullGlyph.ts`) : le MÊME dessin que le crâne porté
 * (`skullCarrierLayer`), pour qu'on reconnaisse le même objet — seule sa place change (au sol /
 * au-dessus d'un joueur). Habillé comme le drapeau depuis le 2026-08-28 (liseré à l'encre du
 * fond, taille agrandie, deux orbites) : cf. l'en-tête de `skullGlyph.ts` pour le pourquoi et la
 * limite (l'icône du jeu existe mais n'est pas cuite dans le document).
 *
 * AUCUN TEXTE, comme les calques voisins : ce qui se dit se dit dans l'infobulle.
 */
import { type XY } from './replayLogic'
import { drawSkullGlyph } from './skullGlyph'

import { type CanvasView, projectTo } from './replayView'
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
 * PRIMITIF PUR, INCHANGÉ : « LE DERNIER POINT ÉMIS, ET SEULEMENT À L'INTÉRIEUR DE LA VIE ». Hors
 * de [t0, t1], il rend `null` — il ne sait rien du portage ni du repos tenu. C'est `skullPresenceAt`
 * (cf. `skullPresence.ts`) qui compose ce primitif avec `skullCarries` pour décider s'il faut
 * TENIR un dernier repos (une prise le corrobore) ou rester absent. Ne pas déplacer cette décision
 * ici : ce fichier reste le socle géométrique, la règle vit une couche au-dessus.
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
 * drawFreeSkull trace le crâne LIBRE à la position MONDE `at` servie, avec le glyphe PARTAGÉ
 * (`skullGlyph.ts`) — le même que le crâne porté. L'appelant (le hook) ne l'invoque qu'après avoir
 * résolu une présence `free` via `skullPresenceAt` : ce fichier ne connaît ni le portage ni le
 * repos tenu, il pose un glyphe là où on le lui dit. `rolling` choisit l'alpha (bouge / immobile).
 */
export function drawFreeSkull(
  ctx: CanvasRenderingContext2D,
  layer: ObjectiveObjectsInput,
  at: XY,
  view: CanvasView,
  rolling: boolean,
): void {
  const p = projectTo(view, at)
  drawSkullGlyph(ctx, p, {
    ink: layer.style.ink,
    outline: layer.style.outline,
    alpha: rolling ? ALPHA_ROLLING : ALPHA_AT_REST,
  })
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
    const at = projectTo(view, now.at)
    const dx = at.x - point.x
    const dy = at.y - point.y
    if (dx * dx + dy * dy <= OBJECTIVE_OBJECT_HIT_RADIUS * OBJECTIVE_OBJECT_HIT_RADIUS) {
      return { now, at }
    }
  }
  return null
}
