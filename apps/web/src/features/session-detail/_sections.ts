/**
 * _sections — modele des SECTIONS d'une colonne de session (D16).
 *
 * Source UNIQUE de l'ordre et de la composition des blocs empiles sous le L3 :
 * consommee par `SessionColumnBody` (qui associe une cle a un rendu) ET par
 * `SessionDetailPage` (qui fusionne les cles des deux colonnes pour composer des
 * RANGEES partagees en mode comparaison). Deux listes divergentes rendraient
 * l'alignement gauche/droite faux sans que rien ne casse — d'ou la centralisation.
 */

export type SessionSectionKey =
  | 'summary'
  | 'outcomes_kills'
  | 'mode_placement'
  | 'fda_radars'
  | 'netscore_fda'
  | 'fda_gap'
  | 'net_lives'
  | 'intensity'
  | 'first_blood'
  | 'coordination'
  | 'range'
  | 'participation'
  | 'mmr_ocdr'
  | 'perf'
  | 'engagement'
  | 'damage'
  | 'frags'
  | 'career_xp'
  | 'usage'
  | 'matches'

/** Ordre canonique d'affichage, de haut en bas. */
export const SESSION_SECTION_ORDER: readonly SessionSectionKey[] = [
  'summary',
  'outcomes_kills',
  'mode_placement',
  'fda_radars',
  'netscore_fda',
  'fda_gap',
  'net_lives',
  'intensity',
  'first_blood',
  'coordination',
  'range',
  'participation',
  'mmr_ocdr',
  'perf',
  'engagement',
  'damage',
  'frags',
  'career_xp',
  'usage',
  'matches',
]

export interface SessionSectionInput {
  /**
   * Bloc `coordination` (lot N1) servi pour CETTE colonne. Le contrat ne sert PAS de
   * miroir `compare_coordination` : la colonne comparée rend donc toujours le placeholder
   * D16 sur cette rangée — c'est une absence de donnée, pas un oubli de câblage.
   */
  hasCoordination: boolean
  /** Bloc `range_profiles` / `compare_range_profiles` (lot N2) servi pour cette colonne. */
  hasRange: boolean
  /**
   * Bloc « usages d'equipement, armes speciales et objectifs » servi par le payload
   * pour CETTE colonne. Absent (vieux serveur, session sans film decode) → la section
   * n'existe pas de ce cote ; en comparaison l'autre colonne rend un placeholder.
   */
  hasUsage: boolean
}

/** Cles effectivement presentes pour une colonne, dans l'ordre canonique. */
export function sessionSectionKeys({
  hasUsage,
  hasCoordination,
  hasRange,
}: SessionSectionInput): SessionSectionKey[] {
  const present: Record<string, boolean> = {
    usage: hasUsage,
    coordination: hasCoordination,
    range: hasRange,
  }
  return SESSION_SECTION_ORDER.filter((key) => present[key] ?? true)
}

/**
 * Union ordonnee des cles de deux colonnes — une rangee par cle : chaque cote rend
 * sa section ou, s'il ne l'a pas, le placeholder « Sans equivalent dans cette session ».
 */
export function mergeSessionSectionKeys(
  left: readonly SessionSectionKey[],
  right: readonly SessionSectionKey[],
): SessionSectionKey[] {
  return SESSION_SECTION_ORDER.filter((key) => left.includes(key) || right.includes(key))
}
