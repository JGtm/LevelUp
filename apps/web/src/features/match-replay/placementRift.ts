/**
 * placementRift — LA FAILLE SPATIO-TEMPORELLE du translocateur quantique, et son PASSAGE.
 *
 * POURQUOI UN FICHIER À PART. `placementShapes.ts` porte les formes des objets posés ; la
 * faille en est une, mais elle a fini par emporter davantage que les autres (deux encres, un
 * halo, et l'arc de téléportation qui relie deux positions). L'extraire garde
 * `placementShapes.ts` sous le seuil de 500 lignes, et suit le précédent de `placementWall.ts`.
 *
 * Comme le reste du dossier : pas de React, pas de couleur écrite — géométrie pure. L'encre
 * arrive du calque appelant, qui la tient des tokens du thème.
 */
import type { XY } from './replayLogic'
import { type PlacementView, project, type ShapeStyle } from './placementShapes'
import type { RiftTeleport } from './placementTeleport'

/**
 * Demi-hauteur de la faille, en pixels d'écran.
 *
 * PLUS HAUTE QUE LARGE, et c'est la seule chose qui la fait lire comme une DÉCHIRURE plutôt
 * que comme un marqueur : le losange de la balise était isotrope, la faille ne l'est pas.
 */
export const RIFT_HALF_HEIGHT_PX = 7

/** Demi-largeur au point le plus ouvert. Le rapport 1:3,5 est ce qui donne la lentille. */
const RIFT_HALF_WIDTH_PX = 2

const RIFT_LINE_WIDTH = 1.4
/** Le cœur : un trait vertical plein, plus court que la déchirure qui l'entoure. */
const RIFT_CORE_RATIO = 0.55
/** Rayon du halo, en multiples de la demi-hauteur : ce qui déborde de la déchirure. */
const RIFT_GLOW_RATIO = 1.9
const RIFT_GLOW_ALPHA = 0.34
const RIFT_LIP_ALPHA = 0.95

/**
 * RiftInk — les DEUX encres de la faille.
 *
 * POURQUOI PAS LA COULEUR DU POSEUR, et pourquoi deux. Le précédent est le MUR (verdict R2-5,
 * 2026-08-18) : un objet du terrain dont la couleur dit ce qu'il EST plutôt que qui l'a posé se
 * reconnaît d'un coup d'œil au milieu de huit trajectoires colorées. La faille est le cas le
 * plus net — l'utilisateur demande (2026-08-27) qu'elle « ressemble à un portail
 * interdimensionnel », et une teinte d'équipe la ferait lire comme un marqueur de camp. Ce
 * qu'on PERD est donc le même que pour le mur : le camp du poseur, lisible à l'infobulle.
 *
 * DEUX encres parce qu'un portail n'est pas un trait : il a un BORD et une LUMIÈRE qui en sort,
 * exactement comme l'éclair de bouche a son halo et son cœur.
 */
export interface RiftInk {
  /** Les lèvres de la déchirure — la teinte la plus saturée. */
  rim: string
  /** Le cœur et le halo — plus clair : c'est la lumière qui passe au travers. */
  core: string
}

/**
 * drawRift — la FAILLE SPATIO-TEMPORELLE ouverte par le translocateur quantique.
 *
 * POURQUOI CETTE FORME REMPLACE LE LOSANGE. Le calque dessinait une « balise » parce que c'est
 * ainsi que le lot du 2026-08-18 avait lu l'objet. L'utilisateur a corrigé la lecture le
 * 2026-08-27 : « y a pas vraiment de balise dans le jeu, c'est un équipement que le joueur
 * porte au poignet ; là ça crée une genre de faille spatio-temporelle sur sa position exacte ».
 * Un losange à cœur plein disait « point posé » ; deux arcs opposés qui se referment sur un
 * trait de lumière disent « ouverture ».
 *
 * AUCUNE ANIMATION, et c'est la même règle que pour le losange qu'elle remplace : rien dans le
 * film ne bat au rythme de cet objet. Une faille qui scintillerait affirmerait une activité que
 * personne n'a mesurée. Ce qui change ici est la FORME et la TEINTE — ce que l'objet EST —, pas
 * un comportement qu'on lui prêterait.
 */
