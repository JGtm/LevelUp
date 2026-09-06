/**
 * thrusterDashFx.ts — LA POUSSÉE DU PROPULSEUR sur le pion du joueur : un dash bref, orienté.
 *
 * LA SOURCE est le document (schéma 38) : `abilityImpulses`, une entrée PLATE par geste —
 * (t, slot, family) — lue dans le corps `tag == 1` des composants i57/i59 du film et attribuée
 * par le rang de capacité de la MÊME VIE. Aucun seuil de vitesse, aucune heuristique. La
 * chaîne est validée contre une vérité terrain : sur le film `1cd3848a`, l'utilisateur a relevé
 * au Theater ses cinq usages (1:51, 1:54, 2:03, 2:05, 2:14) et le calque en date cinq, à moins
 * d'une seconde — précision 5/5, rappel 5/5.
 *
 * ## Pourquoi le PION et pas la fiche
 *
 * Décision de l'utilisateur (2026-09-03) : « trop bref pour mettre sur la fiche mais un petit
 * effet sur le pion oui, fait un truc qui fait comme un dash ». Le geste dure une demi-seconde ;
 * une colonne de fiche ou un éclat de vignette ne le montreraient pas. Ce qu'il montre, c'est
 * un DÉPLACEMENT — donc il se dessine là où le déplacement se lit, sur la carte.
 *
 * ## D'où vient la DIRECTION, et pourquoi elle n'est pas dans l'événement
 *
 * L'impulsion ne porte ni cap ni vecteur : elle date le geste, rien d'autre. La direction se
 * LIT donc dans la trajectoire du porteur autour de l'instant (`tracks[].points`, la même
 * interpolation que le marqueur). Fenêtre AVANT d'abord — la poussée suit le déclenchement,
 * mesurée à 6,2-8,8 m/s de pic contre 2,9-3,6 pour un instant ordinaire — puis repli sur la
 * fenêtre ARRIÈRE quand la vie se ferme dans la seconde (mort pendant le dash : `positionAt`
 * fige alors la dernière position, la fenêtre avant devient nulle).
 *
 * SANS DIRECTION MESURABLE, RIEN NE SE DESSINE. Un dash sans orientation serait une forme
 * inventée ; la doctrine du chantier est que le rejeu se tait plutôt que de deviner.
 *
 * ## Ce que ce fichier ne fait pas
 *
 * Aucun calque hors écran, aucune recuisson : deux projections et une poignée de traits par
 * dash actif, tout en géométrie pure sur un `CanvasRenderingContext2D` (même règle que
 * `grappleLayer.ts` et `replayDraw.ts`). Les encres arrivent DÉJÀ RÉSOLUES depuis les tokens
 * sémantiques — aucun littéral de couleur ici.
 */
import type { ReplayPoint } from '@/lib/api/types'

import { positionAt, type XY } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'
import { type CanvasView, projectTo } from '../model/replayView'
import { buildLivesBySlot, lifeOfSlotAt } from '../model/livesPosition'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
/**
 * LES FAMILLES QUI SE DESSINENT EN DASH. Ce ne sont PAS toutes les familles du calque : une
 * famille absente de cette table ne dessine RIEN, jamais la forme d'une voisine (même règle
 * que `activeEquipmentAt` et que `PLACEMENT_RENDER`).
 *
 * Le propulseur y est seul, et le répulseur n'y entrera pas par ce canal : il n'est PAS dans
 * les impulsions de capacité (négatif mesuré sur neuf canaux, lots R8/R9 du 2026-09-03).
 */
const DASH_FAMILIES: ReadonlySet<string> = new Set(['thruster'])

/**
 * DURÉE DE L'EFFET, en temps RÉEL du match (pas en images) : le dash est un geste court, il
 * doit durer aussi peu en lecture accélérée qu'en 1×. 460 ms — la demi-seconde du geste.
 */
export const THRUSTER_DASH_MS = 460

