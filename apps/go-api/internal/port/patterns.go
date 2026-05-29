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
}
