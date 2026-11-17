/**
 * useResolvedTurns — L'ORIENTATION EFFECTIVE de la carte : le choix retenu s'il y en a un, sinon la
 * règle Auto (`autoTurns`, `lib/replay/orientation`) appliquée à la carte et à la boîte que l'écran
 * offre à la toile.
 *
 * `useReplayView` est son SEUL appelant : tout ce qui dessine ou projette (cadrage, zones nommées,
 * calques, mini-repères, Suivre) lit l'orientation effective qu'il rend, et personne ne recalcule
 * l'Auto de son côté. Rien n'est retenu ni remonté : l'Auto n'écrit pas dans les préférences, elle se
 * réévalue à chaque mesure stabilisée de la boîte (le délai de `useReplayViewport`).
 *
 * L'HYSTÉRÉSIS DE LA RÈGLE a besoin de savoir si le plan est DÉJÀ tourné par l'Auto : ce hook en retient
 * l'état, calculé pendant le rendu (jamais depuis un effet, qui ferait peindre une image à la
 * mauvaise orientation). Un choix retenu l'efface : une carte qui repasse en Auto repart du plan droit.
 *
 * Sans boîte mesurée (largeur ou hauteur nulle) la règle rend 0 : le plan reste droit jusqu'à la mesure.
 */
import { useState } from 'react'

import type { ReplayBounds } from '@/lib/api/types'
import { autoTurns, type QuarterTurns } from '@/lib/replay/orientation'

export interface ResolvedTurns {
  /** L'orientation effective, en quarts de tour horaires : celle que tout le reste lit. */
  turns: QuarterTurns
  /** Vrai quand c'est l'Auto qui a tourné le plan (jamais pour un choix retenu). */
  autoTurned: boolean
}

/**
 * @param choice  le choix retenu pour la carte ; `null` = aucun choix : la règle Auto décide.
 * @param bounds  les bornes de la scène dans le repère monde, avant toute rotation.
 * @param width  la largeur que l'écran offre à la toile, en px CSS.
 * @param freeHeight  la hauteur que l'écran offre à la toile, en px CSS.
 */
export function useResolvedTurns(
  choice: QuarterTurns | null,
  bounds: ReplayBounds,
  width: number,
  freeHeight: number,
): ResolvedTurns {
  const [turnedBefore, setTurnedBefore] = useState(false)
  const turnedNow =
    choice === null &&
    autoTurns(
      { largeur: bounds.maxX - bounds.minX, hauteur: bounds.maxY - bounds.minY },
      { largeur: width, hauteur: freeHeight },
      turnedBefore,
    ) === 1
  if (turnedNow !== turnedBefore) setTurnedBefore(turnedNow)
  return { turns: choice ?? (turnedNow ? 1 : 0), autoTurned: turnedNow }
}
