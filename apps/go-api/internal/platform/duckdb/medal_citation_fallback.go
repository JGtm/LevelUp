// Package duckdb — medal_citation_fallback.go : source unique du fallback
// citation_mappings pour les libellés de médailles absents (ou vides) dans
// medal_definitions.
//
// Pourquoi : trois résolveurs de médailles complètent les IDs manquants via
// citation_mappings.citation_name_display — la vue Match (lookupMedalMeta),
// l'Explorer/Squad (MedalDefinitionsRepo.LookupByIDs) et la tuile de match Home
// (resolveMedalLabels). Avant ce helper, seule la vue Match avait le fallback ;
// les deux autres affichaient des libellés vides. Centralisé ici pour respecter
// la règle DRY ≤2 copies du CLAUDE.md (pas de 3e copie de la requête).
package duckdb

import (
	"context"
	"log/slog"
)

// medalCitationFallbackQuery résout medal_id → citation_name_norm pour les IDs donnés ; le nom, dans
// la langue de la requête, vient ensuite de citationDisplayNames. Le %s est rempli par
// buildLookupQuery (placeholders sûrs).
const medalCitationFallbackQuery = `SELECT medal_id, citation_name_norm
	 FROM citation_mappings
	 WHERE medal_id IN (%s)`

// lookupMedalCitationLabels retourne medal_id → libellé issu de citation_mappings
// pour les IDs donnés (fallback quand medal_definitions n'a pas de nom
// exploitable). Map vide, sans erreur, si db est nil, ids vide, aucune ligne, ou si la table
// `citation_mappings` n'existe pas dans cette base (référentiel propre à un titre : absence
// constatée par sonde). Une sonde ou une lecture en échec rend les libellés lus ET l'erreur : les
// appelants (vue Match, Explorer, Home) la journalisent ou la disent avant de dégrader.
//
// Observabilité : émet un log DEBUG (requested vs resolved) à chaque invocation
// du fallback : sous LEVELUP_LOG_LEVEL=debug, resolved=0 alors que requested>0 trahit une
// citation_mappings vide. Le PC d'appel étant dans le package duckdb,
// le MultiModuleHandler route ce log vers logs/duckdb.log (cf.
// observability/logging/module.go : detectModuleFromCaller).
func lookupMedalCitationLabels(ctx context.Context, db *DB, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 || db == nil {
		return map[int64]string{}, nil
	}
	present, err := relationExists(ctx, db.SQLDb(), "citation_mappings")
	if err != nil {
		return map[int64]string{}, err
	}
	if !present {
		slog.DebugContext(ctx, "medal_citation_fallback: citation_mappings absente de cette base",
			slog.Int("requested", len(ids)))
		return map[int64]string{}, nil
	}
	out, err := medalCitationNames(ctx, db, ids)
	slog.DebugContext(ctx, "medal_citation_fallback",
		slog.Int("requested", len(ids)),
		slog.Int("resolved", len(out)),
	)
	return out, err
}

// medalCitationNames range pour chaque médaille le nom de sa citation dans la langue de la requête.
func medalCitationNames(ctx context.Context, db *DB, ids []int64) (map[int64]string, error) {
	normByMedal, err := lookupLabelsByID(ctx, db, medalCitationFallbackQuery, ids)
	if err != nil {
		return map[int64]string{}, err
	}
	norms := make([]string, 0, len(normByMedal))
	for _, n := range normByMedal {
		norms = append(norms, n)
	}
	names, err := citationDisplayNames(ctx, db, norms)
	out := make(map[int64]string, len(normByMedal))
	for id, norm := range normByMedal {
		if name := names[norm]; name != "" {
			out[id] = name
		}
	}
	return out, err
}

// citationLabelsOrLog est lookupMedalCitationLabels pour les appelants qui dégradent (libellés
// repliés) : l'échec est journalisé en ERREUR (surface, err) AVANT la dégradation, les libellés
// lus sont rendus.
func citationLabelsOrLog(ctx context.Context, db *DB, ids []int64, surface string) map[int64]string {
	out, err := lookupMedalCitationLabels(ctx, db, ids)
	if err != nil {
		slog.ErrorContext(ctx, surface+": libellés de médailles par citation indisponibles", "err", err)
	}
	return out
}
