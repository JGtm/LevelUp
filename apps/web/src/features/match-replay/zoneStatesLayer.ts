/**
 * zoneStatesLayer.ts — L'ÉTAT VIVANT DES ZONES (schéma 15) : qui tient quoi à l'image courante.
 *
 * POURQUOI IL VIT À CÔTÉ DU CALQUE STATIQUE, ET PAS DEDANS. `objectivesLayer.ts` porte la
 * GÉOMÉTRIE des objectifs du mode : elle ne dépend ni de l'image ni de la lecture, se cuit une
 * fois hors écran et se recopie. Celui-ci porte l'ÉTAT, qui change à chaque image — même
 * partage que les socles d'arme, et la seule façon de garder les deux fichiers sous le seuil du
 * dépôt. Les deux tracent la MÊME forme : le contour vient de `traceZonePath`, jamais d'une
 * seconde copie de la géométrie.
 *
 * CE QUE LE CALQUE MONTRE : la zone TEINTÉE de l'encre du camp qui la tient, la colline ACTIVE
 * en surbrillance, et un arc de progression quand la jauge a été mesurée. Une zone sans état à
 * cette frame n'est PAS repeinte : elle garde le trait faible du calque statique, et paraît
 * estompée sous celles qui sont tenues.
 *
 * AUCUN TEXTE, comme le calque statique : la lettre A/B/C affichée en jeu n'existe dans aucune
 * donnée décodée, et le garde-fou du dossier l'interdit.
 */
import {
  traceZonePath,
  type CanvasView,
  type ObjectiveElementReady,
} from './objectivesLayer'
import { canvasScale, worldToCanvas, type XY } from './replayLogic'

import type { ReplayZoneStateReady } from './replayNormalize'
/**
 * ZoneStateNow — ce qu'une zone montre à une frame donnée.
 *
 * `owner` vaut `null` quand PERSONNE ne la tient : c'est une mesure (la valeur neutre du canal
 * de propriété), pas une absence de donnée — d'où le champ, plutôt qu'un état omis.
 */
export interface ZoneStateNow {
  owner: number | null
  active: boolean
  /** Sommet de la jauge atteint pendant l'intervalle, dans [0, 1] ; `null` = zone non contestée. */
  progress: number | null
}

/**
 * zoneStateAt rend l'état de la zone `zoneRef` à la frame demandée, ou `null` quand aucun
 * intervalle ne la couvre — avant la première émission du film, la zone n'a pas d'état CONNU,
 * et la dessiner « neutre » affirmerait quelque chose que l'artefact ne dit pas.
 *
 * FONCTION PURE, testée à part : c'est elle que le rendu appelle à chaque image, et c'est elle
 * que l'onglet des objectifs vivants réutilisera.
 */
export function zoneStateAt(
  states: readonly ReplayZoneStateReady[],
  zoneRef: number,
  frame: number,
): ZoneStateNow | null {
  for (const st of states) {
    if (st.zoneRef !== zoneRef) continue
    for (const sp of st.spans) {
      if (frame < sp.t0 || frame > sp.t1) continue
      return { owner: sp.owner ?? null, active: sp.active, progress: sp.progress ?? null }
    }
  }
  return null
}

/**
 * zoneElementsOf rend les éléments SURFACIQUES dans l'ordre servi — celui que `zoneRef` indexe.
 *
 * L'ORDRE EST LE CONTRAT : `normalizeMapObjectives` empile les zones puis les marqueurs, en
 * conservant l'ordre du serveur, et `zoneStates[].zoneRef` est l'index dans `mapObjectives.zones`.
 * Filtrer sur `kind` restitue donc exactement cette liste.
 */
export function zoneElementsOf(
  elements: readonly ObjectiveElementReady[],
): ObjectiveElementReady[] {
  return elements.filter((e) => e.kind === 'zone')
}

/** Style du calque VIVANT : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface ZoneStateStyle {
  /** Encre d'un camp ; `null` = camp inconnu (aucune ligne « moi ») — le liseré reste neutre. */
  colorOfOwner: (team: number) => string | null
  /** Encre neutre : zone que personne ne tient, et arc de progression sans propriétaire. */
  neutral: string
}

