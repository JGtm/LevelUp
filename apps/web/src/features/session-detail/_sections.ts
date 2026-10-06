/**
 * _sections — modele des SECTIONS d'une colonne de session (D16 ; une cle par CARTE depuis D11 du
 * plan PLAN_SESSIONS_EMPRISE_2026-10-06).
 *
 * Source UNIQUE de l'ordre et de la composition des blocs empiles sous le L3 :
 * consommee par `SessionColumnBody` (qui associe une cle a un rendu) ET par
 * `SessionDetailPage` (qui fusionne les cles des deux colonnes pour composer des
 * RANGEES partagees en mode comparaison). Deux listes divergentes rendraient
 * l'alignement gauche/droite faux sans que rien ne casse — d'ou la centralisation.
 *
 * Deux niveaux de titres : le GROUPE (« Bilan », « Match par match », « Frags et usages ») et,
 * sous « Frags et usages », le SOUS-GROUPE (« Ressources de la soirée », « Prendre, et s'en
 * servir », …). Un titre se pose sur la premiere cle PRESENTE qu'il coiffe, jamais au-dessus de
 * rien. En pleine page, trois PAIRES de cartes partagent une rangee (A|B, C|D, G|H).
 */

import type { SessionPageResponse } from '@/lib/api/types'
import type { SessionManifestKey } from '@/lib/i18n/generated/session'

import {
  sessionColumnBlocks,
  sessionColumnPresence,
  type SessionCardsPresence,
} from './sessionEmprise.logic'

/** Les cartes A a L (et B'), une cle chacune — `sessionCardsPresence` dit lesquelles existent. */
type SessionCardKey = keyof SessionCardsPresence

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
  | 'career_xp'
  | SessionCardKey
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
  'frag_bar',
  'tools',
  'weapon_accuracy',
  'control',
  'fil',
  'grid',
  'mine',
  'production',
  'yield',
  'lives',
  'objective_balance',
  'objective_sheet',
  'equipment',
  'matches',
]

/**
 * LES TROIS SECTIONS TITREES de la page session (chantier « sections transverses »,
 * 2026-09-22). Un groupe coiffe PLUSIEURS cles : le titre se pose une fois, au-dessus de
 * la premiere cle PRESENTE du groupe, et nulle part si le groupe est entierement absent —
 * un titre au-dessus de rien annoncerait une mesure qui n'existe pas.
 *
 * `summary` (la bande KPI) n'a PAS de groupe : c'est l'en-tete de la colonne, pas une
 * section de plus — meme choix que l'accueil. `matches` pose son propre titre avec son
 * tableau (`space-y-3` : un tableau se colle plus a son titre que des graphes).
 */
type SessionSectionGroup = 'overview' | 'match_by_match' | 'kills_usage'

/** Cle de manifeste du titre de chaque groupe. */
export const SESSION_GROUP_TITLE_KEY: Record<SessionSectionGroup, SessionManifestKey> = {
  overview: 'session.detail.section_overview',
  match_by_match: 'session.detail.section_match_by_match',
  kills_usage: 'session.detail.section_kills_usage',
}

const KILLS_USAGE_KEYS: readonly SessionCardKey[] = [
  'frag_bar',
  'tools',
  'weapon_accuracy',
  'control',
  'fil',
  'grid',
  'mine',
  'production',
  'yield',
  'lives',
  'objective_balance',
  'objective_sheet',
  'equipment',
]

/** Groupe de chaque cle — les cles absentes de cette table ne sont coiffees par rien. */
const SESSION_SECTION_GROUPS: Partial<Record<SessionSectionKey, SessionSectionGroup>> = {
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
  ...Object.fromEntries(KILLS_USAGE_KEYS.map((k) => [k, 'kills_usage'])),
}

/** Les sous-groupes de « Frags et usages » (maquette « Après » : intertitres `h3`). */
export type SessionSubgroup = 'resources' | 'prendre' | 'lives' | 'objectif' | 'equipment'

/** Cle de manifeste de l'intertitre de chaque sous-groupe. */
export const SESSION_SUBGROUP_TITLE_KEY: Record<SessionSubgroup, SessionManifestKey> = {
  resources: 'session.detail.subsection_resources',
  prendre: 'session.detail.subsection_prendre',
  lives: 'session.detail.subsection_lives',
  objectif: 'session.detail.subsection_objectif',
  equipment: 'session.detail.subsection_equipment',
}

/** Sous-groupe de chaque carte ; A, B, B' n'en ont pas (directement sous le groupe). */
const SESSION_SECTION_SUBGROUPS: Partial<Record<SessionSectionKey, SessionSubgroup>> = {
  control: 'resources',
  fil: 'resources',
  grid: 'resources',
  mine: 'resources',
  production: 'prendre',
  yield: 'prendre',
  lives: 'lives',
  objective_balance: 'objectif',
  objective_sheet: 'objectif',
  equipment: 'equipment',
}

