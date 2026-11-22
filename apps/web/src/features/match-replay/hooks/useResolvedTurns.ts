/**
 * useResolvedTurns — L'ORIENTATION EFFECTIVE de la carte du rejeu : le choix retenu s'il y en a un, sinon la
 * règle Auto (`useAutoTurned`, `lib/replay`) appliquée à la carte et à la boîte que l'écran offre à la toile.
 *
 * `useReplayView` est son SEUL appelant : tout ce qui dessine ou projette (cadrage, zones nommées,
 * calques, mini-repères, Suivre) lit l'orientation effective qu'il rend, et personne ne recalcule
 * l'Auto de son côté. Rien n'est retenu ni remonté : l'Auto n'écrit pas dans les préférences, elle se
 * réévalue à chaque mesure stabilisée de la boîte (le délai de `useReplayViewport`).
 *
 * L'état de l'hystérésis de la règle est RANGÉ AVEC SA PORTÉE (`scopeKey`, un match) : un autre match repart
 * du plan droit, il ne reprend pas l'orientation du précédent.
 *
 * PENDANT UN EXPORT, l'Auto est GELÉE : la sortie reste celle d'avant l'export, une fenêtre que l'on
 * redimensionne ne retourne pas le plan qu'on enregistre. À la fin de l'export, la règle reprend sur la
 * mesure du moment, avec l'état d'avant.
 *
 * Sans boîte mesurée (largeur ou hauteur nulle) la règle rend 0 : le plan reste droit jusqu'à la mesure.
 */
import type { ReplayBounds } from '@/lib/api/types'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { useAutoTurned } from '@/lib/replay/useAutoTurned'

import { useExportActive } from '../export/exportLayoutStore'

export interface ResolvedTurns {
  /** L'orientation effective, en quarts de tour horaires : celle que tout le reste lit. */
  turns: QuarterTurns
  /** Vrai quand c'est l'Auto qui a tourné le plan (jamais pour un choix retenu). */
  autoTurned: boolean
}

/**
 * @param scopeKey  la portée de l'état retenu : un match. Une autre portée repart du plan droit.
 * @param choice  le choix retenu pour la carte ; `null` = aucun choix : la règle Auto décide.
 * @param bounds  les bornes de la scène dans le repère monde, avant toute rotation.
 * @param width  la largeur que l'écran offre à la toile, en px CSS.
 * @param freeHeight  la hauteur que l'écran offre à la toile, en px CSS.
 */
export function useResolvedTurns(
  scopeKey: string,
  choice: QuarterTurns | null,
  bounds: ReplayBounds,
  width: number,
  freeHeight: number,
): ResolvedTurns {
  const exportActive = useExportActive()
  const autoTurned = useAutoTurned({
    scopeKey,
    choice,
    carte: { largeur: bounds.maxX - bounds.minX, hauteur: bounds.maxY - bounds.minY },
    boite: { largeur: width, hauteur: freeHeight },
    gele: exportActive,
  })
  return { turns: choice ?? (autoTurned ? 1 : 0), autoTurned }
}
