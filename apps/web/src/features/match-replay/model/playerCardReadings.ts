/**
 * playerCardReadings.ts — TOUT CE QU'UNE FICHE LIT À UNE IMAGE, sans React.
 *
 * EXTRAIT DE `ReplayTeams.tsx` LE 2026-09-06 (plan fiches compactes, item 1.3) : ces lectures
 * vivaient dans le corps du composant `PlayerCard`, mêlées au rendu. Les sortir donne deux
 * choses : une tuile qui ne fait plus que rendre ce qu'on lui dit, et une mesure du MODÈLE
 * SEUL (`ui/ReplayTeams.perf.test.tsx`) qui ne passe pas par la réconciliation.
 *
 * AUCUNE RÈGLE N'A CHANGÉ AU PASSAGE — les commentaires qui suivent sont ceux de la fiche,
 * déplacés avec le code qu'ils expliquent. La règle n° 1 des fiches vaut ici : une valeur non
 * lue est une LACUNE (`null`), jamais un zéro.
 */
import {
  playerCountersAt,
  type PlayerCounters,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
import { stripBotSuffix } from '@/lib/players/displayName'

import { activeEquipmentAt } from './equipmentFx'
import { NO_ZONES, zonePresenceAt, type ZonePresence, type ZoneScene } from './equipmentZones'
import { equippedWeapons, type EquippedReading } from './equippedLogic'
import { objectiveMarkAt, type ObjectiveMarkKind } from './objectiveMark'
import { lastTeleportAge, type TranslocationMoment } from './placementTeleport'
import { playerCardFx, type CardFx } from './playerCardFx'
import type { ReplayText } from '../i18n/i18nContract'
import type { PlacementWindowTime } from '../layers/equipmentPlacementsLayer'
import { positionAt, trackWindow } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  playerName,
  playerStateAt,
  type PlayerState,
  type ReplayPlayer,
  type VitalityPresence,
} from '../../../lib/replay/rosterLogic'

/**
 * Ce que les EFFETS d'une fiche lisent en dehors d'elle-même : la scène des zones (poses +
 * camps), l'axe de temps des fenêtres de pose, et les passages par faille. Construite UNE
 * FOIS par la colonne — les fiches la consomment, aucune ne la recalcule.
 */
export interface CardFxScene {
  zones: ZoneScene
  time: PlacementWindowTime
  teleports: readonly TranslocationMoment[]
}

export interface PlayerCardInput {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  presence: VitalityPresence
  /** Durée des éclats d'événement, en images (cf. `FLASH_MS` de la colonne). */
  flashFrames: number
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
  fxScene: CardFxScene
  text: ReplayText
}

export interface PlayerCardReadings {
  /** Compteurs publiés par le film à l'instant lu ; `null` = non publié (pas « zéro »). */
  live: PlayerCounters | null
  state: PlayerState
  /** Le nom à afficher, suffixe « [bot] » retiré, repli « joueur inconnu ». */
  name: string
  /** La lecture ORDONNÉE des armes de la vie courante ; `null` sans vie ou sans loadout lu. */
  equipped: EquippedReading | null
  /** L'index de FILM du joueur : la clé des lancers de grenade ; `null` si le roster le tait. */
  filmIndex: number | null
  zones: ZonePresence
  objective: ObjectiveMarkKind | null
  fx: CardFx
}

export function playerCardReadings({
  player, doc, frame, presence, flashFrames, scoreTimeline, fxScene, text: t,
}: PlayerCardInput): PlayerCardReadings {
  // LES COMPTEURS DU FILM, quand ce joueur est publié. `null` veut dire « pas publié », pas
  // « à zéro » : sur le témoin Slayer 6 joueurs sur 8 en portent, et le mode Oddball n'en
  // publie aucun (0/32 en phase 0). La fiche retombe alors sur les totaux de la BASE, qui
  // valent pour tout le match — c'est ce qu'elle affichait avant ce lot.
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
  const state = playerStateAt(player, frame, presence)
  // Suffixe « [bot] » = marqueur de donnée killsource (schéma 36), pas d'affichage —
  // retiré ici sans toucher au repli `t.unknownPlayer` (playerName() reste `null`-able).
  const rawName = playerName(player)
  const name = (rawName ? stripBotSuffix(rawName) : null) ?? t.unknownPlayer
  const equipped = state.life ? equippedWeapons(doc, state.life.slot, frame) : null
  // L'index de FILM du joueur : la clé des lancers de grenade (l'auteur y est écrit).
  const filmIndex = doc.roster.find((r) => r.xuid === player.xuid)?.filmIndex ?? null
  // Les DEUX éclats d'événement : le coup fatal et la réapparition. Ils durent le temps de
  // leur animation ; le délai NÉGATIF la fait reprendre à son avancement réel, donc elle
  // reste juste après un saut dans le temps de lecture (cf. globals.css).
  const deathAge = state.sinceDeath
  const lifeAge = state.alive && state.life && trackWindow(state.life).start > 0
    ? frame - trackWindow(state.life).start
    : -1
  // L'ÉTAT ACTIF d'équipement de la vie courante : même chaîne slot -> fiche que le flash
  // de mort. L'effet couvre TOUTE la fiche (demande utilisateur du 14/08) et dure
  // exactement l'épisode mesuré — une fiche morte n'en porte jamais (les épisodes se
  // ferment à la mort au plus tard).
  const equipment = state.alive && state.life
    ? activeEquipmentAt(doc, state.life.slot, frame)
    : null
  // LES ZONES SOUS LE JOUEUR et le dernier PASSAGE par faille de cette vie : mêmes portes
  // que la carte (equipmentZones.ts), position interpolée de la vie courante. Sans position
  // lisible — et sur une fiche morte — aucune zone n'est affirmée.
  const pos = state.alive && state.life ? positionAt(state.life.points, frame) : null
  const zones = pos && state.life
    ? zonePresenceAt(fxScene.zones, { slot: state.life.slot, x: pos.x, y: pos.y, frame }, fxScene.time)
    : NO_ZONES
  // L'OBJECTIF PORTÉ : drapeau, crâne, VIP (des périodes attribuées, donc un état qui dure),
  // ou la prise de base (un instant attribué, tenu quelques secondes — cf. objectiveMark.ts).
  // Comme pour l'équipement et les zones, une fiche morte n'en porte aucun : un mort a lâché
  // ce qu'il tenait, et la tuile ne dit plus que la mort.
  const objective = state.alive ? objectiveMarkAt(doc, player.xuid, frame) : null
  const teleportAge = state.alive && state.life
    ? lastTeleportAge(fxScene.teleports, state.life.slot, frame)
    : -1
  // LA COMPOSITION DES EFFETS vit dans playerCardFx.ts (mort, éclats à délai négatif,
  // verre trempé du camouflage, encadrés, voile de l'écran occultant) : la fiche lui donne
  // les âges et rend ce qu'il dit, réparti sur ses DEUX couches (dessous / incrustation).
  const fx = playerCardFx({
    alive: state.alive,
    deathAge,
    lifeAge,
    teleportAge,
    flashFrames,
    equipment,
    zones,
    objective,
    text: t,
  })
  return { live, state, name, equipped, filmIndex, zones, objective, fx }
}
