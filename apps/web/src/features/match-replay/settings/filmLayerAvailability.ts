/**
 * filmLayerAvailability — CE QUE LE FILM PORTE, calque par calque, tel que le canvas le publie au
 * panneau de la vue (`FilmLayerAvailability`). Extrait de `ReplayCanvas.tsx` : le canvas est au plafond
 * de taille (`max-lines`), et chaque calque neuf y ajoutait une ligne à cette table.
 *
 * Chaque entrée est la réponse d'un hook de calque à « as-tu de quoi te peindre ? » ; aucune règle ne
 * se décide ici, ce module les range sous les clés que le panneau lit.
 */
import type { FilmLayerAvailability } from './viewPresetLogic'

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
  respawnForecast: Availability
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
  }
}
