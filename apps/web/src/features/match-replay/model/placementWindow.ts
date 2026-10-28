/**
 * placementWindow.ts — DE QUAND A QUAND une pose d'equipement est a l'ecran.
 *
 * EXTRAIT DE `equipmentPlacementsLayer.ts` LE 2026-08-18 (lot R2-V), qui portait deja une dette
 * de taille (503 lignes) et gagnait la chaine de cap du mur. La decoupe tombe sur la seule
 * question de ce fichier : le TEMPS. Ce qui reste au calque, c'est la decision de rendu et le
 * trace ; ce qui vient ici, c'est la fenetre — et elle se raisonne sans connaitre une seule
 * forme.
 *
 * Le calque REEXPORTE ces trois fonctions : elles font partie de sa surface publique (le survol
 * et les tests les lisent), et la decoupe interne n'a pas a remonter jusqu'a ses appelants.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import type {
  PlacementHoverTime,
  PlacementKind,
  PlacementWindowTime,
} from '../layers/equipmentPlacementsLayer'
import { SHROUD_DURATION_MS, seekerImpulseActive } from '../layers/placementShapes'
import { WALL_DURATION_MS } from '../layers/placementWall'
import { SENSOR_DURATION_MS } from './threatSensor'

/**
 * LES DURÉES DE REPLI, par famille : celles qu'une pose SANS FIN LUE se voit appliquer. Une famille absente de
 * cette table n'a aucune durée connue : elle reste affichée jusqu'à la fin du rejeu, faute de mieux.
 *
 * Les valeurs vivent dans le fichier de LEUR famille (`threatSensor.ts`, `placementWall.ts`,
 * `placementShapes.ts`), avec leur portée et leur géométrie, et cette table ne fait que les rassembler pour la
 * fenêtre. Le capteur tient la durée publiée par l'éditeur ; le mur et l'écran occultant celle que le film
 * mesure (médiane de la barrière, durée de la bulle) — et le film, quand il date leur fin, l'emporte.
 */
const OFFICIAL_DURATION_MS: Partial<Record<PlacementKind, number>> = {
  sensor: SENSOR_DURATION_MS,
  wall: WALL_DURATION_MS,
  shroud: SHROUD_DURATION_MS,
}

/**
 * Une pose dont la FIN peut être LUE. `endRead` ne se pose que sur une pose fabriquée depuis un
 * objet engendré dont le film écrit la suppression (`spawnedObjects.ts` : capteur, champ de
 * réparation, bulle de l'écran occultant) : son `t1` est alors la fin de l'objet, pas la borne inférieure
 * d'une fin inconnue. Les poses DU DOCUMENT portent la leur dans `lifeEnd` (le mur, schéma 106).
 *
 * Le type voyage sous celui des poses du document (`ReplayEquipmentPlacement`) : les tableaux du
 * calque restent typés ainsi, et le drapeau se lit à l'exécution, ici et nulle part ailleurs.
 */
export type WindowedPlacement = ReplayEquipmentPlacement & { readonly endRead?: boolean }

/**
 * placementEndFrame — LA DERNIÈRE IMAGE à laquelle une pose se dessine.
 *
 * `t1` N'EST PAS LA DISPARITION DE L'OBJET, et c'est mesuré, pas supposé (2026-08-18,
 * `filmdec/equipment_life_end_test.go`) : le décodeur ne suit que les records qui portent une
 * position, si bien que `t1` date l'instant où l'objet cesse de BOUGER. Un mur de protection y
 * atteint 0,7 s de médiane et un capteur 2,1 s, quand le recensement des keyframes montre
 * 101 de ces 295 objets encore présents plus d'une seconde après. Pour les poses du document, le
 * film ne date AUCUNE disparition d'équipement — le record de suppression et la queue de records
 * sans position y sont l'un et l'autre du bruit au témoin. `t1` est donc une BORNE INFÉRIEURE
 * (sauf pour les poses à fin lue, première règle ci-dessous).
 *
 * D'OÙ LA RÈGLE, en trois temps et sans rien inventer :
 *  - une pose à FIN LUE s'arrête à cette fin, que la durée de repli ne prolonge pas : la lecture l'emporte sur
 *    toute durée supposée. C'est `endRead` (objet engendré dont le film écrit la suppression : son `t1`) pour
 *    une pose fabriquée, `lifeEnd` (suppression de l'objet que la pose a créé : le mur) pour une pose du document ;
 *  - une famille dont la durée est connue s'y tient FAUTE DE FIN LUE — le capteur de menaces dure
 *    15 s (éditeur), le mur 12,48 s (médiane mesurée), l'écran 7,5 s (lue dans le film) : cf.
 *    OFFICIAL_DURATION_MS ;
 *  - les autres restent affichées jusqu'à la fin du rejeu, faute d'une durée à leur opposer.
 *
 * LE MUR A REJOINT LE CAPTEUR le 2026-08-20 (demande utilisateur). Il restait jusqu'à la fin
 * du rejeu au nom du principe « ne rien affirmer que la mesure ne porte » — mais laisser un
 * mur à l'écran pendant huit minutes affirme lui aussi quelque chose, et de plus faux : la
 * durée officielle est la meilleure information disponible, exactement comme pour le capteur.
 *
 * Jamais AVANT `t1` : la borne mesurée l'emporte si elle dépasse la durée officielle.
 */
export function placementEndFrame(
  p: WindowedPlacement,
  kind: PlacementKind,
  time: PlacementWindowTime,
): number {
  if (p.endRead) return p.t1
  const lastFrame = Math.max(time.frames - 1, p.t1)
  if (p.lifeEnd !== undefined) return Math.min(Math.max(p.lifeEnd, p.t0), lastFrame)
  const duration = OFFICIAL_DURATION_MS[kind]
  if (duration === undefined || !(time.frameMs > 0)) return lastFrame
  const official = p.t0 + Math.round(duration / time.frameMs)
  return Math.min(Math.max(official, p.t1), lastFrame)
}

/** isPlacementActive — la pose se dessine-t-elle à cette image ? (cf. placementEndFrame). */
export function isPlacementActive(
  p: WindowedPlacement,
  kind: PlacementKind,
  frame: number,
  time: PlacementWindowTime,
): boolean {
  return frame >= p.t0 && frame <= placementEndFrame(p, kind, time)
}

/**
 * placementShows — la pose a-t-elle quelque chose À L'ÉCRAN à cette image ?
 *
 * Toutes les familles montrent quelque chose sur toute leur fenêtre d'affichage (cf.
 * placementEndFrame), SAUF le traqueur :
 * son impulsion est unique et brève, et après elle il ne reste rien. Le survol suit le dessin —
 * on n'inspecte pas ce qui n'est pas là, même règle que la bascule des objets non identifiés.
 */
export function placementShows(
  p: ReplayEquipmentPlacement,
  kind: PlacementKind,
  time: PlacementHoverTime,
): boolean {
  if (kind !== 'seeker') return true
  return seekerImpulseActive((time.frame - p.t0) * time.frameMs)
}
