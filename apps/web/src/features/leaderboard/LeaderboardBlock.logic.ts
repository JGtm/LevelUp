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
 *     tirets ; on ne les montre qu'au-delà d'un seuil.
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

/** Ligne du classement, réduite à ce dont la décision d'affichage a besoin. */
export interface EnrichableEntry {
  match_count?: number | null
}

/**
 * showEnrichedColumns — les colonnes détaillées s'affichent-elles ? Oui dès que la part des lignes
 * enrichies atteint `ENRICHED_COLUMNS_MIN_RATIO` ; une ligne sans stats détaillées garde alors des
 * tirets neutres. Aucun bandeau ne dit la part des lignes enrichies (aucun inconnu à l'écran).
 */
export function showEnrichedColumns(entries: readonly EnrichableEntry[]): boolean {
  if (entries.length === 0) return false
  const enriched = entries.reduce((n, e) => (e.match_count != null ? n + 1 : n), 0)
  return enriched / entries.length >= ENRICHED_COLUMNS_MIN_RATIO
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

/**
 * Clés de tri portées par les colonnes ENRICHIES (celles qui disparaissent sous le
 * seuil de couverture). `rank` et `csr` en sont exclus : ces colonnes sont
 * toujours là. Idem `matches`/`value`, qui appartiennent aux catégories de stats.
 */
export const ENRICHED_SORT_KEYS: readonly string[] = [
  'kda',
  'kills',
  'deaths',
  'assists',
  'win_rate',
  'world_matches',
  'accuracy',
  'dmg_per_kill',
  'dmg_per_death',
]

/** La clé de tri appartient-elle à une colonne enrichie ? */
export function isEnrichedSortKey(key: string): boolean {
  return ENRICHED_SORT_KEYS.includes(key)
}

/** Tri par défaut : le rang, croissant — le seul toujours affichable. */
export const DEFAULT_SORT: { key: string; dir: 'asc' | 'desc' } = { key: 'rank', dir: 'asc' }

/**
 * resolveSort donne le tri EFFECTIVEMENT appliqué au rendu.
 *
 * Quand les colonnes enrichies ne sont pas affichées, un tri posé sur l'une
 * d'elles ordonnerait la table par une colonne invisible — et resterait
 * inannulable, puisque l'en-tête qui le porte a disparu. On retombe alors sur le
 * rang croissant. L'état du composant n'est PAS modifié (aucun setState dans un
 * effet) : le tri choisi revient tel quel dès que les colonnes réapparaissent.
 */
export function resolveSort(
  sortKey: string,
  sortDir: 'asc' | 'desc',
  enrichedColumnsShown: boolean,
): { key: string; dir: 'asc' | 'desc' } {
  if (!enrichedColumnsShown && isEnrichedSortKey(sortKey)) {
    return { ...DEFAULT_SORT }
  }
  return { key: sortKey, dir: sortDir }
}
