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
 * UNE PRISE SE DESSINE SELON LA NATURE DE CE QUI EST PRIS (`PickupNature`) : une flèche qui monte du sol
 * pour une arme de râtelier, la même flèche et une étoile pour une arme spéciale, une caisse pour un
 * équipement, la même caisse et une étoile pour un équipement qui change la partie. Aucune silhouette
 * d'arme : la ligne dit le nom, le tracé dit la nature. Le menu de filtre prend la flèche pour toute la
 * famille. L'étoile (un plus de cinq unités en haut à droite) laisse au moins une unité de grille libre
 * entre son trait et celui de la flèche ou de la caisse, épaisseur du trait comprise.
 */
import type { PickupNature } from './feedEventLines'
import type { FeedFilterId } from './feedFilterLogic'
import type { ObjectiveFamily } from './objectiveFamilies'

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
  | 'pickupEquipmentPower'
  | 'vehicle'
  | 'medal'
  | 'assist'
  | 'round'
  | 'reply'

/** La flèche qui monte du sol : la pointe en haut, la tige, le sol. */
const PICKUP_ARROW = 'M12 16V4M7 9l5-5 5 5M4 20h16'
/** La caisse en perspective : le contour à six sommets, puis les trois arêtes qui se rejoignent au centre. */
const PICKUP_CRATE = 'M10.5 6l7 3.5v8l-7 3.5-7-3.5v-8ZM3.5 9.5l7 3.5 7-3.5M10.5 13v8'
/** L'étoile de ce qui compte : un plus, en haut à droite. */
const PICKUP_STAR = 'M20 1.5v5M17.5 4h5'

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
  pickup: PICKUP_ARROW,
  pickupPower: `${PICKUP_ARROW}${PICKUP_STAR}`,
  pickupEquipment: PICKUP_CRATE,
  pickupEquipmentPower: `${PICKUP_CRATE}${PICKUP_STAR}`,
  vehicle: 'M4 15l2-6h12l2 6v3H4ZM7 18v2M17 18v2',
  medal: 'M12 15a6 6 0 1 0 0-12 6 6 0 0 0 0 12ZM8.5 14 7 21l5-3 5 3-1.5-7',
  assist: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM5 21a7 7 0 0 1 14 0',
  round: 'M4 4h16v16H4ZM4 12h16M12 4v16',
  reply: 'M4 5h16v11H10l-4 4v-4H4Z',
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
export const PICKUP_NATURE_GLYPH: Readonly<Record<PickupNature, FeedGlyphKey>> = {
  classic: 'pickup',
  power: 'pickupPower',
  powerup: 'pickupEquipment',
  powerupGameChanger: 'pickupEquipmentPower',
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
  replies: 'reply',
}
