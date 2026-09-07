/**
 * equipmentZones.ts — LE JOUEUR DANS UNE ZONE D'ÉQUIPEMENT : ce que la fiche a le droit
 * d'affirmer quand un joueur se tient dans le disque d'un objet posé.
 *
 * LA SOURCE est la même que celle du calque de la carte : `equipmentPlacements` (schéma 10),
 * filtré par les MÊMES portes — `placementKind` (famille déployée, jamais un objet lâché) et
 * `isPlacementActive` (fenêtre d'affichage, durée officielle du capteur incluse). La fiche ne
 * dit donc jamais autre chose que ce que la carte dessine : si le disque est à l'écran et que
 * le joueur est dedans, la fiche le porte ; si le calque ne dessine pas la pose, la fiche se
 * tait. Deux réponses divergentes à « cette zone existe-t-elle ? » seraient un défaut.
 *
 * TROIS ZONES, ET SEULEMENT TROIS — celles dont l'objet EXERCE un effet de surface :
 *  - le CHAMP DE RÉPARATION (rayon déclaré, cf. placementShapes.REPAIR_FIELD_RADIUS_M) et
 *    l'ÉCRAN OCCULTANT (rayon déclaré, SHROUD_RADIUS_M) valent pour TOUT LE MONDE : le dôme
 *    du jeu soigne quiconque s'y tient, la bulle cache dans les deux sens. Aucun camp n'entre
 *    dans la règle, et c'est fidèle au jeu, pas une simplification ;
 *  - le CAPTEUR DE MENACES ne concerne que les ADVERSAIRES du poseur, et la règle de camp est
 *    EXACTEMENT celle de `sensorReveals` (threatSensor.ts) : `team_side` de la base des deux
 *    côtés, et sans camp connu — poseur non mesuré, vie sans ligne de scoreboard — RIEN n'est
 *    affirmé. La fiche dit ici l'ÉTAT (« il se tient dans la zone d'un capteur adverse »),
 *    au présent et en continu ; la marque de RÉVÉLATION de la carte, elle, garde sa propre
 *    règle (appartenance mesurée à l'instant du ping) — deux affirmations distinctes, toutes
 *    deux vraies. `sincePingMs` est publié pour que l'affichage puisse battre à la cadence
 *    officielle du capteur au lieu d'inventer un rythme.
 *
 * Pas de React, pas de canvas : logique pure, testée (equipmentZones.test.ts).
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import {
  isPlacementActive,
  placementKind,
  type PlacementKind,
  type PlacementToggles,
  type PlacementWindowTime,
} from '../layers/equipmentPlacementsLayer'
import { REPAIR_FIELD_RADIUS_M, SHROUD_RADIUS_M } from '../layers/placementShapes'
import { SENSOR_RADIUS_M, sensorPingAgeMs } from './threatSensor'

/** Ce que la fiche a besoin de LIRE : les poses, et le camp de la vie qui occupe un slot À UNE
 *  IMAGE (même contrat que `PlacementScene.sideOfSlot` — null = camp inconnu, donc jamais un
 *  ennemi ; la frame lève l'ambiguïté d'un slot réattribué entre manches). */
export interface ZoneScene {
  placements: readonly ReplayEquipmentPlacement[]
  sideOfSlot: (slot: number, frame: number) => string | null
}

/** Le joueur interrogé : sa vie (slot), sa position MONDE à l'image, et l'image. */
export interface ZoneQuery {
  slot: number
  x: number
  y: number
  frame: number
}

/** Ce que la fiche doit dire à une frame : chaque zone, indépendamment. */
export interface ZonePresence {
  repair: boolean
  /**
   * ms depuis la POSE de l'écran occultant qui couvre le joueur — la plus récente quand
   * plusieurs se recouvrent (même règle que le capteur : deux horloges superposées ne
   * diraient rien de plus). C'est l'horloge des ÉCLAIRS de la fiche (délai négatif,
   * cf. ReplayTeams.ZoneFxOverlay) : le scintillement reprend à son avancement réel
   * après un saut de lecture, jamais sur un rythme inventé. Null = hors de tout écran.
   */
  shroudSinceMs: number | null
  /**
   * ms depuis le dernier ping du capteur ADVERSE qui couvre le joueur — le plus fraîchement
   * pingé quand plusieurs se recouvrent (même règle que la marque de révélation : deux
   * horloges superposées ne diraient rien de plus). Null = hors de toute zone adverse.
   */
  sensorSincePingMs: number | null
}

