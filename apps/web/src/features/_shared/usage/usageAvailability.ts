/**
 * usageAvailability.ts — LE VOCABULAIRE DES ÉTATS VIDES des formes partagées (D8, 2026-09-21) :
 * une cause, un titre court et une phrase par cause — « aucune donnée » ne distinguait aucune
 * d'entre elles.
 *
 * LA PHRASE SE CHOISIT À L'AFFICHAGE (`usageEmptyMessage`, `usageEmptyTitle`) : c'est l'appelant
 * qui sait pourquoi son bloc est vide.
 */
import type { UsageText } from './usageI18n'

/**
 * LES TROIS CAUSES D'UN BLOC VIDE (D8) — elles ne se corrigent pas de la même façon, donc
 * elles ne s'écrivent pas de la même façon :
 *
 *   - `no-film`       : des matchs, aucun film décodé (la mesure n'a pas eu lieu) ;
 *   - `no-objectives` : des films lus, mais aucun mode à objectif dans la sélection ;
 *   - `load-failed`   : la lecture a échoué — transitoire, donc il faut le dire.
 */
export type UsageEmptyReason = 'no-film' | 'no-objectives' | 'load-failed'

export function usageEmptyMessage(reason: UsageEmptyReason, t: UsageText): string {
  switch (reason) {
    case 'no-film':
      return t.emptyNoFilm
    case 'no-objectives':
      return t.emptyNoObjectives
    case 'load-failed':
      return t.unavailableLoadFailed
  }
}

/**
 * LE TITRE COURT de l'état vide (2026-09-22) — la première ligne de `EmptyStateNotice`,
 * l'état vide canonique de l'app. `usageEmptyMessage` en reste la seconde : une cause, un
 * titre ET une phrase, jamais un titre générique recollé devant des phrases distinctes.
 */
export function usageEmptyTitle(reason: UsageEmptyReason, t: UsageText): string {
  switch (reason) {
    case 'no-film':
      return t.emptyTitleNoFilm
    case 'no-objectives':
      return t.emptyTitleNoObjectives
    case 'load-failed':
      return t.emptyTitleLoadFailed
  }
}
