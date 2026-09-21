/**
 * timeseriesCoordination.logic — les décisions PURES des deux cartes de coordination des
 * Séries temporelles (lot Q, D22-3 et D22-6/7 du 2026-09-21).
 *
 * UN SEUL GRAPHE PAR SUJET (D22-3) : les deux grandeurs d'un sujet partagent l'axe des %
 * et se lisent l'une contre l'autre. Ce module ne fait que PROJETER le bloc Coordination
 * servi par la page (`TimeseriesPageResponse.coordination`, lot N1) sur la frise partagée
 * — aucun quotient n'est recalculé ici : les taux, leurs bruts et leur drapeau
 * d'échantillon sont mesurés côté Go.
 *
 * Deux règles tiennent dans ce module, et un test peut les mettre en défaut sans monter
 * d'arbre React :
 *
 *   1. une soirée à ÉCHANTILLON FAIBLE reste sur l'axe, en bâton creux — la trame du
 *      temps ne doit pas mentir ;
 *   2. un repère de comparaison ABSENT (parité non mesurée : aucun match à effectif
 *      connu) ne se remplace pas par une valeur par défaut, il ne se dessine pas.
 */
import type {
  CoordinationBlock,
  CoordinationSessionPoint,
  Couverture,
} from '@/lib/api/types'
import { shortSessionLabel } from '@/components/charts/sessionBarsTrendChart'

/** Une grandeur, soirée par soirée, prête pour la frise partagée. */
export interface SerieDeSoirees {
  /** Taux par soirée, en POURCENTS (`null` = soirée sans mesure). */
  valuesPct: (number | null)[]
  /** Par soirée : bâton creux (échantillon faible). */
  hollow: boolean[]
  /** Dénominateur de la soirée — rendu en infobulle, jamais dans le taux. */
  volumes: number[]
}

/** Vrai quand le bloc porte de quoi dessiner : disponible ET au moins une soirée. */
export function coordinationDessinable(block: CoordinationBlock | undefined): boolean {
  return block?.available === true && (block.sessions ?? []).length > 0
}

/** Les soirées du bloc, dans l'ordre servi (le service les rend du plus ancien au plus récent). */
export function soireesDe(block: CoordinationBlock | undefined): CoordinationSessionPoint[] {
  return block?.sessions ?? []
}

/** Les libellés d'axe : la DATE de chaque soirée, pas sa plage horaire. */
export function labelsDeSoirees(sessions: CoordinationSessionPoint[]): string[] {
  return sessions.map((s) => shortSessionLabel(s.session_label))
}

/**
 * serieDeSoirees projette une couverture par soirée.
 *
 * Une soirée SANS dénominateur (`n === 0`) rend `null` : un taux de 0 % y serait une
 * invention — rien n'a été mesuré, ce n'est pas « jamais arrivé ».
 */
export function serieDeSoirees(
  sessions: CoordinationSessionPoint[],
  choisir: (p: CoordinationSessionPoint) => Couverture,
): SerieDeSoirees {
  const mesures = sessions.map(choisir)
  return {
    valuesPct: mesures.map((c) => (c.n > 0 ? c.taux * 100 : null)),
    hollow: mesures.map((c) => c.echantillon_faible),
    volumes: mesures.map((c) => c.n),
  }
}

/** Un taux 0..1 en POURCENTS — l'unité du seul axe Y de la frise. */
export function enPourcents(c: Couverture): number {
  return c.taux * 100
}

/**
 * pariteOuRien rend la part équitable 1/n en POURCENTS, ou `null`.
 *
 * Le serveur laisse `parity_pct` absent quand aucun match du périmètre ne porte
 * l'effectif de camp (film non décodé) : pas de repère plutôt qu'un repère inventé.
 */
export function pariteOuRien(pct: number | undefined): number | null {
  return typeof pct === 'number' && pct > 0 ? pct : null
}

/** Le délai médian des ripostes, en SECONDES, ou `null` si aucune riposte mesurée. */
export function delaiMedianS(ms: number | undefined): number | null {
  return typeof ms === 'number' && ms > 0 ? ms / 1000 : null
}
