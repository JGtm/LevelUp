package halo_5

// commendation_defs.go — enrichissement des commendations NATIVES du MatchDetail avec
// leurs définitions (nom + icône) issues du référentiel metadata h5
// (commendation_definitions, seedé par cmd/h5-metadata-fetch depuis l'API Metadata
// officielle /commendations). AXE B prod-gate, suite « définitions natives ».
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

// enrichCommendations peuple Name/IconURL des commendations natives du détail depuis
// le référentiel commendation_definitions. BEST-EFFORT : source nil, erreur de
// lookup ou ID inconnu → la commendation reste brute (le front dégrade sur l'ID
// court). N'écrase jamais un Name déjà présent par une chaîne vide.
func (a *DataAdapter) enrichCommendations(ctx context.Context, detail *canonical.MatchDetail) {
	if a.commendationDefs == nil || detail == nil || len(detail.Commendations) == 0 {
		return
	}
	ids := make([]string, 0, len(detail.Commendations))
	for i := range detail.Commendations {
		if detail.Commendations[i].ID != "" {
			ids = append(ids, detail.Commendations[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	defs, err := a.commendationDefs.LookupCommendations(ctx, ids)
	if err != nil {
		a.logger.DebugContext(ctx, "h5 enrichCommendations: lookup échoué (commendations laissées brutes)",
			"err", err, "n", len(ids))
		return
	}
	for i := range detail.Commendations {
		def, ok := defs[detail.Commendations[i].ID]
		if !ok {
			continue
		}
		if def.Name != "" {
			detail.Commendations[i].Name = def.Name
		}
		if def.IconURL != "" {
			icon := def.IconURL
			detail.Commendations[i].IconURL = &icon
		}
	}
}
