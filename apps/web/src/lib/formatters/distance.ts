/**
 * Distances du jeu : unités du jeu → mètres, À L'AFFICHAGE SEULEMENT.
 *
 * LA RÈGLE. Les positions du rejeu, de la base et de l'API sont en UNITÉS DU JEU (« wu » de
 * Slipspace). Une unité vaut 10 pieds, soit 3,048 m : une distance lue dans le film n'est jamais
 * écrite « m » telle quelle. Les champs d'API dont le nom finit par `_m` restent en unités (ils
 * ne sont pas renommés) ; seule exception, `rayon_m` et `rayons_radar_m`, qui sont de VRAIS
 * mètres (la portée du radar annoncée par le jeu, 18 m en arène, 24 m en Grande bataille).
 *
 * LE FOYER. Ce fichier est le seul à porter le facteur (garde-rail `distance.guard.test.ts`) ; son
 * jumeau Go est `internal/analysis/gameunits`, tenu à la même valeur par un test d'archlint.
 */
/** Mètres dans une unité du jeu (10 pieds). */
export const METERS_PER_GAME_UNIT = 3.048

/** Une distance en unités du jeu, en mètres. */
export function gameUnitsToMeters(units: number): number {
  return units * METERS_PER_GAME_UNIT
}

/** Une distance en vrais mètres, en unités du jeu (la portée du radar, comparée à des positions). */
export function metersToGameUnits(meters: number): number {
  return meters / METERS_PER_GAME_UNIT
}
