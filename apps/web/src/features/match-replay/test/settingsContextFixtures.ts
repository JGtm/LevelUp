/**
 * settingsContextFixtures.ts — LE CONTEXTE DES RÉGLAGES, FABRIQUÉ POUR LES TESTS : ce que le match, le cadre et le
 * fond proposent. Un match de capture complet par défaut, que chaque test réduit. Rien ici n'asserte.
 */
import type { FilmLayerAvailability } from '../settings/filmLayerAvailability'
import type { SettingsContext } from '../settings/settingsAvailability'

/** Un film qui porte tout : zones, socles, armes au sol, poses, drapeaux, véhicules, vies, probabilité, plan des murs. */
export const FULL_FILM: FilmLayerAvailability = {
  zones: true,
  placements: { drawable: true, unnamed: true, dropped: true },
  weaponPads: true,
  groundWeapons: true,
  modeObjectives: true,
  flagCarries: true,
  vipCrown: false,
  skullCarrier: false,
  bombCarrier: false,
  vehicles: true,
  wallPlan: true,
  lives: { spawns: true, deaths: true, killers: true, followedKnown: true },
  respawnForecast: true,
}

/** Un Assassin : ni zone, ni socle, ni objectif ; la visée et la traînée restent. */
export const SLAYER_FILM: FilmLayerAvailability = {
  zones: false,
  placements: { drawable: false, unnamed: false, dropped: false },
  weaponPads: false,
  groundWeapons: false,
  modeObjectives: false,
  flagCarries: false,
  vipCrown: false,
  skullCarrier: false,
  bombCarrier: false,
  vehicles: false,
  wallPlan: false,
  lives: { spawns: false, deaths: false, killers: false, followedKnown: false },
  respawnForecast: false,
}

/** Le contexte d'un match de capture au Cinéma, sur le fond Rendu, avec un relief lisible et du son. */
export function makeSettingsContext(over: Partial<SettingsContext> = {}): SettingsContext {
  return {
    film: FULL_FILM,
    planFeedAvailable: true,
    thinBarAvailable: false,
    drawn: 'rendu',
    backgroundOptions: ['plan', 'rendu', 'relief'],
    reliefLinesAvailable: true,
    floor: 'ready',
    orientationAvailable: true,
    soundAvailable: true,
    heatKillsAvailable: true,
    feedFilters: ['tout', 'kills', 'objectives', 'pickups', 'vehicles', 'medals', 'assists'],
    ...over,
  }
}