/** Aucune allocation par frame : la valeur « hors de toute zone » est partagée. */
export const NO_ZONES: ZonePresence = Object.freeze({
  repair: false,
  shroudSinceMs: null,
  sensorSincePingMs: null,
})

/**
 * LES BASCULES DU TIROIR N'ENTRENT PAS ICI, et c'est un choix dit : elles gouvernent ce que
 * la CARTE affiche en PLUS des objets déployés (objets non identifiés, lâchers). Les trois
 * zones sont toutes des familles déployées, que ces bascules ne touchent pas — l'état d'un
 * joueur ne dépend pas d'un réglage d'affichage.
 */
const NO_TOGGLES: PlacementToggles = { showUnnamed: false, showDropped: false }

/**
 * Rayon de CHAQUE zone, en mètres monde — les MÊMES constantes que le tracé de la carte, et
 * chacune garde le statut que son fichier d'origine lui écrit : officielle pour le capteur,
 * déclarée (choix d'écran) pour le champ et l'écran. Une règle de rendu absente de cette
 * table n'est pas une zone : le mur est un arc, pas un disque, et un objet lâché n'agit pas.
 */
const ZONE_RADIUS_M: Partial<Record<PlacementKind, number>> = {
  field: REPAIR_FIELD_RADIUS_M,
  shroud: SHROUD_RADIUS_M,
  sensor: SENSOR_RADIUS_M,
}

/**
 * zonePresenceAt dit dans quelles zones un joueur se tient à une frame.
 *
 * Le balayage est linéaire sur les poses (dizaines par film) avec deux sorties O(1) en tête
 * (famille sans zone, fenêtre close) : négligeable devant le rendu, même appelé par fiche et
 * par image (huit fiches sur le pire corpus).
 */
export function zonePresenceAt(
  scene: ZoneScene,
  query: ZoneQuery,
  time: PlacementWindowTime,
): ZonePresence {
  let repair = false
  let shroudSinceMs: number | null = null
  let sensorSincePingMs: number | null = null
  // Le joueur interrogé est vivant à l'image (la fiche l'a résolu depuis sa vie courante) : son
  // camp se lit donc à `query.frame`, jamais figé pour tout le match.
  const side = scene.sideOfSlot(query.slot, query.frame)
  for (const p of scene.placements) {
    const kind = placementKind(p, NO_TOGGLES)
    if (!kind) continue
    const radius = ZONE_RADIUS_M[kind]
    if (radius === undefined) continue
    if (!isPlacementActive(p, kind, query.frame, time)) continue
    const dx = query.x - p.x
    const dy = query.y - p.y
    if (dx * dx + dy * dy > radius * radius) continue
    if (kind === 'field') repair = true
    else if (kind === 'shroud') {
      // L'ÂGE DE LA POSE, pas un booléen : les éclairs de la fiche se calent dessus.
      const since = (query.frame - p.t0) * time.frameMs
      if (shroudSinceMs === null || since < shroudSinceMs) shroudSinceMs = since
    } else {
      // CAPTEUR : adverse seulement, les deux camps CONNUS — la règle de sensorReveals,
      // à l'identique. Sans poseur mesuré ou sans camp, le disque se dessine mais la fiche
      // n'affirme aucune inimitié.
      if (p.owner < 0 || side === null) continue
      // Le poseur se lit à l'instant de sa POSE (`p.t0`), où il était vivant sur ce slot.
      const owner = scene.sideOfSlot(p.owner, p.t0)
      if (owner === null || owner === side) continue
      const since = sensorPingAgeMs((query.frame - p.t0) * time.frameMs)
      if (sensorSincePingMs === null || since < sensorSincePingMs) sensorSincePingMs = since
    }
  }
  if (!repair && shroudSinceMs === null && sensorSincePingMs === null) return NO_ZONES
  return { repair, shroudSinceMs, sensorSincePingMs }
}
