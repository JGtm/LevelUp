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
 * # TROIS GESTES, UN SEUL CHEMIN
 *
 * La croix, la molette (`useReplayWheelZoom`) et le clavier (`useReplayShortcuts`) appellent tous
 * les MÊMES actions, avec les mêmes paliers et le même rebornage. C'est ce qui interdit qu'un
 * zoom à la molette et un zoom au bouton finissent par ne plus donner le même résultat.
 *
 * LE GLISSER N'Y EST PAS, et la raison est mesurable : il change le cadrage à CHAQUE mouvement
 * de pointeur, donc recuit les quatre calques statiques à chaque image (le sol fait ~45 000
 * cellules, 10 à 45 ms par cuisson). L'éviter demande un blit recadré et une cuisson à marge —
 * un lot en soi, chiffré, pas encore fait. Les trois gestes ci-dessus vont par pas DISCRETS :
 * une cuisson par cran, exactement le coût d'un redimensionnement de fenêtre.
 */
import { useCallback, useMemo, useState } from 'react'

import type { ReplayBounds } from '@/lib/api/types'

import { ZOOM_LEVELS, clampCenter, sceneCenter, zoomTowards, type ZoomLevel } from '../../../lib/replay/replayLogic'

/** Un indice de palier, ramene dans la liste. Un seul endroit ou cette borne s ecrit. */
function clampIndex(i: number): number {
  return Math.min(Math.max(i, 0), ZOOM_LEVELS.length - 1)
}

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
  /**
   * UN CRAN DE MOLETTE, VERS UN POINT DU MONDE. `dir` vaut +1 (grossir) ou -1 (réduire) ;
   * `towards` est le point sous le pointeur, qui doit rester IMMOBILE à l'écran.
   *
   * Sans ce second argument, la molette ne pourrait grossir que vers le centre de la fenêtre —
   * c'est-à-dire déplacer sous la souris exactement ce qu'on visait avec elle. C'est la
   * différence entre une molette qui attrape et une molette qui chasse.
   */
  zoomAt: (dir: number, towards: { x: number; y: number }) => void
  /** Déplacement en UNITÉS MONDE. La croix et le glisser s'y ramènent tous les deux. */
  panBy: (dx: number, dy: number) => void
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
    setIndex((i) => clampIndex(i + 1))
  }, [])

  // EN DÉZOOMANT, LE CENTRE SE REBORNE — cf. l'en-tête. Il se calcule à partir du centre
  // COURANT (déjà légal) et du palier D'ARRIVÉE, jamais du palier qu'on quitte.
  const zoomOut = useCallback(() => {
    setIndex((i) => {
      const next = clampIndex(i - 1)
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

  // LE DÉPLACEMENT EN UNITÉS MONDE est le primitif ; la croix et le glisser s'y ramènent tous
  // les deux. Le hook ne connaît donc toujours pas les pixels — c'est le geste qui convertit.
  const panBy = useCallback(
    (dx: number, dy: number) => {
      setRaw((prev) => {
        const from = prev ?? sceneCenter(scene)
        return clampCenter(scene, level, from.x + dx, from.y + dy)
      })
    },
    [scene, level],
  )

  const panStep = useCallback(
    (dx: number, dy: number) => {
      panBy(
        dx * ((scene.maxX - scene.minX) / level) * PAN_STEP_RATIO,
        dy * ((scene.maxY - scene.minY) / level) * PAN_STEP_RATIO,
      )
    },
    [panBy, scene, level],
  )

  // LA MOLETTE PASSE PAR LE MÊME CHEMIN QUE LES BOUTONS — mêmes paliers, même rebornage. Elle
  // n'ajoute qu'une chose : le point à garder immobile. C'est ce qui interdit qu'un « zoom à la
  // molette » et un « zoom au bouton » finissent par ne plus donner le même résultat.
  const zoomAt = useCallback(
    (dir: number, towards: { x: number; y: number }) => {
      setIndex((i) => {
        const next = clampIndex(dir > 0 ? i + 1 : i - 1)
        if (next === i) return i
        setRaw((prev) => {
          const from = prev ?? sceneCenter(scene)
          const moved = zoomTowards(from, towards, ZOOM_LEVELS[i], ZOOM_LEVELS[next])
          return clampCenter(scene, ZOOM_LEVELS[next], moved.x, moved.y)
        })
        return next
      })
    },
    [scene],
  )

  return {
    level,
    center,
    zoomAt,
    canZoomIn: index < ZOOM_LEVELS.length - 1,
    canZoomOut: index > 0,
    canPan: level > 1,
    zoomIn,
    zoomOut,
    reset,
    panStep,
    panBy,
  }
}
