/**
 * reliefLogic — LES DÉCISIONS PURES du fond Relief et de la coupe par étage dans le rejeu : quels fonds
 * proposer, quelles pastilles d'étage dessiner, quelle demande de rendu poser, quelle image devient le fond.
 *
 * Aucun React, aucun canvas : `useReplayRelief` les branche sur les requêtes et le calculateur.
 *
 * LA COUPE NE MASQUE AUCUN JOUEUR dans le rejeu (maquette D.1, choix 9) : un joueur présent reste
 * toujours affiché, quelle que soit la coupe. Seul le FOND (et ses traits) suit la coupe ; rien ici ne
 * filtre une position. Seule « Occupation du terrain » masque les positions hors de l'étage choisi.
 */
import { floorRanks, hasNoMesh, type FloorRank, type ReliefData, type ReliefRenderRequest } from '@/lib/replay/relief'
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'

import { floorInRange, type FloorRange } from '../layers/floorRings'
import type { ReplayMapBackgroundLayer } from './useReplayView'

/**
 * backgroundOptions : les fonds que la commande propose, dans l'ordre Plan, Rendu, Relief (décision
 * D.1 choix 2). Le rendu est toujours là ; la commande n'existe que s'il y a un vrai choix (au moins
 * deux fonds) : sans plan ni relief utilisable, le rendu est servi sans rien proposer.
 */
export function backgroundOptions(planAvailable: boolean, reliefUsable: boolean): readonly MapBackgroundVariant[] {
  const options: MapBackgroundVariant[] = []
  if (planAvailable) options.push('plan')
  options.push('rendu')
  if (reliefUsable) options.push('relief')
  return options.length >= 2 ? options : []
}

/** Une pastille d'étage : son rang (0 = « Bas »), sa hauteur de coupe et le nombre d'anneaux d'un pion à cet étage. */
export interface FloorChip extends FloorRank {
  rings: number
}

/**
 * floorChips : une pastille par étage détecté. `rings` est le NOMBRE D'ANNEAUX que porte un pion posé à
 * la hauteur de l'étage (`floorInRange`, le langage d'étage des marqueurs) : l'infobulle fait ainsi le lien
 * entre la commande et ce que l'œil compte sur la carte.
 */
export function floorChips(levels: readonly number[], zRange: FloorRange): FloorChip[] {
  return floorRanks(levels).map((r) => ({ ...r, rings: floorInRange(r.z, zRange) }))
}

/** La commande d'étage telle que le panneau la montre. */
export type FloorState =
  /** Rien à proposer : pas de relief, relief encore en lecture ou illisible. */
  | { state: 'none' }
  /** Carte sans maillage de navigation : un bloc d'explication, aucune commande (D-7). */
  | { state: 'noMesh' }
  /** Des étages : les pastilles et les bornes du curseur. */
  | { state: 'ready'; chips: readonly FloorChip[]; min: number; max: number }

/** floorStateOf : l'état de la commande d'étage pour les données lues (`null` = pas lues, ou relief inutilisable). */
export function floorStateOf(data: ReliefData | null, zRange: FloorRange): FloorState {
  if (!data) return { state: 'none' }
  if (hasNoMesh(data)) return { state: 'noMesh' }
  const { levels, min, max } = data.floors
  if (levels.length === 0 || min === null || max === null) return { state: 'noMesh' }
  return { state: 'ready', chips: floorChips(levels, zRange), min, max }
}

/** La pastille est-elle retenue ? Seule la hauteur exacte de l'étage compte : un curseur libre n'en retient aucune. */
export function isChipSelected(cut: number | null, chip: FloorChip): boolean {
  return cut !== null && Math.abs(cut - chip.z) < 1e-6
}

/** Ce qui décide de la demande de rendu. */
export interface ReliefRequestInput {
  /** Une commande se sert du relief (fond Relief, traits ou coupe) et ses fichiers sont là. */
  needed: boolean
  turns: QuarterTurns
  cut: number | null
  /** La base de teinte quand le fond posé est le Relief ; `null` sinon. */
  tintBase: string | null
  lines: boolean
}

/**
 * reliefRequestOf : la demande de rendu. Le fond Relief veut l'image teintée ; un fond Plan ou Rendu ne
 * veut que les MASQUES de la coupe (sans coupe, il n'y en a pas). L'orientation ne sert qu'à la direction de la
 * lumière : elle ne compte que pour l'image teintée, un rendu de masques ou de traits n'en dépend pas.
 */
export function reliefRequestOf(input: ReliefRequestInput): ReliefRenderRequest | null {
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

/** Les images que le relief a produites pour la frame courante. */
export interface ReliefImages {
  /** Le relief teinté ; `null` tant qu'il n'est pas rendu, ou quand le fond posé n'est pas le Relief. */
  relief: CanvasImageSource | null
  /** Le fond Plan ou Rendu, modulé par la coupe ; `null` sans coupe ou sans masque. */
  masked: CanvasImageSource | null
}

/**
 * composeReliefBackground rend le fond à poser. Le relief teinté quand c'est lui qui est choisi ET
 * rendu ; sinon le fond de base, modulé par la coupe quand elle est active. Tant que le relief se charge,
 * le fond de base reste posé (le plan : un repli sans saut), et c'est lui qui reste s'il échoue.
 */
export function composeReliefBackground(
  base: ReplayMapBackgroundLayer | null,
  selected: MapBackgroundVariant,
  images: ReliefImages,
): ReplayMapBackgroundLayer | null {
  if (!base) return null
  if (selected === 'relief' && images.relief) return { ...base, image: images.relief, variante: 'relief' }
  if (selected !== 'relief' && images.masked) return { ...base, image: images.masked }
  return base
}
