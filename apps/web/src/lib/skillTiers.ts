/**
 * skillTiers.ts — Valeurs canoniques des paliers de skill (CSR / LUSR).
 *
 * Ces strings correspondent aux valeurs stockées en DB dans match_skill_rank.skill_tier.
 * Elles sont utilisées comme valeurs de filtre API côté Explorer (non comme libellés d'affichage).
 *
 * Exception lint-no-hardcoded-fields : ce sont des identifiants API Halo (valeurs de donnée),
 * pas des libellés d'affichage — pattern analogue à lib/medalDifficulty.ts.
 */

export const SKILL_TIER_VALUES = [
  'Bronze',
  'Silver',
  'Gold',
  'Platinum',
  'Diamond',
  'Onyx',
] as const

export type SkillTierValue = typeof SKILL_TIER_VALUES[number]
