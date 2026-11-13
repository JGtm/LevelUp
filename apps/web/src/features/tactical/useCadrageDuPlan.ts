/**
 * useCadrageDuPlan — le cadrage du plan de l'onglet Tactique : la SCÈNE (le repère de la lecture,
 * sinon le calage du fond), l'état du zoom du rejeu 2D (`useReplayZoom` : paliers 1x à 3x, centre
 * reborné) et la FENÊTRE visible qui en sort (`visibleBounds`). Tout ce qui se pose sur le fond s'y
 * projette (`TacticalPlanCalque`, `cadrageDuFond`).
 *
 * LA CARTE TOURNÉE (`turns`, l'orientation retenue pour la carte) : le repère reçu est déjà celui de la
 * carte tournée (`repereDuPlan`) ; sans lecture, le calage du fond est tourné ici. Changer d'orientation
 * ramène le zoom à la carte entière (règle partagée, `useZoomResetOnTurns`).
 */
import { useMemo } from 'react'

import { useReplayZoom } from '@/features/match-replay/hooks/useReplayZoom'
import type { ReplayBounds } from '@/lib/api/types'
import type { MapFrame } from '@/lib/replay/heatPaint'
import { orientBounds, type QuarterTurns } from '@/lib/replay/orientation'
import { visibleBounds } from '@/lib/replay/replayLogic'
import { useZoomResetOnTurns } from '@/lib/replay/useZoomResetOnTurns'

import type { RepereTactique } from './tacticalView.logic'

/** Une scène neutre tant que ni le fond ni la lecture ne disent où est la carte. */
const SCENE_NEUTRE: ReplayBounds = { minX: 0, maxX: 1, minY: 0, maxY: 1, minZ: 0, maxZ: 0 }

/** useCadrageDuPlan — la scène, le zoom et la fenêtre visible du plan. */
export function useCadrageDuPlan(repere: RepereTactique | null, cadreFond: MapFrame | null, turns: QuarterTurns = 0) {
  const scene = useMemo<ReplayBounds>(() => {
    if (repere) return { minX: repere.minX, maxX: repere.maxX, minY: repere.minY, maxY: repere.maxY, minZ: 0, maxZ: 0 }
    if (cadreFond && cadreFond.widthM > 0 && cadreFond.heightM > 0) {
      return orientBounds(
        {
          minX: cadreFond.originX,
          maxX: cadreFond.originX + cadreFond.widthM,
          minY: cadreFond.originY - cadreFond.heightM,
          maxY: cadreFond.originY,
          minZ: 0,
          maxZ: 0,
        },
        turns,
      )
    }
    return SCENE_NEUTRE
  }, [repere, cadreFond, turns])
  const zoom = useReplayZoom(scene)
  useZoomResetOnTurns(turns, zoom.reset)
  const fenetre = useMemo<ReplayBounds>(
    () => visibleBounds(scene, zoom.level, zoom.center.x, zoom.center.y),
    [scene, zoom.level, zoom.center],
  )
  return { scene, zoom, fenetre }
}