/**
 * FENÊTRE DE LECTURE DE LA DIRECTION, en temps réel. 280 ms : assez long pour que la poussée
 * ait produit un déplacement franc (1,7 à 2,5 m au pic mesuré), assez court pour ne pas
 * absorber un virage qui suivrait le dash.
 */
export const THRUSTER_DASH_HEADING_MS = 280

/**
 * PLANCHER NUMÉRIQUE de la lecture de direction, en unités monde. Ce n'est PAS un seuil de
 * détection — le geste est publié par le film, jamais déduit d'une vitesse : c'est la garde
 * qui empêche de normaliser un vecteur nul (vie figée, position clampée après la mort).
 */
const HEADING_MIN = 0.05

/** Un dash prêt à dessiner : l'instant, la vie, sa trajectoire et sa direction lue. */
export interface ThrusterDashFx {
  /** Image de l'impulsion mesurée — l'effet part de là, jamais avant. */
  frame: number
  /** Slot de la vie qui pousse : la couleur d'équipe s'y résout, à l'image de l'impulsion. */
  slot: number
  /** Les points de SA vie — la position du pion s'y relit à chaque image, le dash le suit. */
  points: ReplayPoint[]
  /** Départ de la lecture de direction, en monde. */
  from: XY
  /** Arrivée de la lecture de direction, en monde. La direction est `to - from`, projetée. */
  to: XY
}

/**
 * dashHeading lit la direction du geste dans la trajectoire, autour de l'instant.
 *
 * DEUX FENÊTRES, DANS CET ORDRE : l'AVANT (la poussée suit le déclenchement) puis l'ARRIÈRE
 * (repli quand la vie se ferme pendant le dash). Aucune des deux ne mesure une vitesse : elles
 * ne servent qu'à orienter une forme dont l'existence est, elle, publiée par le film.
 */
export function dashHeading(
  points: ReplayPoint[],
  frame: number,
  span: number,
): { from: XY; to: XY } | null {
  const at = positionAt(points, frame)
  if (!at) return null
  const after = positionAt(points, frame + span)
  if (after && dist2(at, after) > HEADING_MIN * HEADING_MIN) return { from: at, to: after }
  const before = positionAt(points, frame - span)
  if (before && dist2(before, at) > HEADING_MIN * HEADING_MIN) return { from: before, to: at }
  return null
}

function dist2(a: XY, b: XY): number {
  const dx = b.x - a.x
  const dy = b.y - a.y
  return dx * dx + dy * dy
}

/**
 * buildThrusterDashFx précalcule les dashes dessinables : chaque impulsion d'une famille
 * dessinable, jointe à LA VIE QUI COUVRE SON INSTANT et à sa direction lue une fois pour toutes.
 *
 * UNE TRACE = UNE VIE, ET LE SLOT DE BIPED EST RÉATTRIBUÉ À CHAQUE RÉAPPARITION. C'est
 * l'invariant du dossier (`shotFx.ts`, `fireMark.ts`, `riftStations.ts`, `replayMarkers.ts`) et
 * il ne souffre aucun raccourci : indexer `slot -> points` ne garderait que la DERNIÈRE piste
 * du slot, et une impulsion d'une vie antérieure irait chercher les points d'une autre vie —
 * au mieux elle disparaîtrait (l'instant précède les points retenus, `positionAt` rend `null`),
 * au pire elle peindrait un sillage à la position et dans la direction d'UN AUTRE JOUEUR.
 * On groupe donc par slot, puis on retient la vie qui COUVRE l'instant — le patron exact de
 * `buildShotFx` et `buildFireMarks`.
 *
 * `spanFrames` est la fenêtre de direction CONVERTIE par l'appelant (le hook), parce que la
 * durée d'une image dépend du document. Une impulsion sans vie couvrante ou sans direction
 * mesurable ne produit AUCUNE entrée — le calque ne dessine que ce qu'il sait orienter.
 */