/** Les paires de la pleine page : deux cartes d'une meme paire, consecutives, partagent une rangee. */
const SESSION_SECTION_PAIRS: Partial<Record<SessionSectionKey, string>> = {
  frag_bar: 'frags',
  tools: 'frags',
  control: 'resources',
  fil: 'resources',
  production: 'prendre',
  yield: 'prendre',
}

/** Une suite de cles consecutives coiffees par le meme sous-groupe (ou par aucun). */
interface SessionSubRun {
  subgroup: SessionSubgroup | null
  keys: SessionSectionKey[]
}

/**
 * Les cles d'une liste RENDUE, groupees dans l'ordre. Une seule lecture pour les deux
 * rendus de `SessionColumnBody` (pile pleine page, rangees de comparaison) : la place du
 * titre ne peut pas diverger de l'un a l'autre.
 */
interface SessionSectionRun {
  /** Groupe coiffant ces cles, ou `null` pour les cles sans titre (`summary`, `matches`). */
  group: SessionSectionGroup | null
  keys: SessionSectionKey[]
  /** Les memes cles, par sous-groupe. */
  subruns: SessionSubRun[]
}

/**
 * Les suites d'elements consecutifs de meme cle. Sans cle (`null`), un element reste seul, sauf
 * `joinNull` : les cartes sans sous-groupe (A, B, B') forment UNE suite sous leur groupe.
 */
function runsBy<K, T>(keys: readonly T[], keyOf: (k: T) => K | null, joinNull = false): { key: K | null; items: T[] }[] {
  const runs: { key: K | null; items: T[] }[] = []
  for (const item of keys) {
    const k = keyOf(item)
    const last = runs[runs.length - 1]
    if (last && last.key === k && (k != null || joinNull)) last.items.push(item)
    else runs.push({ key: k, items: [item] })
  }
  return runs
}

export function groupSessionSections(keys: readonly SessionSectionKey[]): SessionSectionRun[] {
  return runsBy(keys, (k) => SESSION_SECTION_GROUPS[k] ?? null).map((run) => ({
    group: run.key,
    keys: run.items,
    subruns: runsBy(run.items, (k) => SESSION_SECTION_SUBGROUPS[k] ?? null, true).map((s) => ({
      subgroup: s.key,
      keys: s.items,
    })),
  }))
}

/** Ce qu'une rangee de comparaison ouvre : un titre de groupe, un intertitre, ou les deux. */
interface SessionRowOpening {
  group?: SessionSectionGroup
  subgroup?: SessionSubgroup
}

/**
 * sessionRowOpenings — en comparaison, les titres a poser dans la rangee de chaque cle : celui du
 * groupe et celui du sous-groupe vont dans la rangee de leur PREMIERE cle presente. `rowKeys` etant
 * la meme liste des deux cotes, les deux colonnes posent leurs titres a la meme rangee.
 */
export function sessionRowOpenings(rowKeys: readonly SessionSectionKey[]): Map<SessionSectionKey, SessionRowOpening> {
  const open = new Map<SessionSectionKey, SessionRowOpening>()
  for (const run of groupSessionSections(rowKeys)) {
    if (run.group != null) open.set(run.keys[0], { group: run.group })
    for (const sub of run.subruns) {
      if (sub.subgroup == null) continue
      open.set(sub.keys[0], { ...open.get(sub.keys[0]), subgroup: sub.subgroup })
    }
  }
  return open
}

/** pairSessionKeys — les rangees de la pleine page : deux cles d'une meme paire, consecutives, ensemble. */
export function pairSessionKeys(keys: readonly SessionSectionKey[]): SessionSectionKey[][] {
  return runsBy(keys, (k) => SESSION_SECTION_PAIRS[k] ?? null).map((r) => r.items)
}

/** Les cles optionnelles d'une colonne : ses cartes, la Coordination et la Portee. */
export type SessionSectionPresence = SessionCardsPresence & { coordination: boolean; range: boolean }

/** Cles effectivement presentes pour une colonne, dans l'ordre canonique. */
export function sessionSectionKeys(presence: SessionSectionPresence): SessionSectionKey[] {
  const optional: Partial<Record<SessionSectionKey, boolean>> = presence
  return SESSION_SECTION_ORDER.filter((key) => optional[key] ?? true)
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

/** sessionRowKeys — les rangees partagees de la comparaison : l'union des cles des deux colonnes. */
export function sessionRowKeys(data: SessionPageResponse): SessionSectionKey[] {
  const keysOf = (side: 'current' | 'compare') => {
    const col = sessionColumnBlocks(data, side)
    return sessionSectionKeys({
      ...sessionColumnPresence(col),
      coordination: col.coordination != null,
      range: col.rangeProfiles != null,
    })
  }
  return mergeSessionSectionKeys(keysOf('current'), keysOf('compare'))
}
