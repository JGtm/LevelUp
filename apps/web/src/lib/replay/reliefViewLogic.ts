/**
 * reliefViewLogic — LES DÉCISIONS PURES du fond Relief et de la coupe par étage des trois vues 2D : le rejeu,
 * l'onglet Tactique et « Occupation du terrain ».
 *
 * Aucun React, aucun canvas : `useReliefView` les branche sur les requêtes et le calculateur. Les étages ne sont que des
 * RANGS (« Bas », « Étage 1 »…, D-14) ; le rejeu y ajoute, dans sa feature, le langage des anneaux des marqueurs
 * (`features/match-replay/hooks/reliefLogic.ts`).
 *
 * CE QUE LA COUPE FILTRE (conception §7, liste fermée) :
 *   - les zones nommées, par leur tranche de hauteur (`zonesInCut`) ;
 *   - les positions d'« Occupation du terrain », par le sol qui les porte (`splitPositionsByCut`) ;
 *   tout autre calque reste entier.
 */
import type { MapBackgroundVariant } from './mapBackgroundVariant'
import {
  bearingFloor,
  CUT_BAND_U,
  CUT_MIN_WEIGHT,
  cutWeight,
  floorRanks,
  hasNoMesh,
  type FloorMesh,
  type FloorRank,
  type ReliefData,
  type ReliefRenderRequest,
} from './relief'
import type { QuarterTurns } from './orientation'

/**
 * viewBackgroundOptions : les fonds que la commande propose, dans l'ordre Plan, Rendu, Relief. Le rendu est
 * toujours là ; la commande n'existe que s'il y a un vrai choix (au moins deux fonds).
 */
export function viewBackgroundOptions(planAvailable: boolean, reliefUsable: boolean): readonly MapBackgroundVariant[] {
  const options: MapBackgroundVariant[] = []
  if (planAvailable) options.push('plan')
  options.push('rendu')
  if (reliefUsable) options.push('relief')
  return options.length >= 2 ? options : []
}

/** La commande d'étage telle que la vue la montre. */
export type ViewFloorState =
  /** Rien à proposer : relief pas encore lu, ou illisible. */
  | { state: 'none' }
  /** Carte sans maillage de navigation : aucune commande d'étage (D-7). */
  | { state: 'noMesh' }
  /** Des étages : les pastilles (par rang) et les bornes du curseur. */
  | { state: 'ready'; chips: readonly FloorRank[]; min: number; max: number }

/** viewFloorState : l'état de la commande d'étage pour les données lues (`null` = pas lues ou inutilisables). */
export function viewFloorState(data: ReliefData | null): ViewFloorState {
  if (!data) return { state: 'none' }
  if (hasNoMesh(data)) return { state: 'noMesh' }
  const { levels, min, max } = data.floors
  if (levels.length === 0 || min === null || max === null) return { state: 'noMesh' }
  return { state: 'ready', chips: floorRanks(levels), min, max }
}

/** La pastille est-elle retenue ? Seule la hauteur exacte de l'étage compte : un curseur libre n'en retient aucune. */
export function isCutAtLevel(cut: number | null, z: number): boolean {
  return cut !== null && Math.abs(cut - z) < 1e-6
}

/** L'étage le plus proche de `cut`, pour nommer la valeur du curseur sans unité ; `null` sans étage. */
export function nearestChip<T extends FloorRank>(chips: readonly T[], cut: number): T | null {
  let best: T | null = null
  for (const chip of chips) {
    if (best === null || Math.abs(chip.z - cut) < Math.abs(best.z - cut)) best = chip
  }
  return best
}

/** Ce qui décide de la demande de rendu. */
export interface ReliefViewRequestInput {
  /** Une commande se sert du relief (fond Relief, traits ou coupe) et ses fichiers sont là. */
  needed: boolean
  turns: QuarterTurns
  cut: number | null
  /** La base de teinte quand le fond voulu est le Relief ; `null` sinon. */
  tintBase: string | null
  lines: boolean
}

/**
 * reliefViewRequest : la demande de rendu. Le fond Relief veut l'image teintée ; un fond Plan ou Rendu ne veut
 * que les MASQUES de la coupe (sans coupe, il n'y en a pas). L'orientation ne sert qu'à la direction de la
 * lumière : elle ne compte que pour l'image teintée.
 */
export function reliefViewRequest(input: ReliefViewRequestInput): ReliefRenderRequest | null {
  if (!input.needed) return null
  const tinted = input.tintBase !== null
  return {
    turns: tinted ? input.turns : 0,
    cut: input.cut,
    base: input.tintBase,
    lines: input.lines,
    masks: input.cut !== null && !tinted,
  }
}

/** Une tranche de hauteur : celle d'un volume nommé (zone). */
export interface HeightSpan {
  zBottom: number
  zTop: number
}

/**
 * zoneInCut : la zone reste quand sa tranche `[zBottom ; zTop]` croise la bande de la coupe
 * (`cut ± CUT_BAND_U`). Sans coupe (`null`), toute zone reste. Une tranche dégénérée (bornes inversées ou
 * non finies) est jugée sur sa borne basse : mieux vaut garder une zone que la perdre sur une donnée douteuse.
 */
export function zoneInCut(zone: HeightSpan, cut: number | null): boolean {
  if (cut === null) return true
  const { zBottom, zTop } = zone
  if (!Number.isFinite(zBottom) || !Number.isFinite(zTop)) return true
  const lo = Math.min(zBottom, zTop)
  const hi = Math.max(zBottom, zTop)
  return lo <= cut + CUT_BAND_U && hi >= cut - CUT_BAND_U
}

/** zonesInCut : les zones qui restent sous la coupe ; la MÊME liste (identité) sans coupe, pour ne rien repeindre. */
export function zonesInCut<T extends HeightSpan>(zones: readonly T[], cut: number | null): readonly T[] {
  return cut === null ? zones : zones.filter((z) => zoneInCut(z, cut))
}

/** Une position servie : sa place monde et sa hauteur (pieds du joueur). */
export interface HeightPosition {
  x: number
  y: number
  z: number
}

/**
 * positionInCut : une position reste quand son SOL PORTEUR pèse plus que `CUT_MIN_WEIGHT` dans la coupe
 * (un joueur en l'air garde l'étage de ses pieds) ; sans sol sous elle, on juge sur sa hauteur.
 */
export function positionInCut(p: HeightPosition, mesh: FloorMesh, cut: number): boolean {
  const floor = bearingFloor(mesh, p.x, p.y, p.z)
  return cutWeight(floor ?? p.z, cut) > CUT_MIN_WEIGHT
}

/** Ce que la coupe fait des positions : celles qui restent, et le compte de celles qu'elle masque. */
export interface PositionsByCut<T> {
  kept: readonly T[]
  hidden: number
}

/** splitPositionsByCut : sans coupe, tout reste (même liste) ; sinon les positions hors de l'étage sont masquées. */
export function splitPositionsByCut<T extends HeightPosition>(
  positions: readonly T[],
  mesh: FloorMesh | null,
  cut: number | null,
): PositionsByCut<T> {
  if (cut === null || mesh === null) return { kept: positions, hidden: 0 }
  const kept = positions.filter((p) => positionInCut(p, mesh, cut))
  return { kept, hidden: positions.length - kept.length }
}
