/**
 * placementFixtures.ts — LE DÉCOR PARTAGÉ des tests de poses d'équipement.
 *
 * POURQUOI CE FICHIER (CLAUDE.md n° 6, « à la 3e copie on centralise »). Le lot des familles
 * nommées a découpé le calque en trois modules — la décision (`equipmentPlacementsLayer`), le
 * mur (`placementWall`), les formes à centre projeté (`placementShapes`) — et leurs trois
 * fichiers de tests ont besoin du MÊME cadrage, de la MÊME pose de référence et du MÊME
 * contexte enregistreur. Trois copies auraient divergé au premier réglage d'échelle, et un
 * test qui ne mesure plus la même carte que son voisin ne dit plus rien.
 *
 * LA POSE PAR DÉFAUT EST UN DÉPLOIEMENT DE PANNEAU, et c'est le point : depuis le schéma 10,
 * c'est le SEUL cas qui dessine un arc de mur (origine `deployed` ET identifiant de panneau).
 * L'écrire ici évite que chaque test ne répète les deux conditions du filtre.
 */
import type { ReplayEquipmentPlacement, ReplayPoint } from '@/lib/api/types'

import {
  drawEquipmentPlacementsLayer,
  PLACEMENT_ORIGIN_DEPLOYED,
  type PlacementInk,
  type PlacementScene,
  type PlacementTime,
} from '../layers/equipmentPlacementsLayer'
import { WALL_PANEL_IDS } from '../layers/placementWall'
import { worldToCanvas } from '../model/replayLogic'
import type { ReplayTrackReady } from '../model/replayNormalize'
import { recordingContext } from './recordingContext'

/** 10 m de côté sur 100 px : 10 px par mètre — le plancher de lisibilité ne mord pas. */
export const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}

export const TIME: PlacementTime = {
  frame: 50,
  frameMs: 100,
  /** 600 images de 100 ms = 60 s de rejeu : la borne des poses sans fin connue. */
  frames: 600,
  k: 1,
  reducedMotion: false,
  showUnnamed: false,
  /**
   * LES DEUX BASCULES SONT ÉTEINTES DANS LA FIXTURE, y compris celle des objets lâchés — qui
   * est pourtant ALLUMÉE en production. C'est délibéré : le décor par défaut de
   * ces tests est le comportement HISTORIQUE du calque (seuls les déployés), pour que tout
   * test qui ne parle pas des lâchers continue de mesurer ce qu'il mesurait. Les tests des
   * lâchers, eux, l'allument explicitement — et l'un d'eux vérifie qu'allumée, elle ne change
   * RIEN aux déployés.
   */
  showDropped: false,
}

/** L'IDENTIFIANT DES PANNEAUX du mur — celui sur lequel l'arc se dessine (manifeste). */
export const PANEL_ID = WALL_PANEL_IDS[0]
/** L'APPAREIL du mur : la mesure le dit PORTÉ (13 % de déploiements), il ne dessine rien. */
export const DEVICE_ID = '0x8e2dc574'

/** Les identifiants réels des familles déployables, tels que le manifeste les nomme. */
export const SENSOR_ID = '0x72b63d69'
export const BEACON_ID = '0x730dc70f'
export const SEEKER_ID = '0x4744d742'
export const FIELD_ID = '0x32d97758'

/**
 * L'IDENTIFIANT DU SURBOUCLIER LÂCHÉ, relevé sur le témoin `01e1f945` (KOTH Catalyst) : la
 * SEULE pose de power-up du corpus des onze films, et elle est `dropped`. Écrit ici pour que
 * les tests des objets lâchés parlent de la donnée réelle et non d'un identifiant inventé.
 */
export const OVERSHIELD_ID = '0xb781197a'

/** Une pose : par défaut le déploiement d'un panneau de mur au centre de la carte. */
export function pose(over: Partial<ReplayEquipmentPlacement> = {}): ReplayEquipmentPlacement {
  return {
    t0: 10,
    t1: 100,
    x: 5,
    y: 5,
    family: 'wall',
    id: PANEL_ID,
    owner: 3,
    origin: PLACEMENT_ORIGIN_DEPLOYED,
    ...over,
  }
}

/**
 * La SCÈNE du calque. Par défaut aucune vie : la révélation n'a alors personne à marquer, et
 * chaque test qui la vise fournit ses propres vies et ses propres camps.
 */
export function scene(
  placements: ReplayEquipmentPlacement[],
  over: Partial<PlacementScene> = {},
): PlacementScene {
  return { placements, lives: [], sideOfSlot: () => null, ...over }
}

/** Une vie IMMOBILE, pour la révélation : deux échantillons à la même position. */
export function life(slot: number, x: number, y: number): ReplayTrackReady {
  const points: ReplayPoint[] = [
    { t: 0, x, y },
    { t: 200, x, y },
  ]
  return { slot, team: -1, startFrame: 0, endFrame: 200, points }
}

export const INK = {
  colorOfSlot: () => 'equipe',
  neutral: 'neutre',
  wall: 'mur',
  rift: { rim: 'faille-bord', core: 'faille-coeur' },
}

/** projected — un point monde dans le repère du canvas de test. */
export function projected(x: number, y: number) {
  return worldToCanvas({ x, y }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
}

/**
 * draw — le calque joué sur un contexte enregistreur ; rend la trace des primitives.
 *
 * Les deux derniers arguments ne servent qu'à la RÉVÉLATION : elle a besoin de vies, de leurs
 * camps, et parfois d'une encre par slot pour prouver que la marque porte celle du POSEUR.
 */
export function draw(
  placements: ReplayEquipmentPlacement[],
  time: PlacementTime = TIME,
  sceneOver: Partial<PlacementScene> = {},
  ink: PlacementInk = INK,
) {
  const { ops, ctx } = recordingContext()
  drawEquipmentPlacementsLayer(ctx, scene(placements, sceneOver), VIEW, time, ink)
  return ops
}

/** painted — combien de primitives de tracé une scène a émises : le test « rien du tout ». */
export function painted(
  placements: ReplayEquipmentPlacement[],
  time: PlacementTime = TIME,
): number {
  return draw(placements, time).filter((o) => o.op === 'stroke' || o.op === 'fill').length
}
