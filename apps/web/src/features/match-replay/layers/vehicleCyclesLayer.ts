/**
 * vehicleCyclesLayer.ts — LE MARQUEUR DE RÉAPPARITION D'UN EMPLACEMENT DE VÉHICULE.
 *
 * LA SOURCE est `vehicleCycles` (schéma 63) : un emplacement de naissance par AMAS, publié
 * SEULEMENT quand la récurrence est MESURÉE (au moins deux écarts, écart-type sous 20 % de la
 * médiane — le juge de `PadCycle`). Un emplacement dont le cycle n'est pas établi n'est pas dans
 * la liste : il n'y a rien à en dire, et rien ne se dessine.
 *
 * CE QUE LE MARQUEUR RÉPOND, ET C'EST LA MÊME QUESTION QUE CELLE DES SOCLES D'ARME : « le
 * Warthog n'est plus là — dans combien de temps revient-il ». Le calque des véhicules dessine les
 * véhicules VIVANTS ; quand la vie née ici s'achève, le lieu disparaissait complètement de
 * l'écran, alors que le document sait quand il se rechargera.
 *
 * DISCRET, ET C'EST UNE CONTRAINTE, PAS UN GOÛT : une carte du rejeu porte déjà les pions, les
 * socles, les poses, les zones. Le marqueur est donc un LOSANGE — le vocabulaire du dépôt pour
 * « un objet de la carte, pas un joueur » (`traceDiamond`, partagée avec les socles et le repli
 * de châssis non résolu) — plus petit qu'un socle et à l'opacité du registre « absence prouvée ».
 *
 * IL NE PARAÎT QUE SUR UN EMPLACEMENT LIBRE. Tant qu'une vie née ici est en vigueur, rien ne se
 * dessine : le sprite du véhicule occupe déjà le lieu, et un marqueur de réapparition sous lui
 * annoncerait une attente qui n'a pas commencé (cf. `vehicleCycleTime.ts`, où vit toute la
 * lecture du temps).
 *
 * LE COMPTE À REBOURS EST CELUI DES SOCLES, AU PIXEL PRÈS (`drawCountdown`, exportée par
 * `weaponPadsLayer.ts`) : même police, même écart, même contour. Deux comptes à rebours d'aspect
 * différent sur la même carte se liraient comme deux natures d'information.
 *
 * CE QUE CE CALQUE NE DESSINE JAMAIS :
 *  - UN EMPLACEMENT SANS CYCLE ÉTABLI. Il n'est pas publié, et l'inventer depuis les naissances
 *    reviendrait à refaire côté client un juge que le serveur a déjà rendu.
 *  - UN COMPTE À REBOURS SANS SOURCE. Ni naissance suivante dans le film, ni fin DATÉE d'où
 *    partir : le losange reste seul. Un tiret suggérerait qu'on sait.
 *  - LA FAMILLE. Le losange ne porte ni vignette ni nom : la famille dominante, la médiane, les
 *    déciles et le nombre d'écarts se lisent au SURVOL (`ReplayVehicleCycleTip`), qui est
 *    l'endroit des lectures d'analyse — verdict du 2026-08-18 sur les socles, repris tel quel.
 *
 * Pas de React : géométrie pure + un `CanvasRenderingContext2D`, comme les calques voisins.
 * L'encre arrive de l'appelant, qui la tient des variables du thème.
 */
import { project, type PlacementView } from './placementShapes'
import { drawCountdown, traceDiamond } from './weaponPadsLayer'
import type { XY } from '../../../lib/replay/replayLogic'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleCycleReadingAt, type VehicleCycle } from '../model/vehicleCycleTime'

/**
 * Demi-diagonale du losange, en pixels d'ÉCRAN.
 *
 * ELLE EST DÉLIBÉRÉMENT SOUS CELLE D'UN SOCLE ORDINAIRE (3,2 × 1,25 = 4,0 px après compensation
 * d'aire) : un emplacement de véhicule est un repère de contexte, pas le sujet de la carte. La
 * compensation d'aire du losange est déjà comprise dans ce chiffre — il n'y a qu'une valeur à
 * régler, donc aucun facteur à tenir en phase avec celui des socles.
 */
const CYCLE_HALF_PX = 3.4

/**
 * Opacité du losange : celle du registre « absence prouvée » des socles (`PAD_ALPHA.empty.dot`).
 * Le lieu est vide par construction quand la marque paraît — lui donner l'opacité d'une présence
 * ferait lire un véhicule là où il n'y en a pas.
 */
