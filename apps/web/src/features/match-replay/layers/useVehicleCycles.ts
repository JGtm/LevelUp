/**
 * useVehicleCycles — LES EMPLACEMENTS DE NAISSANCE DE VÉHICULE (`useReplayVehicles`) : l'appariement
 * emplacement -> vies, le nom d'un emplacement et son SURVOL (le seul de ce calque).
 *
 * Extrait de `useReplayVehicles` pour tenir la règle des 80 lignes par fonction. Même règle de lecture
 * que le reste du calque : l'image courante se lit dans une référence, jamais dans un état React.
 */
import { useCallback, useMemo, useState, type PointerEvent, type RefObject } from 'react'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { vehicleCycleLives, vehicleCycleReadingAt, type VehicleCycle } from '../model/vehicleCycleTime'
import { vehicleCycleIndexAt } from './vehicleCyclesLayer'
import type { PlacementView } from './placementShapes'
import type { VehicleCycleHover } from './useReplayVehicles'

interface VehicleCyclesInput {
  cycles: ReplayDocumentReady['vehicleCycles']
  tracks: ReplayDocumentReady['vehicles']
  view: PlacementView
  frameRef: RefObject<number>
  /** Durée réelle d'une frame, en ms. */
  frameMs: number
  /** Faux quand le calque est éteint : rien ne se survole. */
  enabled: boolean
  labelOfFamily: (family: string) => string | null
  /** Le titre générique d'un emplacement qui ne porte aucune famille. */
  cycleTitle: string
}

export function useVehicleCycles({
  cycles, tracks, view, frameRef, frameMs, enabled, labelOfFamily, cycleTitle,
}: VehicleCyclesInput) {
  // L'APPARIEMENT EMPLACEMENT -> VIES, UNE SEULE FOIS PAR DOCUMENT. C'est un balayage de toutes
  // les vies pour chaque emplacement (la maille de l'amas, cf. `vehicleCycleTime`) : le refaire
  // soixante fois par seconde serait un doublon de travail, et le refaire au survol un doublon de
  // source. Le tableau est PARALLÈLE à `cycles` — c'est pourquoi le test de survol rend un RANG.
  const cycleLives = useMemo(
    () => cycles.map((cycle) => vehicleCycleLives(cycle, tracks)),
    [cycles, tracks],
  )
  const livesOf = useCallback((index: number) => cycleLives[index] ?? [], [cycleLives])

  // LE NOM D'UN EMPLACEMENT : la famille DOMINANTE telle que le document la nomme, sa clé à
  // défaut (un nom de véhicule est un nom propre du jeu), et le titre générique quand
  // l'emplacement ne porte aucune famille — jamais le nom d'un voisin.
  const nameOfCycle = useCallback(
    (cycle: VehicleCycle): string =>
      cycle.family ? (labelOfFamily(cycle.family) ?? cycle.family) : cycleTitle,
    [labelOfFamily, cycleTitle],
  )

  const [cycleHover, setCycleHover] = useState<VehicleCycleHover | null>(null)

  /**
   * LE SURVOL D'UN EMPLACEMENT DE NAISSANCE — le seul survol de ce calque (les véhicules
   * eux-mêmes n'en portent aucun).
   *
   * MÊME AMORCE QUE CELLE DES SOCLES ET DES POSES : le rapport pixels CSS -> pixels du contexte
   * se CALCULE plutôt qu'il ne se suppose (la mise en page peut remettre le canevas à l'échelle).
   */
  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || cycles.length === 0 || view.width === 0) {
        setCycleHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const i = vehicleCycleIndexAt(cycles, view, window.devicePixelRatio || 1, at)
      setCycleHover((prev) => {
        if (i < 0) return prev === null ? prev : null
        const cycle = cycles[i]
        const lu = vehicleCycleReadingAt(cycle, cycleLives[i] ?? [], frameRef.current, frameMs)
        const next: VehicleCycleHover = {
          cycle,
          at,
          name: nameOfCycle(cycle),
          occupied: lu.occupant !== null,
          respawn: lu.respawn,
        }
        // LE COMPTE SE COMPARE CHAMP À CHAMP, jamais par identité : la lecture construit un objet
        // neuf à chaque appel, et l'infobulle se recréerait à chaque mouvement de pointeur.
        if (
          prev &&
          prev.cycle === next.cycle &&
          prev.occupied === next.occupied &&
          prev.respawn?.seconds === next.respawn?.seconds &&
          prev.respawn?.measured === next.respawn?.measured &&
          prev.at.x === at.x &&
          prev.at.y === at.y
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, cycles, view, cycleLives, frameRef, frameMs, nameOfCycle],
  )

  const onPointerLeave = useCallback(() => {
    setCycleHover((prev) => (prev === null ? prev : null))
  }, [])

  return { livesOf, cycleHover, onPointerMove, onPointerLeave }
}