// Réglages du calque vivant. Plus francs que le calque statique (qui reste dessous) : c'est le
// contraste entre les deux qui fait lire la bascule, et les zones sans état courant gardent
// leur trait faible — elles paraissent estompées sans qu'on ait à les repeindre.
const ZONE_HELD_FILL_ALPHA = 0.22
const ZONE_HELD_STROKE_ALPHA = 0.95
const ZONE_HELD_STROKE_WIDTH = 2.5
const ZONE_ACTIVE_FILL_ALPHA = 0.3
const ZONE_ACTIVE_STROKE_WIDTH = 3.5
const ZONE_PROGRESS_ALPHA = 0.9
const ZONE_PROGRESS_WIDTH = 3
const ZONE_PROGRESS_MIN_RADIUS = 10

/**
 * drawZoneStates peint, PAR-DESSUS le calque statique, ce que le film dit de chaque zone à
 * l'image courante : teinte du propriétaire, surbrillance de la zone ACTIVE, arc de progression.
 *
 * CE CALQUE N'EST PAS STATIQUE, et c'est tout le point : sa géométrie ne bouge pas, son ÉTAT
 * change à chaque image — comme les socles d'arme. Il se peint donc dans la boucle, pas dans un
 * canvas cuit une fois.
 *
 * AUCUN TEXTE, comme le calque statique : la lettre A/B/C affichée en jeu n'existe dans aucune
 * donnée décodée, et le garde-fou du fichier l'interdit.
 */
export function drawZoneStates(
  ctx: CanvasRenderingContext2D,
  zones: readonly ObjectiveElementReady[],
  states: readonly ReplayZoneStateReady[],
  view: CanvasView,
  frame: number,
  style: ZoneStateStyle,
): void {
  if (states.length === 0) return
  const px = (p: XY) => worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  zones.forEach((e, ref) => {
    const now = zoneStateAt(states, ref, frame)
    if (!now) return
    const ink = now.owner === null ? null : style.colorOfOwner(now.owner)
    paintZoneState(ctx, e, { px, scale, ink: ink ?? style.neutral, held: ink !== null, now })
    if (now.progress !== null) drawProgressArc(ctx, e, px, scale, now.progress, ink ?? style.neutral)
  })
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'une zone vivante a besoin de savoir (règle des 5 paramètres). */
interface ZonePaint {
  px: (p: XY) => XY
  scale: number
  ink: string
  /** `false` = personne ne tient la zone : LISERÉ SEUL, aucun remplissage. */
  held: boolean
  now: ZoneStateNow
}

/** Zone tenue : remplissage + liseré à l'encre du camp. Zone active : les deux, renforcés. */
function paintZoneState(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  p: ZonePaint,
): void {
  traceZonePath(ctx, e, p.px, p.scale)
  if (p.held || p.now.active) {
    ctx.globalAlpha = p.now.active ? ZONE_ACTIVE_FILL_ALPHA : ZONE_HELD_FILL_ALPHA
    ctx.fillStyle = p.ink
    ctx.fill()
  }
  ctx.globalAlpha = ZONE_HELD_STROKE_ALPHA
  ctx.strokeStyle = p.ink
  ctx.lineWidth = p.now.active ? ZONE_ACTIVE_STROKE_WIDTH : ZONE_HELD_STROKE_WIDTH
  ctx.stroke()
}


/**
 * drawProgressArc dessine la JAUGE de capture : un arc qui part du haut et se referme à mesure
 * que la zone est prise. Il est tracé HORS de la forme (rayon un peu plus grand) pour rester
 * lisible sur une zone déjà teintée.
 */
function drawProgressArc(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  px: (p: XY) => XY,
  scale: number,
  progress: number,
  ink: string,
): void {
  const c = px(e)
  const half = e.family === 'cylinder' ? e.radius : Math.max(e.halfX, e.halfY)
  const r = Math.max(half * scale + ZONE_PROGRESS_WIDTH, ZONE_PROGRESS_MIN_RADIUS)
  const start = -Math.PI / 2
  ctx.globalAlpha = ZONE_PROGRESS_ALPHA
  ctx.strokeStyle = ink
  ctx.lineWidth = ZONE_PROGRESS_WIDTH
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, start, start + 2 * Math.PI * Math.min(Math.max(progress, 0), 1))
  ctx.stroke()
}

