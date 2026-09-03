/**
 * LeaderboardBlock.logic — décisions d'affichage du classement CSR mondial,
 * extraites du composant pour être testables sans rendu (et pour ne pas grossir
 * LeaderboardBlock.tsx, déjà au-dessus du seuil de 500 lignes).
 *
 * Deux décisions, une même cause : le relevé mondial et son enrichissement ne
 * couvrent pas tout.
 *
 *  1. Couverture d'enrichissement — les colonnes détaillées (FDA, frags, morts,
 *     précision…) viennent d'un backfill par joueur. Quand une poignée seulement
 *     des joueurs affichés est enrichie, montrer les colonnes produit un mur de
 *     tirets ; on ne les montre qu'au-delà d'un seuil, et on dit ce qui manque.
 *  2. Couplage saison ↔ playlist — toutes les saisons n'ont pas été relevées sur
 *     toutes les playlists. Le catalogue donne les couples réels (playlist_ids
 *     par saison) : le sélecteur de playlist ne propose que ceux-là, sinon
 *     changer de saison désigne un couple jamais capturé (tableau vide).
 */

/**
 * Part MINIMALE de lignes enrichies pour afficher les colonnes détaillées
 * (décision D2 du plan). En dessous, la table reste en mode CSR seul : mieux vaut
 * un tableau court et vrai que 11 colonnes de tirets.
 */
export const ENRICHED_COLUMNS_MIN_RATIO = 0.25

/**
 * Part au-dessus de laquelle la couverture est considérée COMPLÈTE (pas de
 * bandeau). Entre les deux seuils : colonnes affichées ET bandeau « partielles »,
 * pour que les cellules vides soient expliquées plutôt que subies.
 */
export const ENRICHED_FULL_RATIO = 0.8

/** Ligne du classement, réduite à ce dont la couverture a besoin. */
export interface EnrichableEntry {
  match_count?: number | null
}

/** Verdict de couverture : ce que la table montre, et ce qu'elle doit annoncer. */
export interface EnrichmentCoverage {
  /** Nombre de lignes affichées portant des stats détaillées. */
  enriched: number
  /** Nombre de lignes affichées. */
  total: number
  /** Part enrichie (0..1) ; 0 si aucune ligne. */
  ratio: number
  /** Afficher les colonnes détaillées ? */
  showColumns: boolean
  /** Bandeau « indisponibles » (sous le seuil, avec au moins une ligne affichée). */
  showUnavailableNote: boolean
  /** Bandeau « partielles » (colonnes affichées mais couverture incomplète). */
  showPartialNote: boolean
}

/**
 * enrichmentCoverage mesure la part de lignes enrichies et en déduit l'affichage.
 * Table vide → aucune colonne, aucun bandeau (l'état vide parle déjà).
 */
export function enrichmentCoverage(entries: readonly EnrichableEntry[]): EnrichmentCoverage {
  const total = entries.length
  const enriched = entries.reduce((n, e) => (e.match_count != null ? n + 1 : n), 0)
  const ratio = total > 0 ? enriched / total : 0
  const showColumns = total > 0 && ratio >= ENRICHED_COLUMNS_MIN_RATIO
  return {
    enriched,
    total,
    ratio,
    showColumns,
    showUnavailableNote: total > 0 && !showColumns,
    showPartialNote: showColumns && ratio < ENRICHED_FULL_RATIO,
  }
}

/**
 * playlistsForSeason restreint les options de playlist à celles réellement
 * relevées pour la saison choisie.
 *
 * Deux dégradations volontaires vers la liste complète :
 *  - `seasonPlaylistIDs` absent → backend antérieur au champ `playlist_ids` ;
 *  - filtrage vide → le catalogue et la saison se contredisent ; un sélecteur
 *    sans aucune option serait pire que des options optimistes.
 */
export function playlistsForSeason<T extends { value: string }>(
  options: readonly T[],
  seasonPlaylistIDs: readonly string[] | null | undefined,
): T[] {
  const all = [...options]
  if (!seasonPlaylistIDs || seasonPlaylistIDs.length === 0) {
    return all
  }
  const allowed = new Set(seasonPlaylistIDs)
  const kept = all.filter((o) => allowed.has(o.value))
  return kept.length > 0 ? kept : all
}

/**
 * pickEffectiveOption garde le choix de l'utilisateur tant qu'il figure dans les
 * options, sinon retombe sur la première (dérivation au rendu — pas de setState
 * dans un effet). Sert aux deux sélecteurs : après un changement de saison, la
 * playlist courante peut ne plus exister pour cette saison.
 */
export function pickEffectiveOption(options: readonly { value: string }[], current: string): string {
  return options.some((o) => o.value === current) ? current : (options[0]?.value ?? current)
}
