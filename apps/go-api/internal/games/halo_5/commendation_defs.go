package halo_5

// commendation_defs.go — résolution des définitions natives (nom + icône) des
// commendations Halo 5, issues du référentiel metadata h5 (commendation_definitions,
// seedé par cmd/h5-metadata-fetch depuis l'API Metadata officielle /commendations).
// AXE B prod-gate, suite « définitions natives ». Consommée par
// enrichCommendationTotals (commendation_totals.go, totaux à vie).
//
// L'interface est définie côté package consommateur (h5) et retourne un type
// canonique : l'impl DuckDB (platform/duckdb) la satisfait STRUCTURELLEMENT, sans
// import croisé (parité MatchHistorySource).

import (
	"context"

	"levelup/go-api/internal/games/canonical"
)

// CommendationDefSource résout les définitions natives (nom + icône) des commendations
// Halo 5 par UUID, depuis le référentiel metadata.duckdb (commendation_definitions).
// Read-only, best-effort : une erreur ou une clé absente laisse la commendation brute.
type CommendationDefSource interface {
	// LookupCommendations retourne, pour les UUID demandés, leur définition native
	// (clé = commendation_id). Les IDs inconnus sont simplement absents de la map.
	LookupCommendations(ctx context.Context, ids []string) (map[string]canonical.CommendationDefinition, error)
}
