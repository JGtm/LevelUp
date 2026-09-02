/**
 * useReplayZoom — L'ÉTAT DU CADRAGE : de combien on grossit, et sur quoi on est centré.
 *
 * # CE QU'IL NE FAIT PAS, ET C'EST L'ESSENTIEL
 *
 * Il ne projette rien. Il ne connaît ni la toile, ni les pixels, ni le dessin. Il tient DEUX
 * valeurs — un palier et un point du monde — et garantit qu'elles restent légales. C'est
 * `visibleBounds` qui en tire une fenêtre, et `useReplayView` qui en tire une projection.
 *
 * Cette frontière n'est pas de la coquetterie : elle est ce qui rend le zoom testable sans
 * canvas, et ce qui interdit qu'une deuxième façon de dire « où l'on regarde » apparaisse.
 *
 * # LE CENTRE EST TOUJOURS LÉGAL, PAS SEULEMENT À L'AFFICHAGE
 *
 * `visibleBounds` reborne déjà ce qu'on lui donne : on pourrait donc stocker n'importe quoi et
 * laisser l'affichage corriger. Ce serait un piège — l'état dirait une chose, l'écran une autre,
 * et le prochain calcul repartirait de la valeur fausse. Chaque écriture passe donc par
 * `clampCenter`, et l'état ne contient jamais une position qu'on ne peut pas montrer.
 *
 * Le cas qui le rend nécessaire : DÉZOOMER depuis un coin. La fenêtre s'élargit, elle ne tient
 * plus aussi près du bord, et le centre doit reculer. Sans rebornage à l'écriture, il resterait
 * dans un coin où plus rien ne peut le suivre.
 *
 * # PAS DE GLISSER, ET LA RAISON EST MESURABLE
 *
 * La demande laissait le choix (« soit le déplacement à la souris soit une croix
 * directionnelle »). La croix gagne pour deux raisons. L'horloge tourne : glisser pendant que
 * l'action bouge, c'est poursuivre le jeu à la souris. Et surtout, un glisser change le cadrage
 * à CHAQUE mouvement de pointeur — donc recuit les quatre calques statiques à chaque image. Il
 * faudrait un blit décalé et une cuisson à marge pour l'éviter. La croix va par pas discrets :
 * une cuisson par clic, exactement le coût d'un redimensionnement de fenêtre.
 */
import { useCallback, useMemo, useState } from 'react'

import type { ReplayBounds } from '@/lib/api/types'

import { ZOOM_LEVELS, clampCenter, sceneCenter, type ZoomLevel } from './replayLogic'

/**
 * Le pas de la croix, en fraction de la fenêtre VISIBLE (et non de la scène). Un quart : assez
 * pour avancer franchement, assez peu pour garder un repère commun entre avant et après — un pas
 * d'une fenêtre entière ferait perdre le fil de ce qu'on regardait.
 */
export const PAN_STEP_RATIO = 0.25

export interface ReplayZoom {
  level: ZoomLevel
  center: { x: number; y: number }
  canZoomIn: boolean
  canZoomOut: boolean
  /** `false` à 1x : la fenêtre vaut la scène, il n'existe qu'une position légale. */
  canPan: boolean
  zoomIn: () => void
  zoomOut: () => void
  reset: () => void
  /** Un pas de croix. `dx`/`dy` valent -1, 0 ou 1 ; `dy` positif va vers le HAUT de la carte. */
  panStep: (dx: number, dy: number) => void
}

export function useReplayZoom(scene: ReplayBounds): ReplayZoom {
  const [index, setIndex] = useState(0)
  // LE CENTRE N'EST PAS INITIALISÉ DEPUIS `scene` PAR UN EFFET, mais par `null` puis résolu à la
  // lecture : la scène arrive après le premier rendu (elle dépend du document et du fond de
  // carte retenu), et un effet d'initialisation aurait posé un centre sur une scène vide avant
  // de le corriger — un saut de cadrage à l'arrivée des données.
  const [raw, setRaw] = useState<{ x: number; y: number } | null>(null)

  const level = ZOOM_LEVELS[index]
  const center = useMemo(
    () => (raw ? clampCenter(scene, level, raw.x, raw.y) : sceneCenter(scene)),
    [raw, scene, level],
  )

  const zoomIn = useCallback(() => {
    setIndex((i) => Math.min(i + 1, ZOOM_LEVELS.length - 1))
  }, [])

  // EN DÉZOOMANT, LE CENTRE SE REBORNE — cf. l'en-tête. Il se calcule à partir du centre
  // COURANT (déjà légal) et du palier D'ARRIVÉE, jamais du palier qu'on quitte.
  const zoomOut = useCallback(() => {
    setIndex((i) => {
      const next = Math.max(i - 1, 0)
      setRaw((prev) => {
        const from = prev ?? sceneCenter(scene)
        return clampCenter(scene, ZOOM_LEVELS[next], from.x, from.y)
      })
      return next
    })
  }, [scene])

  const reset = useCallback(() => {
    setIndex(0)
    setRaw(null)
  }, [])

  const panStep = useCallback(
    (dx: number, dy: number) => {
      setRaw((prev) => {
        const from = prev ?? sceneCenter(scene)
        const stepX = ((scene.maxX - scene.minX) / level) * PAN_STEP_RATIO
        const stepY = ((scene.maxY - scene.minY) / level) * PAN_STEP_RATIO
        return clampCenter(scene, level, from.x + dx * stepX, from.y + dy * stepY)
      })
    },
    [scene, level],
  )

  return {
    level,
    center,
    canZoomIn: index < ZOOM_LEVELS.length - 1,
    canZoomOut: index > 0,
    canPan: level > 1,
    zoomIn,
    zoomOut,
    reset,
    panStep,
  }
}
