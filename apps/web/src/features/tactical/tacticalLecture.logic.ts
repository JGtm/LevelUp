/**
 * tacticalLecture.logic — l'ÉTAT de la lecture du plan et ce qu'on en affiche, en logique
 * PURE (retours rejeu L2, 2026-09-23).
 *
 * LE DÉFAUT QU'IL FERME : changer de question ou de filtre créait une clé de cache sans
 * donnée ; la vue lisait `isPending` et DÉMONTAIT tout son corps, fond de carte compris,
 * puis le reconstruisait. Depuis, les lectures gardent leur réponse précédente
 * (`placeholderData`) et la vue distingue quatre états — c'est ce module qui les nomme :
 *
 *   - `attente`   : aucune donnée encore (premier chargement) — le cadre et le fond sont
 *                   posés, l'indicateur PAR-DESSUS ;
 *   - `relecture` : la réponse affichée est la PRÉCÉDENTE (placeholder, ou périmètre en
 *                   cours de relecture) — calque et KPI ESTOMPÉS sous « Mise à jour… »
 *                   (décision Q26 du 2026-09-23) ;
 *   - `echec`     : la lecture OU la résolution de son périmètre a échoué — le message
 *                   d'échec, le fond reste, rien de périmé (revue L2-R1 : un périmètre en
 *                   échec suspend le raster, qui gardait son placeholder « Mise à jour… »
 *                   pour toujours) — et, pour la même raison, la composition IMPOSSIBLE
 *                   (coéquipier introuvable, contrôle L2-PARC-1), que la vue nomme ;
 *   - `pret`      : la réponse affichée répond à la demande courante.
 *
 * ESTOMPER, C'EST TOUT CE QUI VIENT DE LA RÉPONSE PRÉCÉDENTE (revue L2-R6/R7) : KPI, calque,
 * légende, pied, messages d'état, cartes Cellule et Coordination — et, sur l'écran d'entrée,
 * la grille des cartes. Une seule classe (`ESTOMPE`), un seul helper (`classeRelecture`).
 */
import type { MapFrame } from '@/lib/replay/heatPaint'

import {
  PLAN_ASPECT_DEFAUT,
  repereAspect,
  type RepereTactique,
  type TacticalQuestion,
} from './tacticalView.logic'

export type TacticalEtatLecture = 'attente' | 'relecture' | 'echec' | 'pret'

/** Classe d'estompage d'une réponse PRÉCÉDENTE pendant la relecture (décision Q26). */
export const ESTOMPE = 'opacity-50'

/** La classe d'un bloc tiré de la réponse affichée : estompé pendant une relecture seulement. */
export function classeRelecture(enRelecture: boolean): string {
  return enRelecture ? `transition-opacity ${ESTOMPE}` : 'transition-opacity'
}

/**
 * etatLecture — l'état de la lecture du plan.
 *
 * L'ÉCHEC PRIME : une réponse gardée d'une lecture qui vient d'échouer ne se présente pas
 * comme courante. Puis l'absence de données (premier chargement), puis la relecture.
 */
export function etatLecture(entree: {
  aDesDonnees: boolean
  enEchec: boolean
  surPlaceholder: boolean
  perimetreEnRelecture: boolean
}): TacticalEtatLecture {
  if (entree.enEchec) return 'echec'
  if (!entree.aDesDonnees) return 'attente'
  if (entree.surPlaceholder || entree.perimetreEnRelecture) return 'relecture'
  return 'pret'
}

const QUESTIONS: ReadonlySet<string> = new Set<TacticalQuestion>([
  'morts',
  'kills',
  'gagne',
  'temps',
  'routes',
  'isole',
])

/**
 * questionServie — la question À LAQUELLE LA RÉPONSE AFFICHÉE RÉPOND (champ `question` du
 * contrat), jamais la question demandée : pendant une relecture, la réponse affichée est
 * celle de l'ANCIENNE question, et une légende, une unité ou une source calculées depuis la
 * nouvelle mentiraient. Une valeur hors vocabulaire (réponse d'une autre version) retombe
 * sur la question demandée.
 */
export function questionServie(servie: string, demandee: TacticalQuestion): TacticalQuestion {
  return QUESTIONS.has(servie) ? (servie as TacticalQuestion) : demandee
}

/**
 * aspectDuPlan — le rapport largeur/hauteur du cadre du plan.
 *
 * Le repère quand la lecture en donne un ; sinon le CALAGE DU FOND seul — c'est le cas du
 * premier chargement, où le cadre doit déjà avoir la taille qu'il gardera (le repère d'une
 * carte à fond EST son calage, donc le rapport ne saute pas à l'arrivée de la réponse) ;
 * sinon le rapport par défaut.
 */
export function aspectDuPlan(fond: MapFrame | null, repere: RepereTactique | null): number {
  if (repere) return repereAspect(repere)
  if (fond && fond.widthM > 0 && fond.heightM > 0) return fond.widthM / fond.heightM
  return PLAN_ASPECT_DEFAUT
}
