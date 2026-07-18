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

// ── Sous-paliers en chiffres romains (source unique) ──────────────────────────
// Index 1→I … 6→VI (index 0 = pas de sous-palier). Aligné sur le Go
// rankSubRoman / FormatTierSubLabel. Garde-rail skillTiers.guard.test.ts :
// interdit toute re-déclaration du littéral sous src/features/**.
const SUB_TIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI'] as const

/**
 * Convertit un numéro de sous-palier (1..6) en chiffre romain pour composer un
 * libellé « Nom + sous-palier » côté web (colonnes CSR Career/Explorer,
 * notifications de palier). 1-indexé ; toute valeur hors 1..6 est rendue en
 * décimal (String(n)). Le cadrage (n>0, borne haute, Onyx sans sous-palier)
 * reste à la charge du consommateur.
 */
export function subTierRoman(n: number): string {
  return SUB_TIER_ROMAN[n] ?? String(n)
}

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

// ── Localisation des libellés de palier (chokepoint FR↔EN) ────────────────────
//
// Les libellés de palier de skill (CSR/LUSR) arrivent du backend soit comme nom
// de palier canonique EN (ex. `CareerCSRRank.tier` = "Gold"/"Platinum"), soit
// comme libellé COMPOSÉ baké à l'écriture — et ce bake n'est PAS locale-aware :
// Halo Infinite bake en FR ("Or IV", "Platine"), Halo 5 bake en EN ("Platinum",
// "Platinum 4"). D'où deux symptômes selon la surface : « Or IV » sous UI EN, et
// « Platinum » sous UI FR (H5).
//
// Ces helpers localisent À L'AFFICHAGE, à partir de la donnée déjà reçue (aucun
// champ backend supplémentaire) : le nom de palier appartient à un ensemble fini
// et stable (les grilles ci-dessus + Champion H5), et le sous-palier n'est qu'un
// chiffre (romain ou arabe) préservé tel quel. Toute entrée non reconnue
// (sentinelle « Placement », valeur brute, vide) est renvoyée inchangée.

interface TierNamePair {
  fr: string
  en: string
}

// Index des noms de palier connus (toute casse FR ou EN) → paire localisée.
// Dérivé de LUSR_TIER_GRID (porte les 6 paires fr/en) + Champion (apex Halo 5,
// au-dessus d'Onyx ; identique dans les deux langues).
const TIER_NAME_BY_KEY: Record<string, TierNamePair> = (() => {
  const out: Record<string, TierNamePair> = {}
  for (const t of LUSR_TIER_GRID.tiers) {
    const pair: TierNamePair = { fr: t.fr, en: t.en }
    out[t.fr.toLowerCase()] = pair
    out[t.en.toLowerCase()] = pair
  }
  out['champion'] = { fr: 'Champion', en: 'Champion' }
  return out
})()

/**
 * Localise un NOM de palier seul (« Or »/« Gold »/« Platinum »…) vers la locale
 * cible. Sert aux surfaces qui composent le libellé côté web depuis un tier
 * canonique EN + sous-palier séparé (ex. colonne CSR de la carrière). Une entrée
 * inconnue (vide, palier hors grille) est renvoyée telle quelle.
 */
export function localizeTierName(name: string, locale: 'fr' | 'en'): string {
  const pair = TIER_NAME_BY_KEY[name.trim().toLowerCase()]
  if (!pair) return name
  return locale === 'en' ? pair.en : pair.fr
}

/**
 * Localise un LIBELLÉ de palier complet baké (« Or IV », « Platinum 4 », « Onyx »,
 * « Onyx 1500 ») : mappe le NOM du palier vers la locale, préserve le suffixe de
 * sous-palier (romain I–VI ou arabe) tel quel. Une entrée dont le nom n'est pas un
 * palier connu (sentinelle « Placement », « Placement (2 restants) », null/vide)
 * est renvoyée INCHANGÉE — sûr pour les consommateurs qui gèrent ces cas à part.
 */
export function localizeTierLabel(
  label: string | null | undefined,
  locale: 'fr' | 'en',
): string | null | undefined {
  if (label == null || label.trim() === '') return label
  const trimmed = label.trim()
  // Sépare un éventuel suffixe : romain (I–VI) ou numérique (sous-palier arabe ou
  // valeur CSR d'Onyx). Le reste = nom du palier candidat.
  const m = trimmed.match(/^(.+?)\s+([IVX]+|\d+)$/)
  const namePart = (m ? m[1] : trimmed).trim()
  const pair = TIER_NAME_BY_KEY[namePart.toLowerCase()]
  if (!pair) return label // nom inconnu → inchangé (placement, brut…)
  const localized = locale === 'en' ? pair.en : pair.fr
  return m ? `${localized} ${m[2]}` : localized
}

