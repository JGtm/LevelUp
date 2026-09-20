/**
 * vehiclesAim.ts — LA VISÉE D'UN OCCUPANT DE VÉHICULE (schéma 39) : quelle direction, à quelle
 * image, et quand retomber sur le cap du châssis.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-03). `vehiclesLayer.ts` a franchi le seuil de taille du
 * dépôt (CLAUDE.md n°5) en gagnant cette septième responsabilité, et la règle est d'EXTRAIRE
 * plutôt que d'exempter — exactement le geste qui avait déjà séparé `vehiclesPaint.ts` (le canvas)
 * de `vehiclesLayer.ts` (les règles). La couture tombe ici sur une frontière nette : ce fichier ne
 * connaît QUE la visée, et il est le seul à lire `ReplayVehicleRide.aim`.
 *
 * CE QUE LE SCHÉMA 31 A CHANGÉ, ET C'EST LE POINT DU LOT. Le cône d'un véhicule était dessiné au
 * CAP DU CHÂSSIS (`vehicleHeadingAt`), pour le seul conducteur, faute de mieux : un occupant
 * attaché ne réplique plus sa position, et le dépôt en concluait qu'il ne répliquait plus rien. Le
 * lot V11 a montré que la faute était dans le DÉCODEUR, pas dans le film — la visée de CHAQUE
 * occupant (conducteur, artilleur, passager, chacun sur son propre slot bipède) continue d'être
 * transmise pendant tout l'épisode, dans des records qui ne portent AUCUNE position. Chiffres
 * publiés avec le champ : justesse 0,2 à 0,5 deg contre la référence `Point.h` (témoin par mélange
 * 75,7 à 93,7 deg), couverture 35 épisodes attestés sur 35, et un écart au cap du châssis de 15,7
 * à 21,8 deg en médiane (q3 39,6 à 52,9 deg) — c'est-à-dire l'erreur que faisait l'ancien cône.
 *
 * CE QUI NE VIENT PAS DE LÀ, réfuté AVEC TÉMOIN au même lot : l'orientation de la TOURELLE en tant
 * qu'objet. L'entité tourelle ne réplique rien du tout (formes de masque plates, sous le plancher
 * de bruit d'une bande fantôme sur un des deux films). Le cône de l'artilleur vient de L'HOMME.
 */
import type { ReplayVehicleAim, ReplayVehicleRide } from '@/lib/api/types'

import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  vehicleAimAngle,
  vehicleDriverAt,
  vehicleHeadingAt,
  FAMILLES_ARME_FIXE,
} from './vehiclesLayer'

/**
 * VEHICLE_AIM_HOLD_FRAMES — combien de frames une lecture de visée reste EN VIGUEUR après son
 * instant, avant que le cône ne retombe sur le cap du châssis.
 *
 * DIX FRAMES, C'EST-À-DIRE UNE SECONDE, ET LE CHIFFRE VIENT DE LA MESURE : le film réplique la
 * visée d'un occupant à 5 à 46 lectures par seconde (lot V11, 35 épisodes attestés, 5 films) pour
 * 10 frames publiées par seconde — une seconde entière sans la moindre lecture n'est donc pas un
 * défaut d'échantillonnage, c'est une interruption réelle du flux.
 *
 * CE N'EST PAS `TIMING_MS.aimHold` (5 s, le maintien du regard d'un PION), et il ne faut pas les
 * confondre : le pion tient sa visée longtemps parce que son `h` n'est transmis que sur ~44 % de
 * ses points, alors que la série d'un occupant est DENSE par construction. Maintenir 5 s ici
 * afficherait une direction périmée là où le châssis, lui, est connu à l'image près.
 */
export const VEHICLE_AIM_HOLD_FRAMES = 10

/**
 * VehicleOccupantAim — CE QU'UN OCCUPANT REGARDE à une image, prêt à peindre.
 *
 * `measured` n'est PAS décoratif : c'est la différence entre une visée LUE dans le film (0,2 à
 * 0,5 deg d'écart à la référence publiée) et le REPLI sur le cap du châssis, qui se trompe de 15,7
 * à 21,8 deg en médiane et de plus de 40 deg au quartile supérieur. Les tests s'y accrochent, et
 * un rendu futur peut le lire (opacité, style) sans avoir à redécouvrir la règle.
 */
export interface VehicleOccupantAim {
  /** Angle CANEVAS du cône, en radians (l'inversion monde -> écran est déjà appliquée). */
  ang: number
  /** Élévation en degrés, positif = vers le haut. 0 = à plat, y compris en repli. */
  pitchDeg: number
  /** true = visée MESURÉE de cet occupant ; false = repli sur le cap du châssis. */
  measured: boolean
}

/**
 * vehicleRideAimReading — la dernière lecture de visée EN VIGUEUR à `frame` pour cet épisode, ou
 * `null` (aucune série, aucune lecture au plus tard à cette image, ou lecture trop ancienne).
 *
 * AUCUNE INTERPOLATION, comme côté serveur : interpoler deux caps ferait tourner le cône par le
 * chemin le plus court à travers 0/360 deg, un artefact que le film ne montre pas. On MAINTIENT la
 * dernière lecture le temps du maintien, on n'invente pas celles qui manquent.
 *
 * UN POINT SANS `h` EST SAUTÉ. Le contrat serveur publie 360 plutôt que 0 (le PIÈGE omitempty,
 * cf. `headingForJSON` côté Go), donc le cas ne devrait pas exister — mais le champ est optionnel
 * au contrat, et un cône laissé à 0 par défaut pointerait l'est sans que rien ne l'ait mesuré.
 */
