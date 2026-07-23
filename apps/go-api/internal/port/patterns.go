package port

import (
	"context"

	"levelup/go-api/internal/analysis/patterns"
)

// PatternsRepository fournit les MatchRow analysées pour le Pattern Engine v3.
//
// Découple le handler GET /patterns de l'accès données : le handler dépend de
// cette interface, l'implémentation DuckDB (duckdb.PatternsRepo) porte le SQL
// cross-DB et le merge. Permet de mocker le chargement en test et d'isoler la
// couche HTTP du moteur de stockage.
type PatternsRepository interface {
	// LoadRows charge et assemble les `limit` matchs les plus récents du joueur
	// (stats + enrichissements + deltas de rating) prêts pour patterns.Analyze.
	LoadRows(ctx context.Context, limit int) ([]patterns.MatchRow, error)
	// ResolveMapLabels résout les noms lisibles des cartes (map_id -> nom) depuis
	// le référentiel metadata du titre courant, dans la langue du contexte
	// (ctxkeys.Locale). Les map_id sans traduction sont absents du résultat (le
	// caller applique un repli). Title-agnostic : même source que les autres
	// pages (asset_translations).
	ResolveMapLabels(ctx context.Context, mapIDs []string) (map[string]string, error)
	// ResolveMapFilterKeys résout les noms de cartes (map_id -> nom) avec une
	// préférence de langue FIXE fr→en, INDÉPENDANTE de la locale de la requête —
	// la clé de filtrage stable des liens pattern→Solo (F7). C'est exactement la
	// valeur que le pipeline de filtres matche (mapUI = MapNameFR ?? MapName,
	// FR-first quelle que soit la locale). Distinct de ResolveMapLabels (localisé,
	// pour l'affichage). Même source (asset_translations), title-agnostic.
	ResolveMapFilterKeys(ctx context.Context, mapIDs []string) (map[string]string, error)
}
