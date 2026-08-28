/**
 * fireMark.ts — LE « ! » DANS LE POINT DU TIREUR, à l'instant du tir.
 *
 * DEMANDE UTILISATEUR DU 2026-08-24, mot pour mot : « mettre un symbole point d'exclamation
 * quand ils tirent [...] dans le point du joueur, centré, pas à côté. Sans rien changer en
 * apparence. » Le calque ne touche donc à RIEN du marqueur (couleur, forme, anneaux, nom) :
 * il pose un glyphe par-dessus le noyau, centré, et seulement pendant la fenêtre du tir.
 *
 * LA MESURE EST CELLE DES ÉCLAIRS DE BOUCHE (shotFx.ts) : `doc.shots`, les tirs qui ont
 * APPLIQUÉ un dégât — le film n'enregistre pas les tirs dans le vide. La fenêtre de
 * rémanence est la MÊME que celle de l'éclair (SHOT_HOLD_MS, 600 ms) : deux effets du même
 * événement qui vivraient des durées différentes se liraient comme deux événements. La
 * MÊLÉE n'entre pas, pour la même raison qu'à l'éclair : un coup de marteau n'est pas un
 * tir.
 *
 * LE GLYPHE EST UNE GÉOMÉTRIE, PAS UN TEXTE : à ~5 px de haut, un « ! » de police se
 * dissout (rendu sous-pixel, métriques de baseline) ; une barre à bouts ronds et un point
 * restent nets à toutes les densités. Il est dessiné à l'encre de CONTOUR des étiquettes
 * (sombre dans les deux thèmes, cf. replayLabels.ts) : c'est la seule encre du calque déjà
 * prévue pour se détacher d'une couleur d'équipe, quelle qu'elle soit.
 *
 * Pas de React : logique pure + un CanvasRenderingContext2D (même règle que grappleLayer).
 */
import { familyOf } from './shotEffects'
import { isAliveAt, positionAt, worldToCanvas } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
interface CanvasView {
  bounds: { minX: number; minY: number; maxX: number; maxY: number }
  width: number
  height: number
  pad: number
}

/** Un tir prêt à marquer : son instant, et la VIE qui le porte. */
export interface FireMarkEntry {
  /** Frame du tir sur la grille du rejeu. */
  frame: number
  /** La vie qui couvre l'instant du tir : le « ! » suit SON marqueur, pas le point d'impact. */
  track: ReplayTrackReady
}

/**
 * buildFireMarks précalcule les tirs marquables : chaque tir non-mêlée joint à la vie qui
 * COUVRE son instant (une trace = une vie, le slot est réattribué à chaque réapparition —
 * même disambiguïsation que buildShotFx). Un tir sans vie couvrante ne marque personne.
 */
export function buildFireMarks(doc: ReplayDocumentReady): FireMarkEntry[] {
  if (doc.shots.length === 0) return []
  const bySlot = new Map<number, ReplayTrackReady[]>()
  for (const t of doc.tracks) {
    const lives = bySlot.get(t.slot)
    if (lives) lives.push(t)
    else bySlot.set(t.slot, [t])
  }
  const out: FireMarkEntry[] = []
  for (const s of doc.shots) {
    if (familyOf(doc.weaponLabels?.[s.w ?? '']?.fx) === 'melee') continue
    const track = (bySlot.get(s.slot) ?? []).find((l) => isAliveAt(l, s.t))
    if (track) out.push({ frame: s.t, track })
  }
  return out
}

// Le glyphe, en pixels d'ÉCRAN (comme tout ce qui s'adresse à l'œil) : il tient DANS le
// noyau du marqueur (rayon 3,4 — cf. CORE_RADIUS, replayMarkers.ts) sans jamais le couvrir
// entièrement — la couleur d'équipe reste lisible autour.
const FIRE_BAR_TOP = 2.1
const FIRE_BAR_BOTTOM = 0.4
const FIRE_DOT_Y = 1.8
const FIRE_DOT_RADIUS = 0.6
const FIRE_WIDTH = 1.1

/** Style du calque : la fenêtre du tir, la porte des vies dessinables, l'encre du thème. */
export interface FireMarkStyle {
  frame: number
  /** Rémanence d'un tir, en frames — la MÊME fenêtre que l'éclair de bouche. */
  hold: number
  /**
   * La PORTE du calque des joueurs : un slot sans propriétaire à l'image n'a pas de marqueur,
   * donc pas de « ! ». Résolue PAR IMAGE (slot réattribué entre manches) — ici l'image courante,
   * la vie étant vivante à cet instant (contrôlé ci-dessous).
   */
  colorOfSlot: (slot: number, frame: number) => string | null
  /** Encre du glyphe (contour des étiquettes ; l'appelant fournit son repli). */
  ink: string
  /** Densité du canevas. */
  k: number
}

/**
 * drawFireMarks pose le « ! » au centre du marqueur de chaque tireur de la fenêtre
 * courante. La position est celle du MARQUEUR à la frame courante (positionAt, la même
 * interpolation) : le joueur bouge pendant la rémanence, le glyphe le suit. Une vie déjà
 * fermée ne marque plus rien — sa croix de mort n'est pas un joueur qui tire.
 */
export function drawFireMarks(
  ctx: CanvasRenderingContext2D,
  entries: FireMarkEntry[],
  view: CanvasView,
  style: FireMarkStyle,
): void {
  ctx.lineCap = 'round'
  for (const e of entries) {
    const age = style.frame - e.frame
    if (age < 0 || age > style.hold) continue
    if (!isAliveAt(e.track, style.frame)) continue
    // La vie couvre l'image courante (contrôle ci-dessus) : y résoudre l'identité désigne bien
    // le propriétaire de CETTE vie, jamais celui d'une autre manche.
    if (!style.colorOfSlot(e.track.slot, style.frame)) continue
    const head = positionAt(e.track.points, style.frame)
    if (!head) continue
    const c = worldToCanvas(head, view.bounds, view.width, view.height, view.pad)
    const k = style.k
    ctx.strokeStyle = style.ink
    ctx.fillStyle = style.ink
    ctx.globalAlpha = 1
    ctx.lineWidth = FIRE_WIDTH * k
    ctx.beginPath()
    ctx.moveTo(c.x, c.y - FIRE_BAR_TOP * k)
    ctx.lineTo(c.x, c.y - FIRE_BAR_BOTTOM * k)
    ctx.stroke()
    ctx.beginPath()
    ctx.arc(c.x, c.y + FIRE_DOT_Y * k, FIRE_DOT_RADIUS * k, 0, Math.PI * 2)
    ctx.fill()
  }
}
