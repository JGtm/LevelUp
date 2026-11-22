/**
 * reliefLogic — CE QUE LE REJEU AJOUTE à la logique commune du fond Relief (`lib/replay/reliefViewLogic`, `useReliefView`) :
 * le langage d'étage des marqueurs (le nombre d'anneaux de chaque pastille) et la pose du fond sur le fond de base du canvas.
 *
 * Aucun React, aucun canvas : `useReplayRelief` les branche sur la vue commune.
 *
 * LA COUPE NE MASQUE AUCUN JOUEUR dans le rejeu (maquette D.1, choix 9) : un joueur présent reste
 * toujours affiché, quelle que soit la coupe. Seul le FOND (et ses traits) suit la coupe ; rien ici ne
 * filtre une position. Seule « Occupation du terrain » masque les positions hors de l'étage choisi.
 */
import type { FloorRank } from '@/lib/replay/relief'
import type { ViewFloorState } from '@/lib/replay/reliefViewLogic'
import type { ReliefSurface } from '@/lib/replay/useReliefView'

import { floorInRange, type FloorRange } from '../layers/floorRings'
import type { ReplayMapBackgroundLayer } from './useReplayView'

/** Une pastille d'étage : son rang (0 = « Bas »), sa hauteur de coupe et le nombre d'anneaux d'un pion à cet étage. */
export interface FloorChip extends FloorRank {
  rings: number
}

/**
 * floorChips : une pastille par rang d'étage. `rings` est le NOMBRE D'ANNEAUX que porte un pion posé à
 * la hauteur de l'étage (`floorInRange`, le langage d'étage des marqueurs) : l'infobulle fait ainsi le lien
 * entre la commande et ce que l'œil compte sur la carte.
 */
export function floorChips(ranks: readonly FloorRank[], zRange: FloorRange): FloorChip[] {
  return ranks.map((r) => ({ ...r, rings: floorInRange(r.z, zRange) }))
}

/** La commande d'étage telle que le panneau « Vue » la montre : celle de la vue commune, ses pastilles portant les anneaux. */
export type FloorState =
  | Exclude<ViewFloorState, { state: 'ready' }>
  | { state: 'ready'; chips: readonly FloorChip[]; min: number; max: number }

/** floorStateWithRings : l'état de la vue commune, chaque pastille d'étage complétée de ses anneaux. */
export function floorStateWithRings(view: ViewFloorState, zRange: FloorRange): FloorState {
  if (view.state !== 'ready') return view
  return { ...view, chips: floorChips(view.chips, zRange) }
}

/**
 * composeReliefBackground rend le fond à poser. L'image que le relief pose (`surface`) prend la place de celle du fond de
 * base : le relief teinté quand c'est le Relief qui est retenu (`reliefShown`, la variante devient `relief`), sinon le fond
 * de base modulé par la coupe. Sans image du relief (en lecture, en échec, sans coupe), le fond de base reste posé : le plan,
 * un repli sans saut.
 */
export function composeReliefBackground(
  base: ReplayMapBackgroundLayer | null,
  surface: ReliefSurface | null,
  reliefShown: boolean,
): ReplayMapBackgroundLayer | null {
  if (!base || !surface) return base
  return reliefShown ? { ...base, image: surface.image, variante: 'relief' } : { ...base, image: surface.image }
}
