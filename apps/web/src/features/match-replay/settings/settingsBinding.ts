/**
 * settingsBinding — LE LIEN ENTRE LA TABLE DES RÉGLAGES ET LES ÉTATS QUI LES PORTENT : pour chaque réglage de la
 * table, sa valeur courante et le geste qui l'écrit. Les états et leur persistance restent là où ils vivent
 * (`useReplaySettings` : calques, effets, chaleur, couleur des points ; `useReplayRelief` : traits et étage ;
 * `useReplayOrientation` ; `useReplaySound` : son, volume, catégories ; le filtre du fil) : ce module ne
 * réinvente aucun réglage, il les range sous les identifiants de la table.
 *
 * UNE ÉCRITURE DE OUI/NON EST UNE BASCULE CONDITIONNELLE : les calques n'exposent que leur bascule, `set` ne la
 * déclenche que si la valeur voulue diffère de la valeur lue. Un geste qui écrit plusieurs réglages à la fois
 * (un préréglage, un interrupteur maître) n'écrit que des réglages distincts, jamais deux fois le même.
 *
 * FONCTION PURE de ses entrées : aucun état propre, rien n'est mémorisé ici.
 */
import { normalizeTurns } from '@/lib/replay/orientation'
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import type { ReplayRelief } from '../hooks/useReplayRelief'
import type { FeedFilterId } from '../model/feedFilterLogic'
import type { ForecastDraw } from '../model/respawnForecast.logic'
import { effectiveLivesWho, type LivesWhen, type LivesWho } from '../model/livesMarks'
import type { SoundCategory } from '../sound/replaySound'
import type { ReplaySound } from '../sound/useReplaySound'
import type { HeatmapMode, HeatmapSpan } from '../../../lib/replay/heatPaint'
import type { ReplayOrientation } from './useReplayOrientation'
import type { MarkerColorsMode, ReplaySettings } from './useReplaySettings'
import type { SettingId, SettingValue, SettingValues } from './settingsTable'

/** Tout ce que le lien lit et écrit. */
export interface BindingInput {
  settings: ReplaySettings
  relief: ReplayRelief
  orientation: ReplayOrientation
  sound: ReplaySound
  /** Le filtre du fil APPLIQUÉ à ce match (le mémorisé s'il est proposé, Tout sinon) et le geste qui le change. */
  feed: { value: FeedFilterId; choose: (id: FeedFilterId) => void }
  /** La lecture de la chaleur RÉELLEMENT servie (sans mort localisable, la présence), pas celle demandée. */
  heatMode: HeatmapMode
  /** Il y a un joueur suivi dans ce match : sans lui, « point de vue » se lit « tous ». */
  followedKnown: boolean
}

export interface SettingsBinding {
  values: SettingValues
  /** Écrit un réglage. Un réglage hors table n'existe pas ; une valeur étrangère à son réglage est ignorée. */
  set: (id: SettingId, value: SettingValue) => void
}

const SOUND_CATEGORY_OF: Readonly<Record<'catWeapon' | 'catGrenade' | 'catMelee' | 'catEquip', SoundCategory>> = {
  catWeapon: 'weapon',
  catGrenade: 'grenade',
  catMelee: 'melee',
  catEquip: 'equipment',
}

function valuesOf(i: BindingInput): SettingValues {
  const { settings: s, relief, orientation, sound, feed } = i
  return {
    aim: s.showAim,
    trail: s.showTrail,
    colors: s.markerColors,
    shotFx: s.showShotFx,
    killFx: s.showKillFx,
    spawns: s.showSpawns,
    deaths: s.showDeaths,
    killers: s.showKillers,
    livesWho: effectiveLivesWho(s.livesWho, i.followedKnown),
    livesWhen: s.livesWhen,
    forecast: s.showRespawnForecast,
    forecastDraw: s.respawnForecastDraw,
    bg: relief.selected,
    reliefLines: relief.lines.show,
    wallPlan: s.showWallPlan,
    floor: relief.floor.cut,
    floorFollow: relief.floor.follow.on,
    orientation: orientation.retained === null ? 'auto' : String(orientation.retained),
    zones: s.showZones,
    heatmap: s.showHeatmap,
    heatMode: i.heatMode,
    heatSpan: s.heatmapSpan,
    weaponPads: s.showWeaponPads,
    groundWeapons: s.showGroundWeapons,
    groundSpecial: s.showGroundWeaponsSpecialOnly,
    groundStarting: s.showGroundWeaponsStarting,
    vehicles: s.showVehicles,
    placements: s.showPlacements,
    placeDropped: s.showDroppedPlacements,
    placeUnnamed: s.showUnnamedPlacements,
    objectives: s.showModeObjectives,
    flags: s.showFlagCarries,
    vip: s.showVipCrown,
    skull: s.showSkullCarrier,
    bomb: s.showBombCarrier,
    feedFilter: feed.value,
    planFeed: s.showPlanFeed,
    sound: sound.on,
    volume: sound.volume,
    catWeapon: sound.categories.weapon,
    catGrenade: sound.categories.grenade,
    catMelee: sound.categories.melee,
    catEquip: sound.categories.equipment,
    thinBar: s.thinBarAtFullFrame,
  }
}