// ── Valeur numérique de tri d'un palier (colonne « Rang » de l'Explorer) ──────
//
// ExplorerMatchRow ne porte AUCUN champ numérique de rang (seulement le libellé
// baké skill_tier_label + placement_done/total). Pour trier la colonne « Rang »
// sur une valeur cohérente (Bronze < … < Onyx < Champion) plutôt que par ordre
// alphabétique du libellé (Bronze, Diamant, Onyx, Or…, faux), on dérive un
// ordinal du libellé : ordre du palier majeur × 10000 + sous-palier.

// Nom de palier (FR ou EN, toute casse) → index majeur (Bronze=0 … Onyx=5,
// Champion=6). Dérivé de LUSR_TIER_GRID (ordre canonique) + Champion (apex H5).
const TIER_ORDINAL_BY_KEY: Record<string, number> = (() => {
  const out: Record<string, number> = {}
  LUSR_TIER_GRID.tiers.forEach((t, i) => {
    out[t.fr.toLowerCase()] = i
    out[t.en.toLowerCase()] = i
  })
  out['champion'] = LUSR_TIER_GRID.tiers.length
  return out
})()

const ROMAN_DIGIT_VALUES: Record<string, number> = { i: 1, v: 5, x: 10 }

/** Convertit un chiffre romain (I–VI usuels des sous-paliers) en entier. */
function romanToInt(roman: string): number {
  const s = roman.toLowerCase()
  let total = 0
  for (let i = 0; i < s.length; i++) {
    const cur = ROMAN_DIGIT_VALUES[s[i]] ?? 0
    const next = ROMAN_DIGIT_VALUES[s[i + 1]] ?? 0
    total += cur < next ? -cur : cur
  }
  return total
}

/**
 * Valeur numérique de tri d'un palier à partir de son libellé baké (« Or IV »,
 * « Diamond 3 », « Onyx 1500 », « Onyx »). Combine l'ordinal du palier majeur et
 * le sous-palier (romain ou arabe) pour un tri croissant cohérent. Renvoie
 * `undefined` pour une entrée non reconnue (sentinelle « Placement », vide, null)
 * → à ranger en bas via `sortUndefined: 'last'`. Le facteur 10000 laisse la place
 * aux valeurs CSR brutes d'Onyx (ex. « Onyx 1500 ») sans chevaucher le palier
 * supérieur.
 */
export function skillTierSortValue(label: string | null | undefined): number | undefined {
  if (label == null) return undefined
  const trimmed = label.trim()
  if (trimmed === '') return undefined
  const m = trimmed.match(/^(.+?)\s+([IVX]+|\d+)$/)
  const namePart = (m ? m[1] : trimmed).trim().toLowerCase()
  const tierIdx = TIER_ORDINAL_BY_KEY[namePart]
  if (tierIdx === undefined) return undefined // « Placement », brut, inconnu → bas
  const sub = m ? (/^\d+$/.test(m[2]) ? Number(m[2]) : romanToInt(m[2])) : 0
  return tierIdx * 10000 + sub
}

/**
 * composeTierLabel — libellé « Nom + sous-palier » d'un palier de skill CLASSÉ
 * (ex. « Diamant III », « Onyx »), à partir d'un tier canonique EN + un numéro de
 * sous-palier. Nom localisé via `localizeTierName`, sous-palier romain via
 * `subTierRoman` ; Onyx (palier ouvert) et tout sous-palier hors 1..6 → nom seul.
 *
 * SOURCE UNIQUE du pattern « compose tier » (CLAUDE.md n°6, règle ≤ 2 copies —
 * garde-rail skillTiers.guard.test.ts). Ne gère PAS les états non classés
 * (placement, tier vide) : c'est à l'appelant de les traiter en amont.
 */
export function composeTierLabel(tier: string, subTier: number, locale: 'fr' | 'en'): string {
  const name = localizeTierName(tier, locale)
  if (tier.trim().toLowerCase() === 'onyx') return name
  return subTier >= 1 && subTier <= 6 ? `${name} ${subTierRoman(subTier)}` : name
}
