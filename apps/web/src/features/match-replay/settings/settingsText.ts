/**
 * settingsText — LES LIBELLÉS ET LES INFOBULLES de la table des réglages : pour chaque identifiant, le texte que
 * porte son dictionnaire d'origine (calques, vies, relief, chaleur, son…), rangé ici une fois pour le panneau et
 * l'accès rapide. Aucun texte n'est recopié : un libellé se change là où il vit.
 *
 * LES OPTIONS D'UN CHOIX (fond, orientation, couleur des points, chaleur, vies, dessin) suivent la même règle :
 * `choiceText` rend le libellé et l'infobulle d'une option dans la langue de la page.
 *
 * LOGIQUE PURE : aucun React.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayText } from '../i18n/i18nContract'
import { normalizeTurns, turnsToDegrees } from '@/lib/replay/orientation'

import type { ReliefText } from '../i18n/relief.i18n'
import type { ChatterFrequency } from '../model/chatterLines'
import { nearestChip } from '@/lib/replay/reliefViewLogic'
import type { SettingId } from './settingsTable'
import type { ReplayFloorControls } from './viewPanelModel'

export interface SettingText {
  label: string
  hint: string
}

/** Un libellé et une infobulle ; l'infobulle vide laisse la ligne sans titre. */
function entry(label: string, hint = ''): SettingText {
  return { label, hint }
}

/** Le texte de chaque réglage de la table, dans la langue de la page. */
export function settingText(locale: ReplayLocale, id: SettingId): SettingText {
  return TEXTS[id](REPLAY_TEXT[locale])
}

const TEXTS: Readonly<Record<SettingId, (t: ReplayText) => SettingText>> = {
  aim: (t) => entry(t.layerAim, t.layerAimHint),
  trail: (t) => entry(t.layerTrail, t.layerTrailHint),
  colors: (t) => entry(t.markerColorsTitle, t.settings.colorsHint),
  shotFx: (t) => entry(t.layerShotFx, `${t.layerShotFxHint} ${t.layerShotFxCoverage}`),
  killFx: (t) => entry(t.layerKillFx, t.layerKillFxHint),
  spawns: (t) => entry(t.lives.layerSpawns, t.lives.layerSpawnsHint),
  deaths: (t) => entry(t.lives.layerDeaths, t.lives.layerDeathsHint),
  killers: (t) => entry(t.lives.layerKillers, t.lives.layerKillersHint),
  livesWho: (t) => entry(t.lives.whoLabel),
  livesWhen: (t) => entry(t.lives.whenLabel),
  forecast: (t) => entry(t.respawnForecast.title, t.respawnForecast.hint),
  forecastDraw: (t) => entry(t.respawnForecast.drawLabel),
  chatterBubbles: (t) => entry(t.chatter.layerBubbles, t.chatter.layerBubblesHint),
  chatterIcon: (t) => entry(t.chatter.layerIcon, t.chatter.layerIconHint),
  bg: (t) => entry(t.layerBackground, t.settings.backgroundHint),
  reliefLines: (t) => entry(t.relief.lines, t.relief.linesHint),
  wallPlan: (t) => entry(t.layerWallPlan, t.layerWallPlanHint),
  floor: (t) => entry(t.relief.floorTitle, t.relief.floorInfo),
  floorFollow: (t) => entry(t.relief.floorFollow, t.relief.floorFollowHint),
  orientation: (t) => entry(t.orientationTitle, t.orientationHint),
  zones: (t) => entry(t.layerZones, t.layerZonesHint),
  heatmap: (t) => entry(t.layerHeatmap, t.layerHeatmapHint),
  heatMode: (t) => entry(t.heatmapReading),
  heatSpan: (t) => entry(t.heatmapSpanTitle),
  weaponPads: (t) => entry(t.layerWeaponPads, t.layerWeaponPadsHint),
  groundWeapons: (t) => entry(t.layerGroundWeapons, t.layerGroundWeaponsHint),
  groundSpecial: (t) => entry(t.layerGroundWeaponsSpecialOnly, t.layerGroundWeaponsSpecialOnlyHint),
  groundStarting: (t) => entry(t.layerGroundWeaponsStarting, t.layerGroundWeaponsStartingHint),
  vehicles: (t) => entry(t.layerVehicles, t.layerVehiclesHint),
  placements: (t) => entry(t.layerPlacements, t.layerPlacementsHint),
  placeDropped: (t) => entry(t.layerPlacementsDropped, t.layerPlacementsDroppedHint),
  placeUnnamed: (t) => entry(t.layerPlacementsUnnamed, t.layerPlacementsUnnamedHint),
  objectives: (t) => entry(t.layerModeObjectives, t.layerModeObjectivesHint),
  flags: (t) => entry(t.layerFlagCarries, t.layerFlagCarriesHint),
  vip: (t) => entry(t.layerVipCrown, t.layerVipCrownHint),
  skull: (t) => entry(t.layerSkullCarrier, t.layerSkullCarrierHint),
  bomb: (t) => entry(t.layerBombCarrier, t.layerBombCarrierHint),
  feedFilter: (t) => entry(t.settings.feedFilterLabel),
  planFeed: (t) => entry(t.frames.planFeed, t.frames.planFeedHint),
  chatterFeed: (t) => entry(t.chatter.layerFeed, t.chatter.layerFeedHint),
  famEnemy: (t) => entry(t.chatter.family.enemy, t.chatter.familyHint),
  famShields: (t) => entry(t.chatter.family.shields, t.chatter.familyHint),
  famSquadDown: (t) => entry(t.chatter.family.squadDown, t.chatter.familyHint),
  famPraise: (t) => entry(t.chatter.family.praise, t.chatter.familyHint),
  famOther: (t) => entry(t.chatter.family.other, t.chatter.familyHint),
  chatterFreq: (t) => entry(t.chatter.frequencyLabel, t.chatter.frequencyHint),
  sound: (t) => entry(t.sound, t.soundHint),
  volume: (t) => entry(t.settings.volumeLabel, t.soundVolume),
  catWeapon: (t) => entry(t.soundCategory.weapon),
  catGrenade: (t) => entry(t.soundCategory.grenade),
  catMelee: (t) => entry(t.soundCategory.melee),
  catEquip: (t) => entry(t.soundCategory.equipment),
  chatterVoice: (t) => entry(t.chatter.layerVoice, t.chatter.layerVoiceHint),
  thinBar: (t) => entry(t.frames.thinBar, t.frames.thinBarHint),
}