/** Les gestes qui n'exposent qu'une bascule : on ne la déclenche que si la valeur voulue diffère de la valeur lue. */
function flagSetters(i: BindingInput): Partial<Record<SettingId, () => void>> {
  const { settings: s, relief, sound } = i
  return {
    aim: s.toggleAim, trail: s.toggleTrail, shotFx: s.toggleShotFx, killFx: s.toggleKillFx,
    spawns: s.toggleSpawns, deaths: s.toggleDeaths, killers: s.toggleKillers, forecast: s.toggleRespawnForecast,
    reliefLines: relief.lines.onToggle, wallPlan: s.toggleWallPlan, floorFollow: relief.floor.follow.onToggle,
    zones: s.toggleZones, heatmap: s.toggleHeatmap, weaponPads: s.toggleWeaponPads,
    groundWeapons: s.toggleGroundWeapons, groundSpecial: s.toggleGroundWeaponsSpecialOnly,
    groundStarting: s.toggleGroundWeaponsStarting, vehicles: s.toggleVehicles, placements: s.togglePlacements,
    placeDropped: s.toggleDroppedPlacements, placeUnnamed: s.toggleUnnamedPlacements,
    objectives: s.toggleModeObjectives, flags: s.toggleFlagCarries, vip: s.toggleVipCrown,
    skull: s.toggleSkullCarrier, bomb: s.toggleBombCarrier, planFeed: s.togglePlanFeed, thinBar: s.toggleThinBarAtFullFrame,
    sound: sound.toggle,
    catWeapon: () => sound.toggleCategory(SOUND_CATEGORY_OF.catWeapon),
    catGrenade: () => sound.toggleCategory(SOUND_CATEGORY_OF.catGrenade),
    catMelee: () => sound.toggleCategory(SOUND_CATEGORY_OF.catMelee),
    catEquip: () => sound.toggleCategory(SOUND_CATEGORY_OF.catEquip),
  } satisfies Partial<Record<SettingId, () => void>>
}

/** Les gestes qui écrivent une valeur : un choix, un niveau, la coupe d'étage. */
function valueSetters(i: BindingInput): Partial<Record<SettingId, (value: SettingValue) => void>> {
  const { settings: s, relief, orientation, sound, feed } = i
  return {
    colors: (v) => s.setMarkerColors(v as MarkerColorsMode),
    livesWho: (v) => s.setLivesWho(v as LivesWho),
    livesWhen: (v) => s.setLivesWhen(v as LivesWhen),
    forecastDraw: (v) => s.setRespawnForecastDraw(v as ForecastDraw),
    bg: (v) => s.setBackgroundStyle(v as MapBackgroundVariant),
    orientation: (v) => orientation.setTurns(v === 'auto' ? null : normalizeTurns(Number(v))),
    heatMode: (v) => s.setHeatmapMode(v as HeatmapMode),
    heatSpan: (v) => s.setHeatmapSpan(v as HeatmapSpan),
    feedFilter: (v) => feed.choose(v as FeedFilterId),
    volume: (v) => sound.setVolume(Number(v)),
    floor: (v) => (v === null ? relief.floor.selectAll() : relief.floor.setHeight(Number(v))),
  }
}

export function bindSettings(i: BindingInput): SettingsBinding {
  const values = valuesOf(i)
  const flags = flagSetters(i)
  const writers = valueSetters(i)
  const set = (id: SettingId, value: SettingValue): void => {
    const toggle = flags[id]
    if (toggle) {
      if (typeof value === 'boolean' && values[id] !== value) toggle()
      return
    }
    writers[id]?.(value)
  }
  return { values, set }
}
