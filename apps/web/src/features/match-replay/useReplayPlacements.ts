/**
 * useReplayPlacements — TOUT CE QUE LE CANVAS A BESOIN DE SAVOIR DES POSES D'ÉQUIPEMENT :
 * combien il y en a à dessiner, sur quel axe de temps, avec quelles bascules, et laquelle est
 * sous le pointeur.
 *
 * POURQUOI CE FICHIER EXISTE, ET CE N'EST PAS UN CHOIX DE GOÛT. `ReplayCanvas.tsx` porte un
 * CLIQUET de taille (`placementFamily.guard.test.ts`) : il ne remonte jamais, et le franchir
 * se corrige en EXTRAYANT, pas en relevant le nombre. Le lot des objets lâchés ajoutait une
 * bascule, un comptage et un argument de survol — quatrième fois que ce cliquet impose une
 * découpe, après `useReplayInks`, `useSlotIdentity` et `useReplayTiming`. Les trois morceaux
 * réunis ici étaient déjà voisins dans le canvas et parlent de la même chose : les poses.
 *
 * CE QU'IL NE FAIT PAS : dessiner. Le tracé reste dans la boucle du canvas
 * (`drawEquipmentPlacementsLayer`), parce qu'il doit s'intercaler entre deux autres calques à
 * un endroit précis de la pile. Ce hook rend ce que la boucle CONSOMME.
 *
 * LES MORCEAUX SONT RENDUS SÉPARÉMENT, ET C'EST DÉLIBÉRÉ : `hover` change à chaque mouvement
 * de pointeur, et le mettre dans les dépendances de `draw` recuirait toute la scène pour une
 * infobulle (même règle que `useReplayWeaponPads`, dont le canvas ne met que le tracé). Les
 * appelants dépendent de `counts.drawable`, `windowTime` et `toggles` — trois valeurs stables.
 */
import { useMemo, type RefObject } from 'react'

import {
  countDrawablePlacements,
  type PlacementToggles,
  type PlacementView,
  type PlacementWindowTime,
} from './equipmentPlacementsLayer'
import { riftTeleports, type RiftTeleport } from './placementTeleport'
import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { usePlacementHover, type PlacementHoverHandlers } from './usePlacementHover'

export interface ReplayPlacementsInput {
  doc: ReplayDocumentReady
  /** Le cadrage PARTAGÉ du canvas : le survol et le dessin lisent la même projection. */
  view: PlacementView
  /** L'image courante, telle que la boucle de lecture la tient. */
  frameRef: RefObject<number>
  /** Bascule du calque entier. Faux = rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  /** Bascule des objets non identifiés (famille `other`). ÉTEINTE par défaut. */
  showUnnamed: boolean
  /**
   * Bascule des objets de PUISSANCE lâchés — le réglage du tiroir, tel quel. Une garde de
   * mode le croisait jusqu'au 2026-08-20 pour l'annuler en Fiesta ; elle a été retirée. Ni ce
   * hook ni le calque n'ont jamais connu la notion de mode, et le document n'en publie aucun.
   */
  showDropped: boolean
}

export interface ReplayPlacements {
  /** Ce que le document donne à dessiner, compté par la porte du tracé (`placementKind`). */
  counts: { drawable: number; unnamed: number; dropped: number }
  /** L'axe de temps qui borne la fenêtre d'une pose (cf. `placementEndFrame`). */
  windowTime: PlacementWindowTime
  /** Les deux bascules, mémoïsées : elles entrent dans les dépendances du tracé. */
  toggles: PlacementToggles
  /** La pose sous le curseur et ses gestionnaires de pointeur. */
  hover: PlacementHoverHandlers
  /** Les passages par une faille du film (cf. `placementTeleport`). */
  teleports: readonly RiftTeleport[]
}

export function useReplayPlacements({
  doc,
  view,
  frameRef,
  enabled,
  showUnnamed,
  showDropped,
}: ReplayPlacementsInput): ReplayPlacements {
  // LES COMPTES PASSENT PAR LA MÊME PORTE QUE LE TRACÉ : une commande du tiroir ne s'allume
  // que si quelque chose se dessinerait derrière elle (même règle que le bouton Zones).
  const counts = useMemo(
    () => countDrawablePlacements(doc.equipmentPlacements),
    [doc.equipmentPlacements],
  )

  // L'axe de temps : `frameMs` est la durée RÉELLE d'une image (le ping du capteur bat en
  // temps de match, pas en nombre d'images), `frames` la borne des poses sans fin connue.
  const windowTime = useMemo(
    () => ({ frameMs: frameToMs(1, doc), frames: doc.frameCount }),
    [doc],
  )

  // Les PASSAGES par une faille : un balayage de toutes les pistes, donc calculé une fois par
  // document et jamais par image. Il vit ici et non dans le composant pour la même raison que
  // les comptes : c'est une lecture des poses, et le composant n'a pas à la refaire.
  const teleports = useMemo(
    () => riftTeleports(doc.equipmentPlacements, doc.tracks, doc.abilities),
    [doc.equipmentPlacements, doc.tracks, doc.abilities],
  )

  const toggles = useMemo<PlacementToggles>(
    () => ({ showUnnamed, showDropped }),
    [showUnnamed, showDropped],
  )

  const hover = usePlacementHover({
    placements: doc.equipmentPlacements,
    view,
    time: windowTime,
    frameRef,
    enabled: enabled && counts.drawable > 0,
    showUnnamed,
    showDropped,
  })

  return { counts, windowTime, toggles, hover, teleports }
}