/** Un libellé d'option par option de chaque choix, et son infobulle quand le dictionnaire en porte une. */
export function choiceText(locale: ReplayLocale, id: SettingId, option: string, meshPercent: number | null = null): SettingText {
  const t = REPLAY_TEXT[locale]
  switch (id) {
    case 'colors':
      return entry(t.markerColorsMode[option as 'team' | 'player'], t.markerColorsHint[option as 'team' | 'player'])
    case 'bg': {
      if (option === 'plan') return entry(t.layerBackgroundPlan, t.layerBackgroundPlanHint)
      if (option === 'rendu') return entry(t.layerBackgroundRender, t.layerBackgroundRenderHint)
      const mesh = meshPercent === null ? '' : ` ${t.relief.backgroundMeshFmt(meshPercent)}`
      return entry(t.relief.background, `${t.relief.backgroundHint}${mesh}`)
    }
    case 'orientation':
      return option === 'auto'
        ? entry(t.frames.orientationAuto, t.frames.orientationAutoHint)
        : entry(`${turnsToDegrees(normalizeTurns(Number(option)))}°`, t.orientationHint)
    case 'heatMode':
      return entry(t.heatmapMode[option as 'presence' | 'kills'], t.heatmapModeHint[option as 'presence' | 'kills'])
    case 'heatSpan':
      return entry(t.heatmapSpan[option as 'match' | 'live'], t.heatmapSpanHint[option as 'match' | 'live'])
    case 'livesWho':
      return entry(t.lives.who[option as 'all' | 'team' | 'followed'], t.lives.whoHint[option as 'all' | 'team' | 'followed'])
    case 'livesWhen':
      return entry(t.lives.when[option as 'all' | 'sofar'], t.lives.whenHint[option as 'all' | 'sofar'])
    case 'forecastDraw':
      return entry(t.respawnForecast.draw[option as 'both' | 'zones' | 'points'], t.respawnForecast.drawHint[option as 'both' | 'zones' | 'points'])
    case 'chatterFreq':
      return entry(t.chatter.frequency[option as ChatterFrequency], t.chatter.frequencyOptionHint[option as ChatterFrequency])
    default:
      return entry(option)
  }
}

/** La note sous « Étage de la fiche ouverte » : l'explication du grisé, ou le rang suivi ; rien sinon. */
export function followNote(t: ReliefText, floor: ReplayFloorControls): string | null {
  if (!floor.follow.available) return t.floorFollowNone
  const { state, cut } = floor
  if (!floor.follow.on || state.state !== 'ready' || cut === null) return null
  const named = nearestChip(state.chips, cut)
  return named ? t.floorFollowOnFmt(t.floorRankFmt(named.rank)) : null
}
