import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { SemanticToken } from '@/lib/accessibility'

/**
 * Groupes LUSR « connus » PAR TITRE — miroir front des chaînes calculées côté
 * backend (Halo Infinite : pair_name → 4 chaînes ; Halo 5 : chaîne unique
 * `h5_arena`, cf. internal/games/halo_5/lusr_chain.go).
 *
 * Rôle (multi-titre) : pour un titre listé ici, on AFFICHE toujours ses groupes
 * connus — même non classés (« Non classé »). Un titre absent de la map → connus
 * vides → on n'affiche QUE les groupes réellement présents dans la donnée
 * (dérivés des checkpoints). Le seul littéral toléré est la clé `halo_infinite`
 * (config par-titre, pas data-path).
 */
export const LUSR_KNOWN_GROUPS_BY_TITLE: Record<string, readonly string[]> = {
  halo_infinite: ['arena_slayer', 'arena_objectif', 'btb', 'chaos'],
}

/**
 * Groupes connus du titre Halo Infinite — conservé pour compat (consommateurs
 * existants : graphes/séries LUSR). Pour le rendu title-aware, préférer
 * {@link knownLusrGroupsForTitle} + {@link resolveLusrGroupsForDisplay}.
 */
export const LUSR_KNOWN_GROUPS = LUSR_KNOWN_GROUPS_BY_TITLE.halo_infinite
export type LusrGroup = (typeof LUSR_KNOWN_GROUPS)[number]

export const LUSR_GROUP_TOKENS: Record<string, SemanticToken> = {
  arena_slayer:   'compare-a',
  arena_objectif: 'compare-b',
  btb:            'divergent-pos',
  chaos:          'narrative-humiliation',
  h5_arena:       'compare-a',
}

/**
 * knownLusrGroupsForTitle — groupes connus déclarés pour `titleSlug`, ou `[]`
 * pour un titre absent de la map (ex. halo_5 → on s'appuie sur la donnée seule).
 */
export function knownLusrGroupsForTitle(titleSlug: string): readonly string[] {
  return LUSR_KNOWN_GROUPS_BY_TITLE[titleSlug] ?? []
}

/**
 * resolveLusrGroupsForDisplay — ordre déterministe des groupes LUSR à afficher :
 * UNION (groupes connus du titre, dans l'ordre déclaré) puis (groupes présents
 * dans la donnée mais non connus, triés alpha pour stabilité).
 *
 * Garantit le no-op Halo Infinite : si `dataGroups` ⊆ connus, on rend exactement
 * les 4 groupes connus dans le même ordre qu'avant (« Non classé » inclus).
 */
export function resolveLusrGroupsForDisplay(
  titleSlug: string,
  dataGroups: Iterable<string>,
): string[] {
  const known = knownLusrGroupsForTitle(titleSlug)
  const knownSet = new Set(known)
  const extra = [...new Set(dataGroups)]
    .filter((g) => !knownSet.has(g))
    .sort()
  return [...known, ...extra]
}

export function lusrChainLabel(group: string, locale: ManifestLocale): string {
  const key = `career.lusr.chain.${group}` as keyof typeof careerManifest
  return careerManifest[key]?.[locale] ?? group
}