export function buildThrusterDashFx(
  doc: ReplayDocumentReady,
  spanFrames: number,
): ThrusterDashFx[] {
  if (doc.abilityImpulses.length === 0) return []
  const bySlot = buildLivesBySlot(doc.tracks)
  const out: ThrusterDashFx[] = []
  for (const imp of doc.abilityImpulses) {
    if (!DASH_FAMILIES.has(imp.family)) continue
    const track = lifeOfSlotAt(bySlot, imp.slot, imp.t)
    if (!track || track.points.length === 0) continue
    const heading = dashHeading(track.points, imp.t, spanFrames)
    if (!heading) continue
    out.push({
      frame: imp.t,
      slot: imp.slot,
      points: track.points,
      from: heading.from,
      to: heading.to,
    })
  }
  return out
}

/** L'état du dash à un âge donné : jusqu'où il s'étire, et avec quelle présence. */
interface DashProgress {
  /** Part de la longueur pleine, dans [0, 1]. */
  reach: number
  alpha: number
}

/** Présence du sillage : franche au départ, éteinte à la fin de la fenêtre. */
const DASH_ALPHA = 0.9
/**
 * Sous MOUVEMENT RÉDUIT, la forme ne bouge plus et ne s'estompe plus : elle est POSÉE, pleine
 * longueur et opacité constante, pendant toute sa fenêtre. C'est la convention de
 * `revealAlpha` (threatSensor.ts) — l'information reste, l'animation s'éteint — et une
 * opacité plus basse compense l'absence de fondu, qui ferait sinon une tache immobile trop
 * lourde sur la carte.
 */
const DASH_ALPHA_STATIC = 0.55

/**
 * dashProgress dit ce qu'il faut dessiner à cet âge, ou `null` hors de la fenêtre.
 *
 * L'étirement suit un easeOutCubic : la poussée part vite puis se calme — le même patron que
 * l'onde du capteur et que l'onde de choc de l'explosion, pour la même raison : c'est ainsi
 * qu'une impulsion se lit.
 */
export function dashProgress(ageMs: number, reducedMotion: boolean): DashProgress | null {
  if (!(ageMs >= 0) || ageMs > THRUSTER_DASH_MS) return null
  if (reducedMotion) return { reach: 1, alpha: DASH_ALPHA_STATIC }
  const p = ageMs / THRUSTER_DASH_MS
  return { reach: 1 - Math.pow(1 - p, 3), alpha: (1 - p) * DASH_ALPHA }
}

// --- Géométrie du tracé, en pixels d'ÉCRAN (multipliés par la densité `k`) -----------------

/**
 * Le sillage part DERRIÈRE le marqueur, pas dessus : `DASH_GAP` le décolle du noyau (3,4 px de
 * rayon plus son liseré) pour que le pion reste lisible et que la poussée se lise comme en
 * sortant. Longueur volontairement courte — c'est un geste, pas une traînée.
 */
const DASH_GAP = 5
const DASH_LEN = 20
const DASH_HALF_W = 3.2
/** Le sillage est un aplat : il passe SOUS les chevrons, qui portent la lecture. */
const DASH_WAKE_ALPHA = 0.55
/**
 * Deux chevrons — des lignes de vitesse — posés le long du sillage. Ils pointent DANS le sens
 * de la poussée : c'est eux qui disent la direction sans ajouter de flèche.
 */
const CHEVRON_AT: readonly number[] = [0.5, 0.85]
const CHEVRON_HALF = 3.4
const CHEVRON_DEPTH = 2.6
const CHEVRON_WIDTH = 1.5

/** L'horloge du calque : image courante, durée réelle d'une image, densité, préférence. */
interface DashTime {
  frame: number
  /** Durée RÉELLE d'une image — l'effet dure en ms de match, jamais en nombre d'images. */
  frameMs: number
  k: number
  reducedMotion: boolean
}

/** Le style : la couleur d'équipe de la vie qui pousse. `null` = vie sans propriétaire. */
export interface DashStyle {
  colorOfSlot: (slot: number, frame: number) => string | null
}

