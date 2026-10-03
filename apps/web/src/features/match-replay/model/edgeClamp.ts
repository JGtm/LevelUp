/**
 * edgeClamp.ts — LA GÉOMÉTRIE PURE DU BORNAGE HORS CADRE (chantier B, plan escouade hors
 * cadre, 2026-09-10).
 *
 * POURQUOI. `worldToCanvas` (`lib/replay/replayLogic.ts`, exposé ici via `projectTo`) est une
 * projection affine SANS bornage : un point hors des bornes du cadrage se projette hors de la
 * toile, le bitmap du canvas l'écrête, et le marqueur disparaît sans repère. À zoom serré
 * (2x/3x), c'est la position de tout un joueur qui se perd. `edgeMarkFor` répond à une seule
 * question, sans rien dessiner : où poser un repère PLAQUÉ À LA MARGE, dans quelle direction
 * pointe-t-il vers la vraie position, et à quelle distance (en mètres) se trouve-t-elle.
 *
 * POURQUOI CE MODULE VIT ICI, ET PAS DANS `lib/replay/` (écart au chemin du plan
 * `PLAN_ESCOUADE_HORS_CADRE_2026-09-09.md`, §4 B0.1, corrigé sur pièce à l'exécution). Le
 * cadrage (`CanvasView`) et sa projection (`projectTo`, `scaleOf`) ne vivent PLUS dans
 * `lib/replay/` depuis le lot K3 (2026-09-05) : ils vivent dans `replayView.ts`, à côté, et
 * c'est cette même page d'en-tête qui interdit « une seconde règle de projection ». Aucun
 * fichier de `lib/replay/` (hors fixtures de test) n'importe quoi que ce soit de la feature —
 * l'inverse casserait la couche basse/haute du dépôt. Ce module respecte donc la même règle
 * que tous les calques (`replayMarkers.ts`, `flagCarriesLayer.ts`…) : il vit dans la feature,
 * à côté du cadrage qu'il consomme, jamais une seconde déclaration.
 *
 * UNITÉS. `c` est un point MONDE. `margeEcran` est en pixels d'ÉCRAN DE RÉFÉRENCE (avant mise à
 * l'échelle), `echelle` est le même facteur que `style.k` ailleurs dans le rejeu (densité du
 * canevas) — la marge PLAQUÉE suit donc la même convention que toute grandeur qui s'adresse à
 * l'œil dans cette feature. La distance rendue, elle, est en MÈTRES DU MONDE : elle divise
 * l'écart de pixels par `scaleOf(view)`, jamais l'inverse.
 *
 * `at`/`angle` SONT DANS L'ESPACE CANVAS (post-projection) : l'inversion de Y (monde +Y en
 * haut, toile +Y en bas) est déjà faite par `projectTo` — ce module ne la refait pas, il ne fait
 * que borner et mesurer ce qu'elle a produit.
 */
import type { XY } from '../../../lib/replay/replayLogic'
import { type CanvasView, projectTo, scaleOf } from './replayView'

/**
 * OFFSCREEN_MARGIN_PX — la marge de bornage CANONIQUE du chantier B, en pixels d'écran de
 * référence (avant mise à l'échelle par `k`). Tous les calques qui bornent un glyphe à la
 * marge (joueurs vivants, croix de mort, porteurs d'objectif) la PARTAGENT — jamais une copie
 * locale (CLAUDE.md n°6) : une marge qui diverge d'un calque à l'autre ferait flotter les
 * repères à des distances différentes du bord pour le même cadrage.
 */
export const OFFSCREEN_MARGIN_PX = 16

/** Ce que le bornage rend pour un point hors cadre : où le poser, vers où, à quelle distance. */
export interface EdgeMark {
  /** Position CANVAS où plaquer le repère, bornée à la marge sur les deux axes. */
  at: XY
  /** Angle CANVAS (radians), de `at` vers la position réelle projetée. */
  angle: number
  /** Distance entre le bord et la position réelle, en MÈTRES du monde. */
  distanceM: number
}

/**
 * edgeMarkFor projette `c` et le borne à `margeEcran * echelle` pixels du bord de `view`.
 *
 * Rend `null` quand la projection tombe DÉJÀ dans le cadre bordé (rien à signaler : le
 * marqueur normal suffit). Un point tout juste sur la marge est considéré DEDANS — la marge
 * est la dernière position valide, pas la première hors cadre.
 */
export function edgeMarkFor(
  c: XY,
  view: CanvasView,
  margeEcran: number,
  echelle: number,
): EdgeMark | null {
  const p = projectTo(view, c)
  const marge = margeEcran * echelle
  const minX = marge
  const maxX = view.width - marge
  const minY = marge
  const maxY = view.height - marge
  const dedans = p.x >= minX && p.x <= maxX && p.y >= minY && p.y <= maxY
  if (dedans) return null
  const at: XY = {
    x: Math.min(Math.max(p.x, minX), maxX),
    y: Math.min(Math.max(p.y, minY), maxY),
  }
  const dx = p.x - at.x
  const dy = p.y - at.y
  const distancePx = Math.hypot(dx, dy)
  return { at, angle: Math.atan2(dy, dx), distanceM: distancePx / scaleOf(view) }
}
