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
} from './equipmentPlacementsLayer'
import { seekerImpulseActive } from './placementShapes'
import { SENSOR_DURATION_MS } from './threatSensor'

/**
 * placementEndFrame — LA DERNIÈRE IMAGE à laquelle une pose se dessine.
 *
 * `t1` N'EST PAS LA DISPARITION DE L'OBJET, et c'est mesuré, pas supposé (2026-08-18,
 * `filmdec/equipment_life_end_test.go`) : le décodeur ne suit que les records qui portent une
 * position, si bien que `t1` date l'instant où l'objet cesse de BOUGER. Un mur de protection y
 * atteint 0,7 s de médiane et un capteur 2,1 s, quand le recensement des keyframes montre
 * 101 de ces 295 objets encore présents plus d'une seconde après. Le film ne date AUCUNE
 * disparition d'équipement — le record de suppression et la queue de records sans position
 * sont l'un et l'autre du bruit au témoin. `t1` est donc une BORNE INFÉRIEURE.
 *
 * D'OÙ LA RÈGLE, en deux temps et sans rien inventer :
 *  - une famille dont la durée est PUBLIÉE par l'éditeur s'y tient — le capteur de menaces
 *    dure 15 s (cf. SENSOR_DURATION_MS), du même genre que sa portée et sa cadence, qui
 *    gouvernent déjà tout son tracé ;
 *  - les autres restent affichées jusqu'à la fin du rejeu. Effacer un mur à 0,7 s
 *    affirmerait une disparition que rien ne mesure ; le laisser en place n'affirme rien.
 *
 * Jamais AVANT `t1` : la borne mesurée l'emporte si elle dépasse la durée officielle.
 */
export function placementEndFrame(
  p: ReplayEquipmentPlacement,
  kind: PlacementKind,
  time: PlacementWindowTime,
): number {
  const lastFrame = Math.max(time.frames - 1, p.t1)
  if (kind !== 'sensor' || !(time.frameMs > 0)) return lastFrame
  const official = p.t0 + Math.round(SENSOR_DURATION_MS / time.frameMs)
  return Math.min(Math.max(official, p.t1), lastFrame)
}

/** isPlacementActive — la pose se dessine-t-elle à cette image ? (cf. placementEndFrame). */
export function isPlacementActive(
  p: ReplayEquipmentPlacement,
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