export function vehicleRideAimReading(
  ride: ReplayVehicleRide,
  frame: number,
): ReplayVehicleAim | null {
  const aim = ride.aim
  if (!aim || aim.length === 0) return null
  for (let i = lastIndexAt(aim, frame); i >= 0; i--) {
    const a = aim[i]
    if (frame - a.t > VEHICLE_AIM_HOLD_FRAMES) return null
    if (a.h !== undefined) return a
  }
  return null
}

/**
 * vehicleOccupantAimAt — LA DIRECTION DU CÔNE D'UN OCCUPANT : sa mesure d'abord, le châssis en
 * repli.
 *
 * LE REPLI RESTE LE CAP DU CHÂSSIS quand l'épisode n'a pas de visée en vigueur à cette image —
 * c'est EXACTEMENT le comportement d'avant la série de visée, si bien qu'un artefact antérieur (aucun `aim`
 * nulle part) rend le même cône qu'avant, sans branche particulière ni test de version.
 */
export function vehicleOccupantAimAt(
  track: ReplayVehicleTrackReady,
  ride: ReplayVehicleRide,
  frame: number,
): VehicleOccupantAim {
  const read = vehicleRideAimReading(ride, frame)
  if (read && read.h !== undefined) {
    return { ang: vehicleAimAngle(read.h), pitchDeg: read.p ?? 0, measured: true }
  }
  // À PLAT EN REPLI, et c'est une mesure : le cap du châssis est la direction d'un DÉPLACEMENT,
  // qui est horizontal — lui prêter une élévation inventerait un second angle.
  return { ang: vehicleAimAngle(vehicleHeadingAt(track, frame)), pitchDeg: 0, measured: false }
}
/**
 * vehicleChassisHeadingAt — LE CAP AUQUEL LE CHÂSSIS SE DESSINE : la visée du CONDUCTEUR sur les
 * familles à arme fixe, la vélocité partout ailleurs.
 *
 * ## Pourquoi cette fonction existe (2026-09-20)
 *
 * LE CHÂSSIS ÉTAIT DESSINÉ À LA DIRECTION DE SA VÉLOCITÉ (`vehicleHeadingAt`) — c'est-à-dire
 * « là où le véhicule se déplace », pas « là où il pointe ». Un véhicule qui recule, dérape,
 * tourne, vole ou est à l'arrêt n'a pas son nez dans cette direction, et 16 à 31 % des points de
 * trajectoire n'en portent aucune (sous le seuil de vitesse du producteur).
 *
 * L'ORIENTATION PROPRE DU FILM A ÉTÉ RÉFUTÉE, elle ne viendra pas combler ce trou : le lot 5.2-B
 * a mesuré le composant candidat de `ti=40` (`i2`) et c'est un vecteur HAUT, pas un avant —
 * médianes de 54 à 105 degrés contre la vélocité.
 *
 * LA DÉCISION DE L'UTILISATEUR (2026-09-20) est donc de prendre la seule direction que le
 * véhicule porte en propre ET qui soit mesurée : « le châssis s'oriente là où l'ARME pointe ».
 * Sur une famille à arme fixe, viser C'EST tourner le véhicule — la visée du conducteur est donc
 * l'avant du châssis, à 0,2-0,5 degré près (schéma 31). Sur une famille à tourelle, elle ne l'est
 * pas et le pilote n'est même pas le tireur : le repli reste la vélocité.
 *
 * ## Les quatre régimes, dans l'ordre
 *
 *   1. VISÉE DU CONDUCTEUR — famille à arme fixe (`FAMILLES_ARME_FIXE`), un épisode du SIÈGE 0
 *      en vigueur à cette image, et une lecture de visée en vigueur dessus.
 *   2. VÉLOCITÉ — tout le reste, exactement comme avant : famille à tourelle, siège 0 vide
 *      (véhicule abandonné, ou conduit par personne), ou épisode sans lecture de visée en vigueur.
 *   3. CAP VOISIN — aucun échantillon en arrière ne porte de cap (véhicule encore à quai).
 *   4. DÉFAUT — `VEHICLE_DEFAULT_HEADING_DEG`, nez vers le haut.
 *
 * Les régimes 2 à 4 sont `vehicleHeadingAt` tel quel : cette fonction ne fait qu'AJOUTER le
 * premier devant.
 *
 * ## Pourquoi ici et pas dans `vehiclesLayer`
 *
 * Ce module est le SEUL à lire `ReplayVehicleRide.aim` (cf. l'en-tête), et `vehiclesLayer` ne
 * l'importe pas — l'y poser créerait un cycle entre les deux fichiers. La TABLE des familles,
 * elle, reste chez les autres tables de familles (`FAMILLES_ARME_FIXE`).
 *
 * ## Ce que cette fonction NE change PAS
 *
 * Le REPLI DU CÔNE de visée (`vehicleOccupantAimAt`) garde `vehicleHeadingAt`, la vélocité nue :
 * ce repli est marqué `measured: false`, et lui donner la visée d'un AUTRE occupant ferait passer
 * une mesure d'autrui pour une approximation de soi.
 */
export function vehicleChassisHeadingAt(
  track: ReplayVehicleTrackReady,
  frame: number,
): number {
  if (track.family !== undefined && FAMILLES_ARME_FIXE.has(track.family)) {
    const driver = vehicleDriverAt(track, frame)
    const read = driver ? vehicleRideAimReading(driver, frame) : null
    if (read?.h !== undefined) return read.h
  }
  return vehicleHeadingAt(track, frame)
}
