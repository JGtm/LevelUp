/**
 * settingsAvailability — CE QUE LE MATCH, LE CADRE ET LE FOND PROPOSENT, réglage par réglage, déduit de la table
 * (`settingsTable`). C'est la règle du dépôt « pas de commande quand il n'y a rien à commander » : un réglage
 * sans objet sur ce match n'est pas affiché, et une option dépendante se lit sous sa mère.
 *
 * DEUX QUESTIONS DISTINCTES, deux fonctions :
 *  - `isProposed` : le match porte-t-il de quoi commander ce réglage ? Elle ne dépend que du film, du cadre et du
 *    fond posé, jamais d'une valeur. Les préréglages, l'état « modifié » et l'interrupteur maître ne regardent
 *    QUE les réglages proposés : un réglage sans objet ne rend pas une vue « modifiée » ;
 *  - `isShown` : la ligne est-elle affichée maintenant ? Elle ajoute la dépendance à la mère (un filtre ne se
 *    montre pas quand son calque est éteint) et les affinages des vies (qui, quand : avec une trace allumée).
 *
 * LOGIQUE PURE : aucun React, aucun stockage.
 */
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import type { FilmLayerAvailability } from './filmLayerAvailability'
import {
  defOf, defsOfRubric, SETTING_DEFS,
  type RubricId, type SettingDef, type SettingId, type SettingValue, type SettingValues,
} from './settingsTable'

/** L'état de la coupe d'étage tel que le relief le publie : rien, carte sans maillage, ou commande prête. */
export type FloorAvailability = 'none' | 'noMesh' | 'ready'

/** Tout ce que le match, le cadre et le fond proposent. */
export interface SettingsContext {
  film: FilmLayerAvailability
  /** Le cadre peut poser le fil des éliminations sur le plan (Cinéma, Plein cadre). */
  planFeedAvailable: boolean
  /** Le cadre peut poser la barre de lecture sur le plan (Plein cadre). */
  thinBarAvailable: boolean
  /** Le fond RÉELLEMENT posé : le plan porte déjà les murs. */
  drawn: MapBackgroundVariant
  /** Les fonds que le match offre ; vide, il n'y a pas de choix. */
  backgroundOptions: readonly MapBackgroundVariant[]
  reliefLinesAvailable: boolean
  floor: FloorAvailability
  orientationAvailable: boolean
  soundAvailable: boolean
  /** Les lectures de la carte de chaleur : « éliminations » n'existe que si une mort est localisable. */
  heatKillsAvailable: boolean
  /** Les filtres du fil que ce match propose (Objectifs seulement avec une ligne d'objectif). */
  feedFilters: readonly string[]
}

/** Le match porte-t-il de quoi commander ce réglage ? */
export function isProposed(id: SettingId, c: SettingsContext): boolean {
  const f = c.film
  switch (id) {
    case 'spawns': return f.lives.spawns
    case 'deaths': return f.lives.deaths
    case 'killers': return f.lives.killers
    case 'livesWho':
    case 'livesWhen': return f.lives.spawns || f.lives.deaths || f.lives.killers
    case 'forecast':
    case 'forecastDraw': return f.respawnForecast
    case 'bg': return c.backgroundOptions.length > 0
    case 'reliefLines': return c.reliefLinesAvailable
    // Le plan porte déjà les murs : poser leur dessin par-dessus n'a pas de sens, le réglage n'est pas proposé.
    case 'wallPlan': return f.wallPlan && c.drawn !== 'plan'
    case 'floor': return c.floor !== 'none'
    case 'floorFollow': return c.floor === 'ready'
    case 'orientation': return c.orientationAvailable
    case 'zones': return f.zones
    case 'heatMode': return c.heatKillsAvailable
    case 'weaponPads': return f.weaponPads
    case 'groundWeapons':
    case 'groundSpecial':
    case 'groundStarting': return f.groundWeapons
    case 'vehicles': return f.vehicles
    case 'placements': return f.placements.drawable
    case 'placeDropped': return f.placements.drawable && f.placements.dropped
    case 'placeUnnamed': return f.placements.drawable && f.placements.unnamed
    case 'objectives': return f.modeObjectives
    case 'flags': return f.flagCarries
    case 'vip': return f.vipCrown
    case 'skull': return f.skullCarrier
    case 'bomb': return f.bombCarrier
    case 'planFeed': return c.planFeedAvailable
    case 'thinBar': return c.thinBarAvailable
    case 'sound':
    case 'volume':
    case 'catWeapon':
    case 'catGrenade':
    case 'catMelee':
    case 'catEquip': return c.soundAvailable
    default: return true
  }
}

