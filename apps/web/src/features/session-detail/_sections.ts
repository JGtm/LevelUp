/**
 * _sections — modele des SECTIONS d'une colonne de session (D16).
 *
 * Source UNIQUE de l'ordre et de la composition des blocs empiles sous le L3 :
 * consommee par `SessionColumnBody` (qui associe une cle a un rendu) ET par
 * `SessionDetailPage` (qui fusionne les cles des deux colonnes pour composer des
 * RANGEES partagees en mode comparaison). Deux listes divergentes rendraient
 * l'alignement gauche/droite faux sans que rien ne casse — d'ou la centralisation.
 */

import type { SessionManifestKey } from '@/lib/i18n/generated/session'

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
  'career_xp',
  'frags',
  'usage',
  'matches',
]

/**
 * LES QUATRE SECTIONS TITREES de la page session (chantier « sections transverses »,
 * 2026-09-22). Un groupe coiffe PLUSIEURS cles : le titre se pose une fois, au-dessus de
 * la premiere cle PRESENTE du groupe, et nulle part si le groupe est entierement absent —
 * un titre au-dessus de rien annoncerait une mesure qui n'existe pas.
 *
 * `summary` (la bande KPI) n'a PAS de groupe : c'est l'en-tete de la colonne, pas une
 * section de plus — meme choix que l'accueil. `matches` pose son propre titre avec son
 * tableau (`space-y-3` : un tableau se colle plus a son titre que des graphes).
 */
export type SessionSectionGroup = 'overview' | 'match_by_match' | 'kills_usage'

/** Cle de manifeste du titre de chaque groupe. */
export const SESSION_GROUP_TITLE_KEY: Record<SessionSectionGroup, SessionManifestKey> = {
  overview: 'session.detail.section_overview',
  match_by_match: 'session.detail.section_match_by_match',
  kills_usage: 'session.detail.section_kills_usage',
}

/** Groupe de chaque cle — les cles absentes de cette table ne sont coiffees par rien. */
export const SESSION_SECTION_GROUPS: Partial<Record<SessionSectionKey, SessionSectionGroup>> = {
  outcomes_kills: 'overview',
  mode_placement: 'overview',
  fda_radars: 'overview',
  netscore_fda: 'match_by_match',
  fda_gap: 'match_by_match',
  net_lives: 'match_by_match',
  intensity: 'match_by_match',
  first_blood: 'match_by_match',
  coordination: 'match_by_match',
  range: 'match_by_match',
  participation: 'match_by_match',
  mmr_ocdr: 'match_by_match',
  perf: 'match_by_match',
  engagement: 'match_by_match',
  damage: 'match_by_match',
  career_xp: 'match_by_match',
  frags: 'kills_usage',
  usage: 'kills_usage',
}

/**
 * Les cles d'une liste RENDUE, groupees dans l'ordre. Une seule lecture pour les deux
 * rendus de `SessionColumnBody` (pile pleine page, rangees de comparaison) : la place du
 * titre ne peut pas diverger de l'un a l'autre.
 */
export interface SessionSectionRun {
  /** Groupe coiffant ces cles, ou `null` pour les cles sans titre (`summary`, `matches`). */
  group: SessionSectionGroup | null
  keys: SessionSectionKey[]
}

export function groupSessionSections(
  keys: readonly SessionSectionKey[],
): SessionSectionRun[] {
  const runs: SessionSectionRun[] = []
  for (const key of keys) {
    const group = SESSION_SECTION_GROUPS[key] ?? null
    const last = runs[runs.length - 1]
    if (last && last.group === group && group != null) last.keys.push(key)
    else runs.push({ group, keys: [key] })
  }
  return runs
}

export interface SessionSectionInput {
  /**
   * Bloc `coordination` / `compare_coordination` (lots N1 et S) servi pour CETTE colonne.
   * Les deux colonnes portent la section quand les deux sessions ont un bloc ; le
   * placeholder D16 ne reste que pour une session réellement sans coordination.
   */
  hasCoordination: boolean
  /** Bloc `range_profiles` / `compare_range_profiles` (lot N2) servi pour cette colonne. */
  hasRange: boolean
  /**
   * Le bloc « usages d'equipement, armes speciales et objectifs » va-t-il DESSINER
   * quelque chose pour CETTE colonne (`sessionUsageShowsSomething`) ? Ce n'est plus
   * « le payload le porte-t-il » : un bloc servi mais muet laisserait le titre
   * « Frags et usages » au-dessus de rien. En comparaison, l'autre colonne rend un
   * placeholder.
   */
  hasUsage: boolean
  /** La carte des frags a-t-elle du contenu (`sessionFragCardHasContent`) ? */
  hasFrags: boolean
}

/** Cles effectivement presentes pour une colonne, dans l'ordre canonique. */
export function sessionSectionKeys({
  hasUsage,
  hasFrags,
  hasCoordination,
  hasRange,
}: SessionSectionInput): SessionSectionKey[] {
  const present: Record<string, boolean> = {
    usage: hasUsage,
    frags: hasFrags,
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