const CYCLE_ALPHA = 0.35

/** Marge de confort autour du losange, en pixels d'écran : on vise près, pas au pixel. */
const CYCLE_HOVER_MARGIN_PX = 4

/** Ce que le calque a besoin de savoir de l'instant courant (même contrat que `PadTime`). */
export interface VehicleCycleTimeView {
  frame: number
  /** Durée RÉELLE d'une image (`frameToMs(1, doc)`) : le compte à rebours bat en temps réel. */
  frameMs: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
}

/** Ce que le calque emprunte au thème. Aucun token ici : l'appelant les a résolus. */
export interface VehicleCycleStyle {
  /** Encre neutre : un emplacement est un objet du terrain, il n'a pas de camp. */
  ink: string
  /** Les deux encres du compte à rebours — rempli à l'une, cerné de l'autre. */
  fill: string
  outline: string
  /** Le compte à rebours déjà localisé, dans son écriture COMPACTE (celle de la carte). */
  countdownLabel: (seconds: number) => string
}

/** Demi-diagonale d'écran du losange d'un emplacement, à la densité courante. */
function cycleHalfPx(k: number): number {
  return CYCLE_HALF_PX * k
}

/**
 * vehicleCycleIndexAt — le RANG de l'emplacement sous un point du canvas, ou -1.
 *
 * LE RANG ET NON L'OBJET, parce que le survol a besoin des VIES de l'emplacement autant que de
 * l'emplacement : elles sont appariées une fois par document, dans un tableau parallèle (cf.
 * `useReplayVehicles`), et les retrouver par recherche reviendrait à refaire l'appariement.
 *
 * LA ZONE EST UN DISQUE CENTRÉ SUR LE LOSANGE, et elle ne dépend PAS de l'occupation : une cible
 * qui apparaîtrait et disparaîtrait avec l'image serait impossible à viser. C'est l'infobulle qui
 * dit ce qu'il y a à cet instant — y compris « un véhicule s'y trouve ».
 *
 * Le plus proche l'emporte quand deux emplacements se recouvrent : deux familles peuvent naître
 * à quelques mètres l'une de l'autre.
 */
export function vehicleCycleIndexAt(
  cycles: readonly VehicleCycle[],
  view: PlacementView,
  k: number,
  at: XY,
): number {
  const reach = cycleHalfPx(k) + CYCLE_HOVER_MARGIN_PX * k
  let best = -1
  let bestD2 = reach * reach
  for (let i = 0; i < cycles.length; i++) {
    const c = project({ x: cycles[i].x, y: cycles[i].y }, view)
    const d2 = (at.x - c.x) ** 2 + (at.y - c.y) ** 2
    if (d2 <= bestD2) {
      best = i
      bestD2 = d2
    }
  }
  return best
}

/**
 * drawVehicleCyclesLayer trace les emplacements LIBRES à l'image courante.
 *
 * `livesOf` rend les vies nées à l'emplacement de rang `i` — appariées UNE FOIS par document par
 * l'appelant. Ce calque ne les cherche pas : l'appariement est un balayage de toutes les vies par
 * emplacement, bien trop cher à refaire soixante fois par seconde.
 */
export function drawVehicleCyclesLayer(
  ctx: CanvasRenderingContext2D,
  cycles: readonly VehicleCycle[],
  livesOf: (index: number) => readonly ReplayVehicleTrackReady[],
  view: PlacementView,
  time: VehicleCycleTimeView,
  style: VehicleCycleStyle,
): void {
  if (cycles.length === 0 || view.width === 0) return
  const half = cycleHalfPx(time.k)
  ctx.save()
  for (let i = 0; i < cycles.length; i++) {
    const cycle = cycles[i]
    const lu = vehicleCycleReadingAt(cycle, livesOf(i), time.frame, time.frameMs)
    // UN EMPLACEMENT OCCUPÉ NE PORTE AUCUNE MARQUE : le véhicule y est déjà dessiné.
    if (lu.occupant) continue
    const c = project({ x: cycle.x, y: cycle.y }, view)
    ctx.globalAlpha = CYCLE_ALPHA
    ctx.fillStyle = style.ink
    traceDiamond(ctx, c, half)
    ctx.fill()
    if (lu.respawn === null) continue
    drawCountdown(ctx, c, c.y - half, style.countdownLabel(lu.respawn.seconds), {
      fill: style.fill,
      outline: style.outline,
      k: time.k,
    })
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