/** Les réglages proposés pour ce match, dans l'ordre de la table. */
export function proposedIds(c: SettingsContext): readonly SettingId[] {
  return SETTING_DEFS.filter((d) => isProposed(d.id, c)).map((d) => d.id)
}

/** La ligne est-elle affichée maintenant, selon l'état de sa mère et des traces de vies ? */
export function isShown(id: SettingId, c: SettingsContext, values: SettingValues): boolean {
  if (!isProposed(id, c)) return false
  const def = defOf(id)
  if (def.parent && def.hideWhenParentOff && values[def.parent] === false) return false
  if (id === 'livesWho' || id === 'livesWhen') {
    const l = c.film.lives
    return (values.spawns === true && l.spawns) || (values.deaths === true && l.deaths) || (values.killers === true && l.killers)
  }
  return true
}

/** Les lignes affichées d'une rubrique, dans l'ordre de la table. */
export function shownOfRubric(rubric: RubricId, c: SettingsContext, values: SettingValues): readonly SettingDef[] {
  return defsOfRubric(rubric).filter((d) => isShown(d.id, c, values))
}

/** Les options d'un choix pour CE match (« éliminations » de la chaleur seulement avec une mort localisable). */
export function optionsOf(id: SettingId, c: SettingsContext): readonly string[] {
  const options = defOf(id).options ?? []
  if (id === 'heatMode') return c.heatKillsAvailable ? options : options.filter((o) => o !== 'kills')
  if (id === 'livesWho') return c.film.lives.followedKnown ? options : options.filter((o) => o !== 'followed')
  if (id === 'bg') return c.backgroundOptions
  return options
}

/** L'état de l'interrupteur maître d'une rubrique : ses interrupteurs affichés, allumés sur total. */
export interface MasterState {
  on: number
  total: number
  state: 'all' | 'some' | 'none'
}

export function masterState(rubric: RubricId, c: SettingsContext, values: SettingValues): MasterState {
  const counted = shownOfRubric(rubric, c, values).filter((d) => d.master)
  const on = counted.filter((d) => values[d.id] === true).length
  const total = counted.length
  return { on, total, state: on === 0 ? 'none' : on === total ? 'all' : 'some' }
}

/** Ce que l'interrupteur maître écrit : chaque interrupteur proposé de la rubrique, à la valeur demandée. */
export function masterWrites(rubric: RubricId, c: SettingsContext, on: boolean): ReadonlyArray<readonly [SettingId, SettingValue]> {
  return defsOfRubric(rubric).filter((d) => d.master && isProposed(d.id, c)).map((d) => [d.id, on] as const)
}

/** L'état d'un réglage épinglé : proposé ou grisé, et pourquoi. */
export function pinStatus(id: SettingId, c: SettingsContext): { enabled: boolean; reason: 'frame' | 'match' | null } {
  if (isProposed(id, c)) return { enabled: true, reason: null }
  return { enabled: false, reason: id === 'planFeed' ? 'frame' : 'match' }
}

/**
 * LE MODE DE JEU, tel que le film le laisse lire : l'enjeu que le match porte. C'est la clé à laquelle une vue se
 * rattache (« pour les matchs à drapeau »), lue des capacités du film et non d'un nom de mode : elle ne dépend ni du
 * titre ni de la langue. Le premier enjeu porté l'emporte (un mode à drapeau porte aussi des zones).
 */
export const MATCH_MODE_KEYS = ['flag', 'skull', 'vip', 'bomb', 'zones', 'open'] as const
export type MatchModeKey = (typeof MATCH_MODE_KEYS)[number]

export function matchModeKey(f: FilmLayerAvailability): MatchModeKey {
  if (f.flagCarries) return 'flag'
  if (f.skullCarrier) return 'skull'
  if (f.vipCrown) return 'vip'
  if (f.bombCarrier) return 'bomb'
  if (f.modeObjectives) return 'zones'
  return 'open'
}
