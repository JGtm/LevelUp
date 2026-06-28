/**
 * View-model NORMALISÉ des citations/commendations — frontière title-agnostic entre
 * les données brutes (CitationItem Infinite, NativeCommendationTotal H5, snippets de
 * match) et l'UI partagée (CitationsView / CitationCard). Tout titre se projette sur
 * CitationDisplayItem via un normaliseur (cf. normalize.ts) ; l'UI ne connaît QUE ce
 * type. Ajouter Halo 7 = un normaliseur de plus, zéro changement d'UI.
 */
export interface CitationDisplayItem {
  /** Clé stable de liste (name_norm | id | snippet.key). */
  key: string
  /** Libellé résolu (name_display | name || `#${id8}`). */
  name: string
  /** Description optionnelle (Infinite / snippets). */
  description?: string
  /** Image (image_url Infinite | icon_url H5). */
  imageUrl?: string
  /** Progression 0..100 (mastery_pct | progress_pct). */
  pct: number
  /** Paliers atteints (earned_tiers | tier_index | 0). */
  tierIndex: number
  /** Nombre total de paliers (0 → pas d'anneau, fallback icône). */
  tierCount: number
  /** Total à vie / cumulé. */
  total: number
  /** Seuil du prochain palier (0 si maîtrisé). */
  nextTierTarget: number
  /** Maîtrisée — PRÉ-CALCULÉ via citationMastery() dans le normaliseur. */
  isMastered: boolean
  /** Nouvellement maîtrisée CE match (surfaces de match uniquement). */
  isNewlyMastered: boolean
  /** Source — pilote le pied de tuile (progression palier vs total à vie). */
  source: CitationSource
}

export type CitationSource = 'infinite' | 'native'

export interface CitationCategoryView {
  /** Libellé brut de catégorie (Title-case au rendu). */
  category: string
  items: CitationDisplayItem[]
  /** Citations complétées : Infinite = group.completed ; natif = Σ isMastered. */
  completed: number
}

export interface CitationsViewModel {
  categories: CitationCategoryView[]
  /** Σ completed (numérateur de l'en-tête de maîtrise). */
  masteredTotal: number
  /** Dénominateur : Infinite = nb total de citations ; natif = total_count. */
  itemsTotal: number
  source: CitationSource
  /** Barre de filtres disponible (Infinite seulement). */
  hasFilters: boolean
}
