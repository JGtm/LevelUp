/**
 * vehicleCycleTime.ts — CE QUE LE DOCUMENT DIT D'UN EMPLACEMENT DE NAISSANCE DE VÉHICULE À UN
 * INSTANT : est-il occupé, et si non, dans combien de temps un véhicule y revient.
 *
 * LA SOURCE est `vehicleCycles` (schéma 63) : un emplacement par AMAS de naissances, avec la
 * médiane, les déciles, le nombre d'écarts MESURÉS et le nombre de manques COMPTÉS. La clé est
 * ABSENTE quand le cycle n'est pas établi — un emplacement sans récurrence mesurée n'existe pas
 * dans la liste, et il n'y a rien à en dire (même doctrine que `PadCycle`).
 *
 * POURQUOI CE FICHIER EST SÉPARÉ DU CALQUE, et c'est le patron de `weaponPadTime.ts` : cette
 * lecture est du TEMPS PUR. Elle sert au tracé, au survol et à l'infobulle — trois appelants pour
 * une seule règle — et n'a besoin ni d'un canvas, ni d'une encre, ni d'une densité de pixels.
 *
 * L'APPARIEMENT EMPLACEMENT -> VIES SE REFAIT ICI, ET IL DOIT : le document publie l'amas (son
 * barycentre) et les vies (leur naissance), jamais le lien entre les deux. Il se retrouve à la
 * MAILLE DE LA MESURE (`VEHICLE_CYCLE_CLUSTER_M`, la constante du producteur Go), et une vie
 * SANS naissance située n'entre dans aucun emplacement — lui en inventer un ferait naître un
 * compte à rebours entre deux endroits différents.
 *
 * « OCCUPÉ » VEUT DIRE « UNE VIE NÉE ICI EST ENCORE EN VIGUEUR », pas « un véhicule se trouve
 * au-dessus du point ». C'est la même sémantique que l'occupation d'un socle d'arme, et c'est
 * celle qui répond à la question posée : l'horloge du jeu repart quand l'objet DISPARAÎT (mesure
 * de l'item 2.4, reprise par `vehicle_cycles.go`). Un Warthog né ici et parti à l'autre bout de
 * la carte n'a donc PAS libéré son emplacement — et de fait, aucun véhicule n'y renaîtra tant
 * qu'il roule.
 *
 * LA FIN DOIT ÊTRE DATÉE ET DESTRUCTRICE, exactement comme côté Go : une fin `film_end` n'est pas
 * une mort et une fin `unknown` n'est pas datée. Prédire depuis elles reviendrait à mesurer le
 * recensement des images-clés, l'erreur que `V2_SPAWNS_COOLDOWNS` avait nommée.
 */
import type {
  ReplayDocumentReady,
  ReplayVehicleTrackReady,
} from '../../../lib/replay/replayNormalize'
import { respawnAt, type Respawn } from './respawnCountdown'
import { vehicleVisibleAt } from './vehiclesLayer'

/**
 * Rayon d'agglomération des naissances en UN emplacement, en mètres.
 *
 * C'EST LA CONSTANTE DU PRODUCTEUR, RECOPIÉE ICI PARCE QUE LE DOCUMENT NE LA PUBLIE PAS —
 * `vehicleCycleClusterM` de `film/replay/vehicle_cycles.go`. Un garde-rail la relit dans le
 * fichier Go (`vehicleCycleTime.test.ts`) : deux mailles différentes apparieraient des vies à
 * des emplacements voisins, et le décalage serait invisible à l'écran.
 */
export const VEHICLE_CYCLE_CLUSTER_M = 2.0

/** La valeur de `VehicleTrack.end` qui vaut une fin DATÉE — la seule (cf. l'en-tête). */
const VEHICLE_END_DESTROYED = 'destroyed'

/** Un emplacement de naissance tel que le document le publie. */
export type VehicleCycle = ReplayDocumentReady['vehicleCycles'][number]

/**
 * vehicleCycleLives — les vies nées à cet emplacement, dans l'ordre de leur naissance.
 *
 * À APPELER UNE FOIS PAR DOCUMENT, jamais par image : c'est un balayage de toutes les vies pour
 * chaque emplacement. Le calque le mémoïse (cf. `useReplayVehicles`).
 */
