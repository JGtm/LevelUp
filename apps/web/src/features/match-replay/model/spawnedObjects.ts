/**
 * spawnedObjects.ts — LES OBJETS ENGENDRÉS PAR L'ÉQUIPEMENT PORTÉ, rendus dessinables : le
 * capteur, le traqueur, l'écran occultant et le champ de réparation, à l'endroit où ils se sont
 * posés.
 *
 * POURQUOI UNE SOURCE À PART. Le calque des poses ne dessine que `equipmentPlacements`
 * `deployed`, et depuis la règle du 15/09/2026 (`origineDeLaPose`, côté Go) seuls les panneaux du
 * mur le sont : un appareil porté qui tombe est un lâcher, pas un déploiement. Le capteur, le
 * traqueur, l'écran et le champ ne se dessinaient donc plus du tout, alors que le document porte
 * leur objet engendré avec sa vie lue (`carriedEquipment[].tracked[]`) et — depuis le schéma 105 —
 * sa POSITION DE REPOS (`land`, `x`, `y`, `z`, solidaires, lus sur la trajectoire de la même vie).
 *
 * CE QUE CE MODULE FAIT : fabriquer, de chaque objet engendré SITUÉ, une pose que le calque, la
 * révélation du capteur, la présence en zone et le survol savent déjà traiter. Une pose fabriquée
 * est une pose `deployed` de la famille, au repos, au nom du porteur — et rien de plus. Un objet
 * non situé ne produit RIEN : la position ne se devine pas (le porteur est réfuté par la mesure,
 * le vol médian va de 1,3 à 15 m).
 *
 * CE QU'IL NE FAIT PAS, ET C'EST UNE EXIGENCE (utilisateur, 2026-10-28) :
 *  - LE MUR ET LA FAILLE NE PASSENT PAS PAR ICI. Ils sont visibles et sonnent aujourd'hui par leur
 *    propre canal (`equipmentPlacements` pour les panneaux du mur, `riftPoses` pour la faille) :
 *    leurs objets engendrés existent aussi dans `tracked`, et les reprendre les dessinerait deux
 *    fois. La liste des familles est NOMMÉE (`SPAWNED_OBJECT_FAMILIES`) et un test interdit d'y
 *    voir une famille de `SPAWNED_OBJECT_OWN_CHANNEL_FAMILIES`.
 *  - LES SONS NE LISENT JAMAIS CES POSES. `replaySound.ts` lit `doc.equipmentPlacements`, que ce
 *    module ne modifie pas ; les quatre familles sonnent déjà par `carriedEquipment[].uses`, un
 *    second son serait un doublon. Seul `drawablePlacements` mélange les deux sources, et il ne
 *    sert qu'au dessin, à la révélation, aux zones et au survol.
 *
 * LA FENÊTRE DE CHAQUE POSE (décision du 2026-10-28, mesurée sur sept films témoins) :
 *  - capteur et champ de réparation : de `land` à `t1` LU (`endRead`) — la suppression de l'objet
 *    est écrite par le film, la durée officielle ne la prolonge pas. Sans `t1`, la règle existante
 *    de la famille (15 s pour le capteur, fin du rejeu pour le champ) ;
 *  - traqueur : une impulsion unique ancrée à `land` (règle existante de la famille). Son `t1` est
 *    la fin du PROJECTILE (2 images médianes), jamais celle de l'effet : il n'est pas lu ;
 *  - écran occultant : ancré à `land`, fin = règle existante (cf. `SHROUD_DURATION_MS`, durée
 *    inconnue). Son `t1` est de même la fin du projectile.
 *
 * Pas de React, pas de couleur, pas de canvas : de la lecture de document, testée seule.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { PLACEMENT_ORIGIN_DEPLOYED, placementIsDeployedObject } from '../layers/equipmentPlacementsLayer'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { WindowedPlacement } from './placementWindow'

/**
 * LES QUATRE FAMILLES dont l'objet engendré se dessine par cette source. Identifiants STABLES du
 * document (`abilityLabels[rang].family`), jamais des libellés.
 */
export const SPAWNED_OBJECT_FAMILIES = [
  'sensor',
  'threat_seeker',
  'shroud_screen',
  'repair_field',
] as const

export type SpawnedObjectFamily = (typeof SPAWNED_OBJECT_FAMILIES)[number]

/**
 * LES FAMILLES QUI ONT LEUR PROPRE CANAL ET QUE CETTE SOURCE NE REPREND JAMAIS : le mur (poses
 * `deployed` de ses panneaux) et la balise du translocateur (`riftPoses`, dessinée comme faille).
 */
export const SPAWNED_OBJECT_OWN_CHANNEL_FAMILIES = ['wall', 'translocator_beacon'] as const

/** Les familles dont le `t1` de l'objet engendré est SA suppression (et non la fin d'un projectile). */
const END_READ_FAMILIES: ReadonlySet<string> = new Set<SpawnedObjectFamily>(['sensor', 'repair_field'])

/** Identifiant d'une pose fabriquée : elle ne porte aucun tag `eqip`, et ne prétend pas en avoir. */
export const SPAWNED_OBJECT_ID = 'spawned'

type CarriedObject = ReplayDocumentReady['carriedEquipment'][number]
type Tracked = CarriedObject['tracked'][number]
/** Ce que ce module lit du document. */
type SpawnedSource = Pick<ReplayDocumentReady, 'carriedEquipment' | 'abilityLabels'>

