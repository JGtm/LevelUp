/**
 * vies.logic — QUELLES APPARITIONS FORMENT LES GRAPPES DE RÉAPPARITION.
 *
 * Même vocabulaire que le paramètre `vies` du contrat (`TacticalRasterBody.vies`) :
 *   - `depart` : la première vie seule, une par match (la lecture d'avant la bascule) ;
 *   - `toutes` : toutes les apparitions du joueur — première vie, réapparitions, débuts de manche.
 *     Avec une grappe choisie, la lecture des trajets ne compte que les vies parties de cette grappe.
 */
import { lectureDeRejeu, type TacticalQuestion } from './tacticalView.logic'

export type TacticalVies = 'depart' | 'toutes'

export const VIES_PAR_DEFAUT: TacticalVies = 'depart'

/** L'ordre des boutons de la bascule. */
export const VIES_VALEURS: readonly TacticalVies[] = ['depart', 'toutes']

/**
 * La valeur postée : le défaut ne s'envoie pas. Le corps d'une lecture « Départ » — donc sa clé de
 * cache — reste celui d'avant la bascule, et rien ne change tant qu'on n'y touche pas.
 */
export function viesPostees(vies: TacticalVies): TacticalVies | undefined {
  return vies === VIES_PAR_DEFAUT ? undefined : vies
}

/**
 * La bascule est proposée dès que la lecture peut servir des grappes : une lecture d'artefact (temps,
 * trajets), des grappes déjà servies, ou un réglage déjà posé — qu'il faut pouvoir ôter. Sous
 * « Départ », une carte peut n'avoir aucune grappe alors qu'« Toutes les vies » en aurait : la bascule
 * ne dépend donc pas des seules grappes servies.
 */
export function viesProposees(params: {
  question: TacticalQuestion
  nbGrappes: number
  spawn: string
  vies: TacticalVies
}): boolean {
  return params.nbGrappes > 0 || params.spawn !== '' || params.vies !== VIES_PAR_DEFAUT || lectureDeRejeu(params.question)
}
