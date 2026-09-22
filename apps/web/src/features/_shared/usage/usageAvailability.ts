/**
 * usageAvailability.ts — LA DISPONIBILITÉ du bloc « usages d'équipement, armes spéciales et
 * objectifs » : la distinction entre les DEUX raisons de ne rien avoir (règle des deux portes,
 * 2026-09-05, registre L4) — et, depuis le 2026-09-21, LE VOCABULAIRE DES ÉTATS VIDES (D8).
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 *
 * ÉLARGI le 2026-09-09 (E5.8, PLAN_EQUIPEMENT_GACHIS_2026-09-09) : `SessionUsageBlock`
 * (page Sessions) et `EquipmentUsageBlock` (Synthèse/Escouade) portent la MÊME forme
 * `available` / `unavailable_reason` / `matches_measured` — la même règle des deux
 * portes s'applique aux deux, donc le même helper, typé structurellement plutôt que
 * dupliqué (CLAUDE.md n°6).
 *
 * LA PORTE REND UNE CAUSE, PLUS UNE PHRASE (2026-09-21) : la phrase se choisit à l'affichage
 * (`usageEmptyMessage`), parce que les appelants ont désormais DEUX causes de plus à dire que
 * la porte ne connaît pas — une sélection sans socle (Super Fiesta) et une sélection sans
 * objectif ne sont pas des indisponibilités du bloc.
 */
import type { UsageText } from './usageI18n'

/** Le sous-ensemble commun à `SessionUsageBlock` et `EquipmentUsageBlock`. */
export interface UsageAvailabilityLike {
  available: boolean
  unavailable_reason?: string
  matches_measured: number
}

/**
 * LES QUATRE CAUSES D'UN BLOC VIDE (D8) — elles ne se corrigent pas de la même façon, donc
 * elles ne s'écrivent pas de la même façon :
 *
 *   - `no-film`       : des matchs, aucun film décodé (la mesure n'a pas eu lieu) ;
 *   - `no-pads`       : des films lus, mais le mode n'allume aucun socle (Super Fiesta) ;
 *   - `no-objectives` : des films lus, mais aucun mode à objectif dans la sélection ;
 *   - `load-failed`   : la lecture a échoué — transitoire, donc il faut le dire.
 *
 * « Aucune donnée » ne distingue aucun de ces quatre cas, et c'est précisément la phrase
 * qu'il a fallu remplacer.
 */
export type UsageEmptyReason = 'no-film' | 'no-pads' | 'no-objectives' | 'load-failed'

export function usageEmptyMessage(reason: UsageEmptyReason, t: UsageText): string {
  switch (reason) {
    case 'no-film':
      return t.emptyNoFilm
    case 'no-pads':
      return t.emptyNoPads
    case 'no-objectives':
      return t.emptyNoObjectives
    case 'load-failed':
      return t.unavailableLoadFailed
  }
}

export type UsageAvailability =
  | { kind: 'ok' }
  | { kind: 'hidden' }
  | { kind: 'empty'; reason: UsageEmptyReason }

/**
 * usageAvailability — l'état du bloc, et la distinction entre les DEUX raisons de ne
 * rien avoir (règle des deux portes, 2026-09-05, registre L4) :
 *
 *   - absent du payload (vieux serveur) → RIEN, pas de bloc fantôme ;
 *   - `unsupported` → RIEN NON PLUS : le TITRE ne publie pas de résumé d'usage (pas de
 *     décodeur de film, donc pas d'artefact, donc rien à résumer — jamais). Une carte
 *     « Ce titre ne publie pas de résumé d'usage des films » était un bloc mort : elle
 *     occupait une place, ne disait rien d'actionnable, et ne disparaîtrait jamais ;
 *   - `load_failed` → état vide AVEC la raison : le titre sait le produire, c'est CETTE
 *     lecture-là qui a échoué. Transitoire, donc il faut le dire ;
 *   - aucun match mesuré → état vide « aucun film » (les objectifs, au scope indépendant
 *     des films, restent affichables par l'appelant).
 */
export function usageAvailability(usage: UsageAvailabilityLike | null | undefined): UsageAvailability {
  const kind = usageAvailabilityKind(usage)
  if (kind === 'hidden') return { kind: 'hidden' }
  if (kind === 'ok') return { kind: 'ok' }
  // `empty` a DEUX causes ici : la lecture a échoué (le titre sait produire), ou le titre
  // produit mais aucun match de ce scope n'a de film mesuré.
  const reason: UsageEmptyReason = usage != null && !usage.available ? 'load-failed' : 'no-film'
  return { kind: 'empty', reason }
}

/**
 * usageAvailabilityKind — LA MÊME DÉCISION QUE `usageAvailability`, SANS LA CAUSE.
 *
 * Existe parce qu'un appelant peut avoir besoin de savoir si le bloc va rendre QUELQUE
 * CHOSE sans avoir de dictionnaire sous la main (le titre de section « Frags et usages »
 * de la page Sessions, qui ne doit pas se poser au-dessus de rien). Dupliquer la règle des
 * deux portes pour ça l'aurait fait diverger : `usageAvailability` DÉLÈGUE ici, c'est le
 * seul endroit où la décision est écrite.
 */
export function usageAvailabilityKind(
  usage: UsageAvailabilityLike | null | undefined,
): UsageAvailability['kind'] {
  if (usage == null) return 'hidden'
  if (!usage.available) return usage.unavailable_reason === 'unsupported' ? 'hidden' : 'empty'
  if (usage.matches_measured <= 0) return 'empty'
  return 'ok'
}
