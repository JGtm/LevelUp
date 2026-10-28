/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : TOUTE FAMILLE QUI A UNE RÈGLE DE DESSIN SE DESSINE SUR SON TÉMOIN RÉEL.
 *
 * POURQUOI. Pendant des semaines le capteur, le traqueur, l'écran occultant et le champ de
 * réparation sont restés dans `PLACEMENT_RENDER` avec leur forme, leurs tests verts et leur
 * infobulle, sans qu'une seule pose n'atteigne l'écran : les fixtures étaient des poses
 * `deployed` écrites à la main, que le document réel ne publie plus. Ce test remplace la
 * fixture par un extrait RÉEL (`test/spawnedObjectWitnesses.ts`) et exige, par famille, au moins
 * une pose dessinée à l'écran par le chemin que le calque emprunte (`drawablePlacements` pour les
 * poses, `riftStations` pour la faille, qui se dessine dans le même calque sans être une pose).
 *
 * MUTATION VÉRIFIÉE (2026-10-28) : `drawablePlacements` rendant `doc.equipmentPlacements` seul
 * fait rougir les quatre familles engendrées ; le mur et la faille, eux, restent verts — c'est
 * exactement leur non-régression.
 *
 * LA FAMILLE `other` (règle `unnamed`) EST EXEMPTÉE : c'est un outil de diagnostic, éteint par
 * défaut, qui ne désigne aucun objet nommé, et les sept documents témoins du schéma 105 n'en
 * portent aucune pose réelle.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { riftStations } from '../model/riftStations'
import { drawablePlacements } from '../model/spawnedObjects'
import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'
import { SPAWNED_WITNESSES, type SpawnedWitnessFamily } from '../test/spawnedObjectWitnesses'
import {
  isPlacementActive,
  PLACEMENT_RENDER,
  placementKind,
  type PlacementToggles,
} from './equipmentPlacementsLayer'

const TOGGLES: PlacementToggles = { showUnnamed: false, showDropped: true }
const RIFT_FAMILY = 'translocator_beacon'

/** Familles à règle de dessin d'objet nommé (hors `null` : décidé de ne rien dessiner ; hors `unnamed`). */
const DRAWN_FAMILIES = Object.entries(PLACEMENT_RENDER)
  .filter(([, kind]) => kind !== null && kind !== 'unnamed')
  .map(([family]) => family)

/** Combien de poses (ou de failles) de la famille sont à l'écran à leur premier instant. */
function drawnCount(doc: ReplayDocumentReady, family: string): number {
  if (family === RIFT_FAMILY) return riftStations(doc).length
  const time = { frameMs: 100, frames: doc.frameCount }
  return drawablePlacements(doc).filter((p) => {
    if (p.family !== family) return false
    const kind = placementKind(p, TOGGLES)
    return kind !== null && isPlacementActive(p, kind, p.t0, time)
  }).length
}

describe('garde-rail : un témoin réel par famille à règle de dessin', () => {
  it('chaque famille à règle de dessin a un témoin, et chaque témoin une famille à règle', () => {
    expect(Object.keys(SPAWNED_WITNESSES).sort()).toEqual([...DRAWN_FAMILIES].sort())
  })

  it.each(Object.keys(SPAWNED_WITNESSES) as SpawnedWitnessFamily[])(
    '%s : au moins une pose dessinée sur son extrait réel',
    (family) => {
      expect(drawnCount(SPAWNED_WITNESSES[family](), family)).toBeGreaterThan(0)
    },
  )

  it('les poses du mur et de la faille viennent de leur canal, pas de la source des objets engendrés', () => {
    const wall = SPAWNED_WITNESSES.wall()
    expect(drawablePlacements(wall)).toBe(wall.equipmentPlacements)
    // Leurs objets engendrés (`tracked`) ne sont jamais repris : cf. `spawnedObjects.test.ts`.
    const rift = SPAWNED_WITNESSES.translocator_beacon()
    expect(drawablePlacements(rift)).toBe(rift.equipmentPlacements)
  })
})

describe('garde-rail : les sons ne lisent jamais les poses fabriquées', () => {
  it('aucun module de sound/ n’importe la source des objets engendrés', () => {
    const sons = sourcesDeLaFeature().filter((f) => cheminCourt(f).startsWith('sound/'))
    // Un garde qui ne regarderait aucun fichier resterait vert sans rien tenir.
    expect(sons.length).toBeGreaterThan(10)
    const fautifs = sons.filter((f) => /spawnedObjects|drawablePlacements/.test(lire(f))).map(nomDe)
    expect(fautifs).toEqual([])
  })
})

/**
 * LE CÂBLAGE : seuls les sons, la source des objets engendrés et le compte d'usage lisent encore
 * `doc.equipmentPlacements`. Tout autre module (canvas, fiches, hook des poses, survol, zones)
 * passe par `drawablePlacements` : y remettre le tableau du document seul est la régression
 * d'origine (le capteur, le traqueur, l'écran et le champ ne s'affichent plus).
 */
const LECTEURS_AUTORISES = new Set([
  'sound/replaySound.ts', // les sons : le mur seul sonne par ses poses, les quatre familles par `uses`
  'model/spawnedObjects.ts', // la source elle-même
  'model/equipmentUsageLogic.ts', // les poses déployées du document, puis `spawnedObjectDeployments`
])

/** Le code sans ses commentaires : un commentaire peut nommer le canal sans le lire. */
function sansCommentaires(src: string): string {
  return src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

describe('garde-rail : le câblage des poses passe par drawablePlacements', () => {
  it('aucun module hors liste nommée ne lit doc.equipmentPlacements', () => {
    const sources = sourcesDeLaFeature()
    // Les modules à câbler existent bien : un garde qui ne les verrait pas resterait vert.
    for (const nom of ['ReplayCanvas.tsx', 'ReplayTeams.tsx', 'useReplayPlacements.ts']) {
      expect(sources.map(nomDe)).toContain(nom)
    }
    const fautifs = sources
      .filter((f) => !LECTEURS_AUTORISES.has(cheminCourt(f)))
      .filter((f) => sansCommentaires(lire(f)).includes('doc.equipmentPlacements'))
      .map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('les trois consommateurs de dessin et de zones lisent la source des objets engendrés', () => {
    const lus = (nom: string) => sansCommentaires(lire(sourcesDeLaFeature().find((f) => nomDe(f) === nom)!))
    expect(lus('useReplayPlacements.ts')).toMatch(/drawablePlacements\(doc\)/)
    expect(lus('ReplayTeams.tsx')).toMatch(/drawablePlacements\(doc\)/)
    expect(lus('ReplayCanvas.tsx')).toMatch(/placements\.all/)
  })
})
