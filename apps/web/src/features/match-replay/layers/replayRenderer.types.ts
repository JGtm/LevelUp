/**
 * replayRenderer.types.ts — CE QUE LE MOTEUR DE LA TOILE 2D CONSOMME.
 *
 * `ReplaySceneInputs` est l'état complet dont `replayRenderer.ts` a besoin pour peindre UNE image
 * sur UNE toile : le document, le cadrage de cette toile (`view`), les encres, les identités par
 * slot, les calques câblés par hook et les listes d'effets déjà bâties. Rien n'y est lu depuis
 * React ni depuis le DOM : un appelant qui dispose de ces valeurs (la toile du rejeu, une
 * mini-carte avec une autre vue) peint avec le même moteur.
 *
 * Les types des hooks sont repris par `ReturnType` : le moteur n'importe de ces modules que des
 * types, la forme servie par chaque hook reste ainsi sa seule définition.
 */
import type { RefObject } from 'react'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayerFocusLayers } from '../hooks/useReplayPlayerFocus'
import type { ReplayTiming } from '../hooks/useReplayTiming'
import type { ReplayMapBackgroundLayer } from '../hooks/useReplayView'
import type { ReplayLocale } from '../i18n/i18n'
import type { CanvasView } from '../model/replayView'
import type { useReplayAbilityFx } from './useReplayAbilityFx'
import type { useReplayBombBlast } from './useReplayBombBlast'
import type { useReplayBombCarrier } from './useReplayBombCarrier'
import type { useReplayFlagCarries } from './useReplayFlagCarries'
import type { useReplayFx } from './useReplayFx'
import type { useReplayGrenadeRest } from './useReplayGrenadeRest'
import type { useReplayGroundWeapons } from './useReplayGroundWeapons'
import type { useReplayInks } from './useReplayInks'
import type { useReplayKillFocus } from './useReplayKillFocus'
import type { useReplayLives } from './useReplayLives'
import type { useReplayObjectiveObjects } from './useReplayObjectiveObjects'
import type { useReplayPlacements } from './useReplayPlacements'
import type { useReplayRespawnForecast } from './useReplayRespawnForecast'
import type { useReplaySkullCarrier } from './useReplaySkullCarrier'
import type { useReplayVehicles } from './useReplayVehicles'
import type { useReplayVipCrown } from './useReplayVipCrown'
import type { useReplayWeaponPads } from './useReplayWeaponPads'
import type { SlotIdentity } from './useSlotIdentity'
import type { ReplayZoneStates } from './useZoneStates'

type Inks = ReturnType<typeof useReplayInks>
type Fx = ReturnType<typeof useReplayFx>
type Placements = ReturnType<typeof useReplayPlacements>

/**
 * Les calques CUITS HORS ÉCRAN (sol, zones nommées, chaleur, objectifs) et le cadrage contre
 * lequel ils ont été cuits : le moteur les recopie, décalés quand la cuisson date d'un autre
 * cadrage (glisser, suivi).
 */
export interface CookedLayers {
  origin: RefObject<CanvasView | null>
  wallPlan: RefObject<HTMLCanvasElement | null>
  zones: RefObject<HTMLCanvasElement | null>
  heat: RefObject<HTMLCanvasElement | null>
  objectives: RefObject<HTMLCanvasElement | null>
}

/** Les bascules qui ouvrent ou ferment un calque entier. */
export interface SceneShow {
  zones: boolean
  shotFx: boolean
  placements: boolean
  killFx: boolean
  modeObjectives: boolean
  /** Le calque des murs est dessiné (vrai quand le fond posé n'est pas déjà le plan). */
  wallPlan: boolean
  aim: boolean
  trail: boolean
}

/** Les encres que le moteur pose lui-même (celles des calques câblés par hook vivent dans leur hook). */
export interface SceneInks {
  geometry: Inks['geometry']
  grenade: Inks['grenade']
  shot: Inks['shot']
  fx: Inks['fx']
  floorEdge: Inks['floor']['edge']
  wall: Inks['wall']
  rift: Inks['rift']
  shroud: Inks['shroud']
  self: Inks['self']
  labelStroke: Inks['labelStroke']
  markerOutline: Inks['markerOutline']
}

/** Les poses d'équipement et les failles, tels que `useReplayPlacements` les sert. */
export interface ScenePlacements {
  all: Placements['all']
  windowTime: Placements['windowTime']
  toggles: Placements['toggles']
  rift: Placements['rift']
  /** Nombre de poses dessinables : zéro ferme le calque. */
  drawableCount: number
}

/** Les calques que leur hook câble et nomme : le moteur n'en connaît que le peintre et l'id. */
export interface SceneLayers {
  weaponPads: ReturnType<typeof useReplayWeaponPads>
  groundWeapons: ReturnType<typeof useReplayGroundWeapons>
  vehicles: ReturnType<typeof useReplayVehicles>
  abilityFx: ReturnType<typeof useReplayAbilityFx>
  grenadeRest: ReturnType<typeof useReplayGrenadeRest>
  flags: ReturnType<typeof useReplayFlagCarries>
  objectiveObjects: ReturnType<typeof useReplayObjectiveObjects>
  vipCrown: ReturnType<typeof useReplayVipCrown>
  skullCarrier: ReturnType<typeof useReplaySkullCarrier>
  bombCarrier: ReturnType<typeof useReplayBombCarrier>
  bombBlast: ReturnType<typeof useReplayBombBlast>
  killFocus: ReturnType<typeof useReplayKillFocus>
  lives: ReturnType<typeof useReplayLives>
  respawnForecast: ReturnType<typeof useReplayRespawnForecast>
}

/** L'état d'une toile : tout ce que le moteur lit pour peindre une image. */
export interface ReplaySceneInputs {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  reducedMotion: boolean
  /** Le cadrage DE CETTE TOILE : le fond, les calques et les cuissons s'y rapportent. */
  view: CanvasView
  /** La taille de dessin en px CSS (le moteur applique la densité de la toile). */
  size: { width: number; height: number }
  zRange: { min: number; max: number }
  background: ReplayMapBackgroundLayer | null
  cooked: CookedLayers
  timing: ReplayTiming['timing']
  eventHoldFrames: number
  shotHoldFrames: number
  show: SceneShow
  ink: SceneInks
  identity: Pick<
    SlotIdentity,
    'colorOfSlot' | 'colorOfSlotOrLast' | 'markOfSlot' | 'nameOfSlot' | 'campOfSlot' | 'seatedAtSlot'
  >
  focus: ReplayPlayerFocusLayers
  fx: Pick<Fx, 'shotFx' | 'fireMarks' | 'killFx' | 'objectivePulses'>
  zones: ReplayZoneStates
  placements: ScenePlacements
  layers: SceneLayers
}
