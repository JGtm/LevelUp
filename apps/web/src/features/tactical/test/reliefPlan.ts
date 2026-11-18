/**
 * test/reliefPlan — LE RELIEF D'UNE CARTE FACTICE pour les tests des plans de l'onglet Tactique : trois étages
 * (0, 2,2 et 4,4) ou une carte sans maillage, la frame qu'un calculateur rendrait, et deux zones nommées à hauteurs
 * distinctes. Rien ici n'asserte.
 */
import type { TacticalRaster } from '@/lib/api/types'
import { parseReliefData, type ReliefFrame } from '@/lib/replay/relief'
import { rawRelief, sceneCalibration } from '@/lib/replay/relief/reliefScene.testutil'

/** Les données d'un relief de 40 x 10 pixels : trois blocs d'étages, ou aucun maillage. */
export function donneesDuRelief(sansMaillage = false) {
  const rects = sansMaillage
    ? []
    : [
        { i0: 0, j0: 0, i1: 9, j1: 9, z: 0 },
        { i0: 10, j0: 0, i1: 19, j1: 9, z: 2.2 },
        { i0: 20, j0: 0, i1: 29, j1: 9, z: 4.4 },
      ]
  return parseReliefData(
    rawRelief({
      width: 40,
      height: 10,
      levels: sansMaillage ? [] : [0, 2.2, 4.4],
      rects,
      steps: [[0, 50, 390, 0]],
      meshShare: sansMaillage ? undefined : 0.46,
      degradations: sansMaillage ? ['sans-maillage'] : [],
    }),
  )
}

/** Le sidecar d'un fond : son calage, une image plan, et un relief quand `relief`. */
export function sidecarDuRelief(relief: boolean) {
  return {
    calibration: { ...sceneCalibration(40, 10), convention: 'test' },
    planImage: 'plan.webp',
    ...(relief ? { reliefImage: 'r.webp', reliefData: 'r.json' } : {}),
  }
}

/** La frame qu'un calculateur rendrait : l'image teintée, aucun masque, des traits vides. */
export function frameDuRelief(): ReliefFrame {
  const empty = { points: new Float32Array(0), offsets: new Uint32Array(1) }
  return { width: 40, height: 10, cut: null, image: new Uint8ClampedArray(4), alpha: null, planAlpha: null, lines: { steps: empty, contours: empty, walls: empty } }
}

/** Une toile de la taille du relief, que `ReliefCanvases` rendrait. */
export function canvasDuRelief(): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = 40
  canvas.height = 10
  return canvas
}

/** Deux zones nommées : « Base » de 0 à 2 et « Pont » de 4 à 6 (le Bas et l'Étage 2 de la carte factice). */
export const ZONES_A_HAUTEUR: NonNullable<TacticalRaster['zones']> = [
  { name: '', fr: 'Base', en: 'Base', x: 20, y: 40, z: 1, z_bottom: 0, z_top: 2, volume_index: 1, big: true, polygon: [[10, 30], [30, 30], [30, 50], [10, 50]] },
  { name: '', fr: 'Pont', en: 'Bridge', x: 60, y: 10, z: 5, z_bottom: 4, z_top: 6, volume_index: 2, polygon: [[50, 0], [70, 0], [70, 20], [50, 20]] },
]