/**
 * drawThrusterDashLayer peint les dashes actifs à l'image courante.
 *
 * LA COULEUR SE RÉSOUT À L'IMAGE DE L'IMPULSION, jamais à l'image courante : un slot de biped
 * est réattribué aux réapparitions, et l'image courante peut déjà appartenir à une AUTRE vie
 * (même règle que `SensorReveal.ownerFrame`).
 */
export function drawThrusterDashLayer(
  ctx: CanvasRenderingContext2D,
  fx: readonly ThrusterDashFx[],
  view: CanvasView,
  time: DashTime,
  style: DashStyle,
): void {
  for (const f of fx) {
    const prog = dashProgress((time.frame - f.frame) * time.frameMs, time.reducedMotion)
    if (!prog) continue
    const color = style.colorOfSlot(f.slot, f.frame)
    if (!color) continue
    const head = positionAt(f.points, time.frame)
    if (!head) continue
    const dir = screenDirection(f, view)
    if (!dir) continue
    // SAUVEGARDE PAR DASH, comme le calque du grappin : l'état du contexte revient intact à
    // l'appelant, et un film sans propulseur n'émet PAS un seul appel (le calque suivant
    // hérite exactement de ce qu'il avait avant).
    ctx.save()
    drawDash(ctx, project(head, view), dir, prog, { k: time.k, color })
    ctx.restore()
  }
}

/**
 * screenDirection projette les deux bornes de la lecture et normalise EN PIXELS.
 *
 * La normalisation se fait après projection et pas avant : le passage monde -> canvas peut
 * inverser un axe et n'a pas la même échelle en x et en y — un vecteur unitaire monde n'y
 * resterait pas unitaire, ni même orienté pareil.
 */
function screenDirection(f: ThrusterDashFx, view: CanvasView): XY | null {
  const a = project(f.from, view)
  const b = project(f.to, view)
  const dx = b.x - a.x
  const dy = b.y - a.y
  const len = Math.hypot(dx, dy)
  if (!(len > 0)) return null
  return { x: dx / len, y: dy / len }
}

/** drawDash : le sillage derrière le pion, puis les chevrons qui disent le sens. */
function drawDash(
  ctx: CanvasRenderingContext2D,
  c: XY,
  u: XY,
  prog: DashProgress,
  paint: { k: number; color: string },
): void {
  const k = paint.k
  const n = { x: -u.y, y: u.x }
  const len = DASH_LEN * k * prog.reach
  // Le sillage : un coin qui part du bord du marqueur et s'affine vers la queue.
  const bx = c.x - u.x * DASH_GAP * k
  const by = c.y - u.y * DASH_GAP * k
  const w = DASH_HALF_W * k
  ctx.fillStyle = paint.color
  ctx.strokeStyle = paint.color
  ctx.globalAlpha = prog.alpha * DASH_WAKE_ALPHA
  ctx.beginPath()
  ctx.moveTo(bx + n.x * w, by + n.y * w)
  ctx.lineTo(bx - n.x * w, by - n.y * w)
  ctx.lineTo(bx - u.x * len, by - u.y * len)
  ctx.closePath()
  ctx.fill()
  // Les chevrons, posés le long du sillage et pointant dans le sens de la poussée.
  ctx.globalAlpha = prog.alpha
  ctx.lineWidth = CHEVRON_WIDTH * k
  for (const at of CHEVRON_AT) {
    const px = bx - u.x * len * at
    const py = by - u.y * len * at
    ctx.beginPath()
    ctx.moveTo(px + n.x * CHEVRON_HALF * k, py + n.y * CHEVRON_HALF * k)
    ctx.lineTo(px + u.x * CHEVRON_DEPTH * k, py + u.y * CHEVRON_DEPTH * k)
    ctx.lineTo(px - n.x * CHEVRON_HALF * k, py - n.y * CHEVRON_HALF * k)
    ctx.stroke()
  }
}

function project(p: XY, view: CanvasView): XY {
  return projectTo(view, p)
}
