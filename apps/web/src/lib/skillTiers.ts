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

// Paliers LUSR — valeurs synchronisées avec skill_config.go (SkillTiers).
// Labels FR = FormatTierLabel() côté Go ; labels EN = valeur brute API Halo.
// Ce fichier est whitelisté dans lint-no-hardcoded-fields : ces strings sont
// des identifiants de rang Halo Infinite, pas des libellés d'affichage génériques.
export const LUSR_TIERS: Array<{
  min: number
  max: number
  token: 'perf-tier-1' | 'perf-tier-2' | 'perf-tier-3' | 'perf-tier-4' | 'perf-tier-5'
  fr: string
  en: string
}> = [
  { min: 1200, max: 1400, token: 'perf-tier-1', fr: 'Argent',  en: 'Silver'   },
  { min: 1400, max: 1600, token: 'perf-tier-2', fr: 'Or',      en: 'Gold'     },
  { min: 1600, max: 1800, token: 'perf-tier-3', fr: 'Platine', en: 'Platinum' },
  { min: 1800, max: 2000, token: 'perf-tier-4', fr: 'Diamant', en: 'Diamond'  },
  { min: 2000, max: 9999, token: 'perf-tier-5', fr: 'Onyx',    en: 'Onyx'     },
]
