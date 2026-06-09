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

// ── Grilles de paliers de skill (LUSR / CSR) pour les bandes de classement ──
// Ce fichier est whitelisté dans lint-no-hardcoded-fields : les noms de tier
// sont des identifiants de rang Halo Infinite, pas des libellés génériques.

/**
 * Une bande de palier sur l'axe rating_value. `min`/`max` sont sur l'échelle de
 * la métrique. `subTiers` = nombre de sous-paliers de largeur égale (1 = pas de
 * sous-palier, ex. Onyx ouvert).
 */
export interface SkillTier {
  min: number
  max: number
  fr: string
  en: string
  subTiers: number
}

export interface SkillTierGrid {
  tiers: SkillTier[]
  /** Numérotation des sous-paliers : 'roman' (LUSR, façon Go FormatTierSubLabel)
   *  ou 'arabic' (CSR, façon Halo « Diamond 3 »). */
  subTierStyle: 'roman' | 'arabic'
}

/**
 * Grille LUSR — échelle legacy 1000-2000. Bornes ET nombres de sous-paliers
 * synchronisés avec Go skill_v2/tier.go (DefaultTierBoundaries) + legacy_mapping.go
 * (LegacyTierRange = 200 pts/tier ; largeur sous-palier = 200 / subTiers).
 * ⚠️ Couplage manuel : si la grille Go change, mettre à jour ici.
 */
export const LUSR_TIER_GRID: SkillTierGrid = {
  subTierStyle: 'roman',
  tiers: [
    { min: 1000, max: 1200, fr: 'Bronze',  en: 'Bronze',   subTiers: 6 },
    { min: 1200, max: 1400, fr: 'Argent',  en: 'Silver',   subTiers: 3 },
    { min: 1400, max: 1600, fr: 'Or',      en: 'Gold',     subTiers: 6 },
    { min: 1600, max: 1800, fr: 'Platine', en: 'Platinum', subTiers: 2 },
    { min: 1800, max: 2000, fr: 'Diamant', en: 'Diamond',  subTiers: 3 },
    { min: 2000, max: 9999, fr: 'Onyx',    en: 'Onyx',     subTiers: 1 },
  ],
}

/**
 * Grille CSR — échelle Halo Infinite brute (competitive). Chaque tier
 * Bronze..Diamant = 300 pts / 6 sous-rangs de 50 ; Onyx ouvert (1500+),
 * numéroté sans sous-rangs.
 */
export const CSR_TIER_GRID: SkillTierGrid = {
  subTierStyle: 'arabic',
  tiers: [
    { min: 0,    max: 300,  fr: 'Bronze',  en: 'Bronze',   subTiers: 6 },
    { min: 300,  max: 600,  fr: 'Argent',  en: 'Silver',   subTiers: 6 },
    { min: 600,  max: 900,  fr: 'Or',      en: 'Gold',     subTiers: 6 },
    { min: 900,  max: 1200, fr: 'Platine', en: 'Platinum', subTiers: 6 },
    { min: 1200, max: 1500, fr: 'Diamant', en: 'Diamond',  subTiers: 6 },
    { min: 1500, max: 9999, fr: 'Onyx',    en: 'Onyx',     subTiers: 1 },
  ],
}

/** Grille à utiliser selon les types de rating présents : CSR si tous les
 *  types sont 'CSR', sinon LUSR (défaut, et cas mixte legacy-scale). */
export function gridForRatingTypes(ratingTypes: Array<string | null | undefined>): SkillTierGrid {
  const upper = ratingTypes.map(t => (t ?? '').toUpperCase()).filter(Boolean)
  return upper.length > 0 && upper.every(t => t === 'CSR') ? CSR_TIER_GRID : LUSR_TIER_GRID
}

/** Position d'un rating à l'intérieur de son sous-palier courant. */
export interface SubTierPosition {
  /** Avancement [0,1) vers le sous-palier suivant. */
  pct: number
  /** Borne basse (sur l'échelle du rating) du sous-palier courant. */
  subTierMin: number
  /** Largeur du sous-palier sur l'échelle du rating. */
  subTierWidth: number
}

/**
 * Position d'un `ratingValue` dans son sous-palier courant, selon la grille.
 *
 * Indispensable parce que la largeur d'un sous-palier N'EST PAS constante :
 * CSR = 50 pts partout, mais LUSR (échelle legacy 1000-2000) varie selon le
 * tier (Bronze 200/6≈33, Argent 200/3≈67, Platine 200/2=100…). Calculer la
 * progression avec une largeur fixe (l'ancien `rating mod 50`) faussait la
 * barre LUSR et faisait disparaître la portion "avant match".
 *
 * Retourne `null` pour le palier ouvert (Onyx, `subTiers ≤ 1`) ou hors grille :
 * pas de sous-palier suivant → pas de barre de progression significative.
 */
export function subTierPosition(grid: SkillTierGrid, ratingValue: number): SubTierPosition | null {
  const tier = grid.tiers.find(t => ratingValue >= t.min && ratingValue < t.max)
  if (!tier || tier.subTiers <= 1) return null
  const subTierWidth = (tier.max - tier.min) / tier.subTiers
  const subIndex = Math.floor((ratingValue - tier.min) / subTierWidth)
  const subTierMin = tier.min + subIndex * subTierWidth
  return { pct: (ratingValue - subTierMin) / subTierWidth, subTierMin, subTierWidth }
}