export function drawRift(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: ShapeStyle,
  ink: RiftInk,
): void {
  const h = RIFT_HALF_HEIGHT_PX * style.k
  const w = RIFT_HALF_WIDTH_PX * style.k
  ctx.save()
  // LE HALO D'ABORD, sous les lèvres : un dégradé ELLIPTIQUE, étiré dans l'axe de la
  // déchirure. Un halo rond ferait une bulle ; c'est l'étirement qui dit « ça s'ouvre ».
  ctx.save()
  ctx.translate(c.x, c.y)
  ctx.scale(0.5, 1)
  const r = h * RIFT_GLOW_RATIO
  const g = ctx.createRadialGradient(0, 0, 0, 0, 0, r)
  g.addColorStop(0, ink.core)
  g.addColorStop(0.35, ink.core)
  g.addColorStop(1, 'transparent')
  ctx.globalAlpha = RIFT_GLOW_ALPHA
  ctx.fillStyle = g
  ctx.beginPath()
  ctx.arc(0, 0, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
  ctx.lineCap = 'round'
  ctx.lineWidth = RIFT_LINE_WIDTH * style.k
  // Les deux lèvres de la déchirure : deux arcs quadratiques opposés, sommet à sommet.
  ctx.globalAlpha = RIFT_LIP_ALPHA
  ctx.strokeStyle = ink.rim
  ctx.beginPath()
  ctx.moveTo(c.x, c.y - h)
  ctx.quadraticCurveTo(c.x + w, c.y, c.x, c.y + h)
  ctx.quadraticCurveTo(c.x - w, c.y, c.x, c.y - h)
  ctx.stroke()
  // Le cœur, dans l'encre claire : la lumière qui sort de l'ouverture.
  ctx.globalAlpha = 1
  ctx.strokeStyle = ink.core
  ctx.beginPath()
  ctx.moveTo(c.x, c.y - h * RIFT_CORE_RATIO)
  ctx.lineTo(c.x, c.y + h * RIFT_CORE_RATIO)
  ctx.stroke()
  ctx.restore()
}

/**
 * Durée du LIEN de téléportation, en millisecondes.
 *
 * BREF, parce que l'utilisateur l'a demandé bref (« un effet interdimensionnel qui relie
 * l'ancienne position à la nouvelle brièvement »), et surtout parce qu'un lien qui persiste
 * finirait par se confondre avec une trajectoire — ce qu'il n'est pas.
 */
export const RIFT_LINK_MS = 600
/** Amplitude de l'arc, en fraction de la distance : ce qui l'écarte du droit chemin. */
const RIFT_LINK_BOW = 0.18
const RIFT_LINK_WIDTH = 2.2
const RIFT_LINK_ALPHA = 0.55
/** Rétrécissement des deux bouts pendant l'effacement : de 1,0 à 0,45 fois l'échelle. */
const RIFT_LINK_END_MIN = 0.45

/**
 * drawRiftLink — LE PASSAGE : l'arc bref qui relie l'ancienne position à la nouvelle quand un
 * joueur se téléporte.
 *
 * CE QU'IL AFFIRME, ET CE QU'IL N'AFFIRME PAS. Il dit qu'un joueur était LÀ et qu'il est
 * maintenant ICI, à un instant daté — deux positions mesurées et un instant mesuré. Il ne
 * prétend PAS que le joueur a suivi cette courbe : entre les deux bouts d'une téléportation il
 * n'y a pas de trajectoire, il y a un LIEN. D'où le POINTILLÉ, qui est la convention du calque
 * pour « lu, mais non parcouru », et l'ARC, qui l'écarte du trait droit d'un tir.
 *
 * `progress` va de 0 (à l'instant du saut) à 1 (à l'effacement).
 */
export function drawRiftLink(
  ctx: CanvasRenderingContext2D,
  from: XY,
  to: XY,
  progress: number,
  style: ShapeStyle,
  ink: RiftInk,
): void {
  const p = Math.min(1, Math.max(0, progress))
  const reste = 1 - p
  if (reste <= 0) return
  const dx = to.x - from.x
  const dy = to.y - from.y
  // La courbure est portée par la NORMALE au segment : l'arc s'écarte toujours du même côté,
  // ce qui rend deux liens successifs comparables au lieu de partir chacun au hasard.
  const cx = (from.x + to.x) / 2 - dy * RIFT_LINK_BOW
  const cy = (from.y + to.y) / 2 + dx * RIFT_LINK_BOW
  ctx.save()
  ctx.lineCap = 'round'
  ctx.setLineDash([3 * style.k, 3 * style.k])
  ctx.lineWidth = RIFT_LINK_WIDTH * style.k
  ctx.globalAlpha = RIFT_LINK_ALPHA * reste
  ctx.strokeStyle = ink.core
  ctx.beginPath()
  ctx.moveTo(from.x, from.y)
  ctx.quadraticCurveTo(cx, cy, to.x, to.y)
  ctx.stroke()
  ctx.setLineDash([])
  ctx.restore()
  // Les deux bouts : une faille au départ, une à l'arrivée. Elles rétrécissent avec
  // l'effacement — c'est le seul mouvement de tout le calque, et il est porté par un
  // ÉVÉNEMENT DATÉ, pas par une durée inventée.
  // MOUVEMENT RÉDUIT : les bouts gardent leur taille. Ce qui reste — les deux failles et
  // l'arc qui les relie — dit tout ce que l'effet a à dire ; seul le rétrécissement part.
  const facteur = style.reducedMotion ? 1 : RIFT_LINK_END_MIN + (1 - RIFT_LINK_END_MIN) * reste
  const bout: ShapeStyle = { ...style, k: style.k * facteur }
  ctx.save()
  ctx.globalAlpha = reste
  drawRift(ctx, from, bout, ink)
  drawRift(ctx, to, bout, ink)
  ctx.restore()
}

/**
 * drawTeleportLinks — les liens des passages ENCORE VISIBLES à cette frame.
 *
 * La fenêtre est comptée en TEMPS DE MATCH (`frameMs`), pas en nombre d'images : à 60 images
 * par seconde comme à 10, l'effet dure les mêmes 600 ms. C'est la règle déjà tenue par le ping
 * du capteur et la fin de vol des grenades.
 *
 * PAS DE BASCULE D'INTERFACE dédiée : le lien n'est pas un objet posé de plus qu'on afficherait
 * ou non, c'est ce que FAIT une faille déjà affichée. Il suit donc la même porte qu'elle.
 */
export function drawTeleportLinks(
  ctx: CanvasRenderingContext2D,
  teleports: readonly RiftTeleport[],
  view: PlacementView,
  time: ShapeStyle & { frame: number; frameMs: number },
  ink: RiftInk,
): void {
  for (const t of teleports) {
    const ageMs = (time.frame - t.frame) * time.frameMs
    if (ageMs < 0 || ageMs > RIFT_LINK_MS) continue
    drawRiftLink(
      ctx,
      project({ x: t.from.x, y: t.from.y }, view),
      project({ x: t.to.x, y: t.to.y }, view),
      ageMs / RIFT_LINK_MS,
      time,
      ink,
    )
  }
}
