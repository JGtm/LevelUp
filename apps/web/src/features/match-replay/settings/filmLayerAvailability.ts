/**
 * filmLayerAvailability — CE QUE LE FILM PORTE, calque par calque, tel que le canvas le publie au
 * panneau de la vue (`FilmLayerAvailability`). Extrait de `ReplayCanvas.tsx` : le canvas est au plafond
 * de taille (`max-lines`), et chaque calque neuf y ajoutait une ligne à cette table.
 *
 * Chaque entrée est la réponse d'un hook de calque à « as-tu de quoi te peindre ? » ; aucune règle ne
 * se décide ici, ce module les range sous les clés que le panneau lit.
 */

/** Ce que le film porte, tel que le canvas le publie (les mêmes questions que le panneau). */
export interface FilmLayerAvailability {
  zones: boolean
  placements: { drawable: boolean; unnamed: boolean; dropped: boolean }
  weaponPads: boolean
  groundWeapons: boolean
  modeObjectives: boolean
  flagCarries: boolean
  vipCrown: boolean
  skullCarrier: boolean
  bombCarrier: boolean
  vehicles: boolean
  /**
   * Les traces des vies : chaque calque se propose dès qu'il a au moins une marque à poser.
   * `followedKnown` n'est pas un calque : il ne conditionne que l'option « Point de vue » de l'affinage.
   */
  lives: { spawns: boolean; deaths: boolean; killers: boolean; followedKnown: boolean }
  /**
   * La probabilité de réapparition : l'interrupteur se propose quand la réponse est prête (ou pas encore
   * connue alors que le calque est éteint).
   */
  respawnForecast: boolean
  /** Pourquoi aucune probabilité n'est servie, quand la réponse la refuse ; le panneau le dit à la place de l'interrupteur. */
  respawnForecastRefusal?: string | null
  /** Le fond retenu porte un plan des murs : il ne fait que conditionner l'affichage de sa bascule. */
  wallPlan: boolean
}

/** Un calque qui sait s'il a de quoi se peindre. */
interface Availability {
  available: boolean
}

/** Les réponses des hooks de calque, telles que le canvas les tient. */
export interface FilmLayerSources {
  /** Nombre de zones nommées de la carte. */
  zoneCount: number
  placements: { counts: { drawable: number; unnamed: number; dropped: number } }
  weaponPads: Availability
  groundWeapons: Availability
  /** Nombre d'objectifs statiques du mode joué. */
  objectiveCount: number
  flags: Availability
  vipCrown: Availability
  skullCarrier: Availability
  bombCarrier: Availability
  vehicles: Availability
  /** Le fond retenu porte un plan des murs. */
  hasWallPlan: boolean
  lives: FilmLayerAvailability['lives']
  /** La probabilité de réapparition : de quoi se peindre, et le motif du refus quand la réponse en donne un. */
  respawnForecast: Availability & { refusal?: string | null }
}

export function filmLayerAvailability(s: FilmLayerSources): FilmLayerAvailability {
  return {
    zones: s.zoneCount > 0,
    placements: {
      drawable: s.placements.counts.drawable > 0,
      unnamed: s.placements.counts.unnamed > 0,
      dropped: s.placements.counts.dropped > 0,
    },
    weaponPads: s.weaponPads.available,
    groundWeapons: s.groundWeapons.available,
    modeObjectives: s.objectiveCount > 0,
    flagCarries: s.flags.available,
    vipCrown: s.vipCrown.available,
    skullCarrier: s.skullCarrier.available,
    bombCarrier: s.bombCarrier.available,
    vehicles: s.vehicles.available,
    wallPlan: s.hasWallPlan,
    lives: s.lives,
    respawnForecast: s.respawnForecast.available,
    respawnForecastRefusal: s.respawnForecast.refusal ?? null,
  }
}