/** Un objet engendré dont la position de repos est lue en entier (les quatre champs sont solidaires). */
type SituatedTracked = Tracked & { land: number; x: number; y: number; z: number }

function isSituated(t: Tracked): t is SituatedTracked {
  return (
    t.land !== undefined && t.x !== undefined && t.y !== undefined && t.z !== undefined
  )
}

/** La famille de l'objet porté de ce rang, si c'est l'une des quatre ; null sinon. */
function spawnedFamilyOf(
  labels: ReplayDocumentReady['abilityLabels'],
  rank: number,
): SpawnedObjectFamily | null {
  const family = labels?.[String(rank)]?.family
  return (SPAWNED_OBJECT_FAMILIES as readonly (string | undefined)[]).includes(family)
    ? (family as SpawnedObjectFamily)
    : null
}

/** Chaque objet engendré de chaque objet porté d'une des quatre familles, dans l'ordre du document. */
function* spawnedObjects(
  doc: SpawnedSource,
): Generator<{ family: SpawnedObjectFamily; slot: number; tracked: Tracked }> {
  for (const carried of doc.carriedEquipment) {
    const family = spawnedFamilyOf(doc.abilityLabels, carried.r)
    if (family === null) continue
    for (const tracked of carried.tracked) yield { family, slot: carried.slot, tracked }
  }
}

/**
 * spawnedObjectPlacements — les poses fabriquées des objets engendrés SITUÉS, dans l'ordre du
 * document. Le propriétaire est le slot du porteur, `t0` l'image de repos, la position celle de
 * repos. Le `t1` vaut la fin LUE (jamais avant le repos) quand la famille la porte et que le film
 * l'écrit ; sinon il vaut `t0` et la pose ne dit pas que sa fin est lue.
 */
export function spawnedObjectPlacements(doc: SpawnedSource): WindowedPlacement[] {
  const out: WindowedPlacement[] = []
  for (const { family, slot, tracked } of spawnedObjects(doc)) {
    if (!isSituated(tracked)) continue
    const end = END_READ_FAMILIES.has(family) ? tracked.t1 : undefined
    out.push({
      family,
      id: SPAWNED_OBJECT_ID,
      origin: PLACEMENT_ORIGIN_DEPLOYED,
      owner: slot,
      t0: tracked.land,
      t1: end === undefined ? tracked.land : Math.max(end, tracked.land),
      x: tracked.x,
      y: tracked.y,
      z: tracked.z,
      ...(end === undefined ? {} : { endRead: true }),
    })
  }
  return out
}

/** Les tableaux de dessin, calculés une fois par document (le calque les relit à chaque image). */
const DRAWABLE_BY_DOC = new WeakMap<object, readonly ReplayEquipmentPlacement[]>()

/**
 * drawablePlacements — TOUT CE QUE LE CALQUE DES POSES A À DESSINER : les poses du document
 * d'abord, dans leur ordre, puis les poses fabriquées. Sans objet engendré situé, c'est le tableau
 * du document lui-même.
 *
 * SEUL le dessin, la révélation du capteur, les zones et le survol lisent ce tableau. Les sons
 * continuent de lire `doc.equipmentPlacements`, que cette fonction ne modifie jamais.
 */
export function drawablePlacements(
  doc: Pick<ReplayDocumentReady, 'equipmentPlacements' | 'carriedEquipment' | 'abilityLabels'>,
): readonly ReplayEquipmentPlacement[] {
  const cached = DRAWABLE_BY_DOC.get(doc)
  if (cached) return cached
  const spawned = spawnedObjectPlacements(doc)
  const all = spawned.length === 0 ? doc.equipmentPlacements : [...doc.equipmentPlacements, ...spawned]
  DRAWABLE_BY_DOC.set(doc, all)
  return all
}

/** Un déploiement compté : la famille, le porteur, et l'image où l'objet apparaît (le lancer). */
export interface SpawnedDeployment {
  family: SpawnedObjectFamily
  owner: number
  t0: number
}

/**
 * spawnedObjectDeployments — UN OBJET ENGENDRÉ = UN DÉPLOIEMENT de sa famille, SITUÉ OU NON : le
 * geste a eu lieu même quand son lieu n'a pas été lu. Il se date à l'apparition de l'objet (le
 * lancer), là où le porteur est sans conteste le propriétaire de la vie.
 *
 * UNE FAMILLE QUE LES POSES DU DOCUMENT DÉPLOIENT DÉJÀ N'EST PAS RECOMPTÉE : les artefacts
 * antérieurs à la règle du 15/09/2026 portent encore des poses `deployed` pour ces appareils, et
 * les compter deux fois doublerait le geste.
 */
export function spawnedObjectDeployments(
  doc: Pick<ReplayDocumentReady, 'equipmentPlacements' | 'carriedEquipment' | 'abilityLabels'>,
): SpawnedDeployment[] {
  const countedElsewhere = new Set(
    doc.equipmentPlacements.filter(placementIsDeployedObject).map((p) => p.family),
  )
  const out: SpawnedDeployment[] = []
  for (const { family, slot, tracked } of spawnedObjects(doc)) {
    if (countedElsewhere.has(family)) continue
    out.push({ family, owner: slot, t0: tracked.t0 })
  }
  return out
}