export function vehicleCycleLives(
  cycle: VehicleCycle,
  tracks: readonly ReplayVehicleTrackReady[],
): ReplayVehicleTrackReady[] {
  const r2 = VEHICLE_CYCLE_CLUSTER_M * VEHICLE_CYCLE_CLUSTER_M
  const out = tracks.filter((t) => {
    if (!t.spawn) return false
    return (t.spawn.x - cycle.x) ** 2 + (t.spawn.y - cycle.y) ** 2 <= r2
  })
  return out.sort((a, b) => a.t0 - b.t0)
}

/**
 * vehicleCycleOccupantAt — la vie née ici qui est ENCORE EN VIGUEUR à cette image, ou `null`.
 *
 * `vehicleVisibleAt` est le prédicat du calque des véhicules, et c'est volontairement le MÊME :
 * ce qui est dessiné sur la carte et ce qui occupe un emplacement doivent dire la même chose,
 * sans quoi un marqueur de réapparition apparaîtrait sous un sprite encore visible.
 */
export function vehicleCycleOccupantAt(
  lives: readonly ReplayVehicleTrackReady[],
  frame: number,
): ReplayVehicleTrackReady | null {
  for (const life of lives) {
    if (life.t0 > frame) break
    if (vehicleVisibleAt(life, frame)) return life
  }
  return null
}

/**
 * vehicleDatedEnd — la frame de fin d'une vie quand le film l'ÉCRIT comme une destruction,
 * `null` sinon. Port exact de `vehicleDatedEnd` (Go) : c'est le seul instant de fin qui vaille.
 */
function vehicleDatedEnd(life: ReplayVehicleTrackReady): number | null {
  if (life.end !== VEHICLE_END_DESTROYED) return null
  return typeof life.tEnd === 'number' ? life.tEnd : null
}

/**
 * vehicleCycleRespawnAt — le compte à rebours de l'emplacement, ou `null`.
 *
 * DEUX SOURCES, DANS L'ORDRE QUE `respawnCountdown` impose à tout le rejeu : la prochaine
 * naissance VUE dans le film d'abord (le rejeu connaît la suite), la médiane du cycle depuis la
 * dernière fin DATÉE ensuite.
 *
 * IL NE S'APPELLE QUE SUR UN EMPLACEMENT LIBRE : c'est l'appelant qui l'a constaté
 * (`vehicleCycleOccupantAt`), et le vérifier deux fois ferait deux règles.
 */
export function vehicleCycleRespawnAt(
  cycle: VehicleCycle,
  lives: readonly ReplayVehicleTrackReady[],
  frame: number,
  frameMs: number,
): Respawn | null {
  let nextSpawn: number | null = null
  let emptySince: number | null = null
  for (const life of lives) {
    if (life.t0 > frame) {
      nextSpawn = life.t0
      break
    }
    const fin = vehicleDatedEnd(life)
    // LA DERNIÈRE FIN DATÉE ET PASSÉE, et pas la première : un emplacement qui a vu trois vies
    // mourir prédit depuis la troisième. Une fin postérieure à l'image courante appartient au
    // futur du film et ne dit rien de l'instant qu'on regarde.
    if (fin !== null && fin <= frame) emptySince = fin
  }
  return respawnAt(frame, frameMs, { nextSpawn, emptySince, medianS: cycle.medianS })
}

/** Ce qu'un emplacement dit de lui-même à une image : son occupant s'il y en a un, son compte. */
export interface VehicleCycleReading {
  occupant: ReplayVehicleTrackReady | null
  respawn: Respawn | null
}

/**
 * vehicleCycleReadingAt — la lecture complète d'un emplacement à une image.
 *
 * UN EMPLACEMENT OCCUPÉ N'A PAS DE COMPTE À REBOURS, et c'est le sens même du marqueur : rien
 * n'est attendu là où quelque chose se trouve déjà.
 */
export function vehicleCycleReadingAt(
  cycle: VehicleCycle,
  lives: readonly ReplayVehicleTrackReady[],
  frame: number,
  frameMs: number,
): VehicleCycleReading {
  const occupant = vehicleCycleOccupantAt(lives, frame)
  if (occupant) return { occupant, respawn: null }
  return { occupant: null, respawn: vehicleCycleRespawnAt(cycle, lives, frame, frameMs) }
}
