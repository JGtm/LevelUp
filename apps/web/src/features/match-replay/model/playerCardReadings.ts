/**
 * playerCardReadings.ts — TOUT CE QU'UNE FICHE LIT À UNE IMAGE, sans React.
 *
 * Ces lectures sont sorties du rendu : la tuile (`ui/ReplayPlayerCard.tsx`) ne fait que rendre ce
 * qu'on lui dit, et le modèle se mesure seul, sans la réconciliation de React. La règle n° 1
 * des fiches vaut ici : une valeur non lue est une LACUNE (`null`), jamais un zéro.
 */
import {
  playerCountersAt,
  type PlayerCounters,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
import { stripBotSuffix } from '@/lib/players/displayName'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { activeEquipmentAt } from './equipmentFx'
import { NO_ZONES, zonePresenceAt, type ZonePresence, type ZoneScene } from './equipmentZones'
import { equippedWeapons, type EquippedReading } from './equippedLogic'
import { objectiveMarkAt, type ObjectiveMarkKind } from './objectiveMark'
import { lastTeleport, type TranslocationMoment } from './placementTeleport'
import { playerCardFx, type CardFx } from './playerCardFx'
import type { ReplayText } from '../i18n/i18nContract'
import type { PlacementWindowTime } from '../layers/equipmentPlacementsLayer'
import { positionAt, trackWindow } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  playerName,
  playerStateAt,
  rosterEntryKey,
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

/**
 * cardName — LE NOM D'UNE TUILE DE PLACE, écrit une seule fois pour la fiche et pour la tuile
 * « pas encore apparu » : le nom du joueur (`playerName` : la base d'abord, le film ensuite), le
 * suffixe de donnée « [bot] » retiré, et le repli « joueur inconnu ».
 */
export function cardName(player: ReplayPlayer, t: ReplayText): string {
  // Suffixe « [bot] » = marqueur de donnée killsource (schéma 36), pas d'affichage —
  // retiré ici sans toucher au repli `t.unknownPlayer` (playerName() reste `null`-able).
  const rawName = playerName(player)
  return (rawName ? stripBotSuffix(rawName) : null) ?? t.unknownPlayer
}

/**
 * countersShown — LES COMPTEURS QUE LE TRIPLET AFFICHE : ceux du film à l'instant lu quand ce joueur
 * est publié, sinon les totaux de la base ; `null` sans l'un ni l'autre. Partagé entre le triplet de la fiche et
 * celui de la ligne de la colonne, qui l'affiche sous une autre forme.
 */
export function countersShown(
  live: PlayerCounters | null,
  board: MatchScoreboardRow | undefined,
): { kills?: number | null; deaths?: number | null; assists?: number | null } | null {
  if (live) return live
  if (!board) return null
  return { kills: board.kills, deaths: board.deaths, assists: board.assists }
}

/** Ce que dit une place qui ne rend aucune fiche : le nom de l'occupant qui la tient (`null` = place libre), l'état écrit et son infobulle. */
export interface SeatWaitingReading {
  name: string | null
  label: string
  hint: string
}

/**
 * seatWaitingReading — LA RÈGLE D'UNE PLACE EN ATTENTE, écrite une seule fois pour la tuile et pour
 * la ligne : sans occupant (Q20), la place est LIBRE et ne montre aucun nom ; avec un occupant qui
 * n'a pas encore de corps (Q21), il est nommé comme sur sa fiche (`cardName`) et « pas encore
 * apparu ». Rien d'autre n'est dit : ni vitalité, ni armes, ni compteurs.
 */
export function seatWaitingReading(occupant: ReplayPlayer | null, t: ReplayText): SeatWaitingReading {
  const name = occupant ? cardName(occupant, t) : null
  return name === null
    ? { name, label: t.seatVacant, hint: t.seatVacantHint }
    : { name, label: t.seatNotSpawned, hint: t.seatNotSpawnedHint }
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
  const name = cardName(player, t)
  const equipped = state.life ? equippedWeapons(doc, state.life.slot, frame) : null
  // L'index de FILM du joueur : la clé des lancers de grenade (l'auteur y est écrit).
  //
  // UN BOT N'A PAS DE XUID AU ROSTER (`r.xuid === ''`) : la jointure passe par
  // `rosterEntryKey` (rosterLogic.ts), DANS LE MÊME ESPACE que `player.xuid`
  // (`'bot:<nom>'`, cf. `buildPlayers`). Comparer `r.xuid` à `player.xuid` directement ne
  // matche jamais un bot — le badge de lancer de grenade ne s'allumait alors JAMAIS pour lui
  // (constat P2-5, audit vies anonymes 2026-09-06).
  const filmIndex = doc.roster.find((r) => rosterEntryKey(r) === player.xuid)?.filmIndex ?? null
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
  const teleport = state.alive && state.life
    ? lastTeleport(fxScene.teleports, state.life.slot, frame)
    : null
  // LA COMPOSITION DES EFFETS vit dans playerCardFx.ts (mort, éclats à délai négatif,
  // verre trempé du camouflage, encadrés, voile de l'écran occultant) : la fiche lui donne
  // les âges et rend ce qu'il dit, réparti sur ses DEUX couches (dessous / incrustation).
  const fx = playerCardFx({
    alive: state.alive,
    deathAge,
    lifeAge,
    teleportAge: teleport?.age ?? -1,
    teleportPose: teleport?.pose ?? false,
    flashFrames,
    equipment,
    zones,
    objective,
    text: t,
  })
  return { live, state, name, equipped, filmIndex, zones, objective, fx }
}
