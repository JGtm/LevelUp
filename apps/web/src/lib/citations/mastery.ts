/**
 * citationMastery — point de décision UNIQUE « cette citation/commendation est-elle
 * maîtrisée ? ». Chokepoint title-agnostic partagé par TOUTES les surfaces (tuiles de
 * match, détail de match, scoreboard, page Citations Infinite ET H5). Avant ce helper,
 * la règle était dupliquée et calibrée sur Infinite (anneau doré sur `is_newly_mastered`
 * seul), ce qui affichait à tort « en cours » une commendation H5 maîtrisée de longue
 * date. Fonction pure, sans React ni I/O.
 *
 * L'entrée utilise les noms de champs BRUTS de l'API : chaque surface passe son objet
 * tel quel, zéro mise en forme. Les titres exposent des synonymes (Infinite :
 * earned_tiers/mastery_pct ; H5 : tier_index/progress_pct/is_mastered) → on les
 * réconcilie ici, une fois pour toutes.
 */
export interface MasteryInput {
  /** Flag autoritatif (NativeCommendationTotal H5 — totaux à vie). */
  is_mastered?: boolean | null
  /** Nombre total de paliers (toutes les formes tierées). 0 → pas d'anneau. */
  tier_count?: number | null
  /** Paliers atteints (snippets + total natif). */
  tier_index?: number | null
  /** Synonyme Infinite de tier_index (CitationItem). */
  earned_tiers?: number | null
  /** Progression 0..100 (snippets + total natif). */
  progress_pct?: number | null
  /** Synonyme Infinite de progress_pct (CitationItem). */
  mastery_pct?: number | null
}

/**
 * Maîtrisée = palier final franchi (cette partie OU avant), ou anneau plein 100 %
 * pour les citations sans paliers. Distinct de `is_newly_mastered` (notion par-match,
 * conservée séparément pour le libellé « nouvellement maîtrisée »).
 */
export function citationMastery(c: MasteryInput): boolean {
  // 1) Flag explicite du backend (page totaux native) → fait foi.
  if (c.is_mastered === true) return true
  const tierCount = c.tier_count ?? 0
  const tierIndex = c.tier_index ?? c.earned_tiers ?? 0
  // 2) Tierée : palier final atteint.
  if (tierCount > 0) return tierIndex >= tierCount
  // 3) Non tierée : remplissage à 100 %.
  return (c.progress_pct ?? c.mastery_pct ?? 0) >= 100
}
