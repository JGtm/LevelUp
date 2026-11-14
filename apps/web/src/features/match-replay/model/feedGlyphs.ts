/**
 * feedGlyphs.ts — LES PICTOGRAMMES DU FIL ET DE SON MENU : des tracés vectoriels sur une grille
 * de 24, au trait, à l'encre courante (le parent choisit la couleur, jamais le tracé). Données
 * seulement ; le composant qui les dessine est `ui/ReplayEventGlyph.tsx`.
 *
 * LES SEPT PREMIERS TRACÉS SONT CEUX DE LA MAQUETTE du fil (éliminations, drapeau, arme,
 * véhicule, liste, silhouette, médaille) ; les autres familles d'objectif (zone, colline, crâne,
 * couronne, bombe) suivent le même trait. Aucune vignette d'atlas dont l'index bouge par saison :
 * un trait vectoriel ne dépend d'aucun asset.
 *
 * UNE PRISE SE DESSINE SELON LA NATURE DE CE QUI EST PRIS, la même que celle qui teinte les socles
 * de la carte (`PadFamily`) : arme de râtelier (l'arme seule), arme spéciale (l'arme et une étoile),
 * équipement (un bouclier). Le menu de filtre garde l'arme seule pour toute la famille.
 */
import type { FeedFilterId } from './feedFilterLogic'
import type { ObjectiveFamily } from './objectiveFamilies'
import type { PadFamily } from './weaponPadFamilies'

export type FeedGlyphKey =
  | 'all'
  | 'kill'
  | 'flag'
  | 'zone'
  | 'hill'
  | 'skull'
  | 'vip'
  | 'bomb'
  | 'pickup'
  | 'pickupPower'
  | 'pickupEquipment'
  | 'vehicle'
  | 'medal'
  | 'assist'

/** Le tracé (attribut `d`) de chaque pictogramme. */
export const FEED_GLYPH_PATH: Readonly<Record<FeedGlyphKey, string>> = {
  all: 'M4 6h16M4 12h16M4 18h16',
  kill: 'M12 3v5M12 16v5M3 12h5M16 12h5M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Z',
  flag: 'M5 21V4M5 4h12l-3 4 3 4H5',
  zone: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18ZM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z',
  hill: 'M3 20 10 6l4 8 3-4 4 10Z',
  skull: 'M12 3a8 8 0 0 0-5 14v3h10v-3a8 8 0 0 0-5-14ZM9.5 12h.01M14.5 12h.01',
  vip: 'M3 18 5 7l4.5 5L12 5l2.5 7L19 7l2 11Z',
  bomb: 'M11 9a6 6 0 1 0 0 12 6 6 0 0 0 0-12ZM15 8l3-3M18 5l1-2',
  pickup: 'M3 9h13l2 2h3v3h-9l-2 4H7l1-4H3Z',
  pickupPower: 'M2 11h12l2 2h3v3h-8l-2 4H6l1-4H2ZM19 2v6M16 5h6',
  pickupEquipment: 'M12 3l7 3v5c0 5-3 8.5-7 10-4-1.5-7-5-7-10V6Z',
  vehicle: 'M4 15l2-6h12l2 6v3H4ZM7 18v2M17 18v2',
  medal: 'M12 15a6 6 0 1 0 0-12 6 6 0 0 0 0 12ZM8.5 14 7 21l5-3 5 3-1.5-7',
  assist: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM5 21a7 7 0 0 1 14 0',
}

/** Le pictogramme de la ligne d'une action d'objectif : celui de sa famille. */
export const OBJECTIVE_FAMILY_GLYPH: Readonly<Record<ObjectiveFamily, FeedGlyphKey>> = {
  flag: 'flag',
  zone: 'zone',
  hill: 'hill',
  skull: 'skull',
  vip: 'vip',
  bomb: 'bomb',
}

/** Le pictogramme d'une ligne de prise, selon la nature de ce qui est pris. */
export const PICKUP_NATURE_GLYPH: Readonly<Record<PadFamily, FeedGlyphKey>> = {
  classic: 'pickup',
  power: 'pickupPower',
  powerup: 'pickupEquipment',
}

/** Le pictogramme de chaque entrée du menu de filtre. */
export const FEED_FILTER_GLYPH: Readonly<Record<FeedFilterId, FeedGlyphKey>> = {
  tout: 'all',
  kills: 'kill',
  objectives: 'flag',
  pickups: 'pickup',
  vehicles: 'vehicle',
  medals: 'medal',
  assists: 'assist',
}
