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

// medalCitationFallbackQuery résout medal_id → citation_name_display pour les
// IDs donnés. Le %s est rempli par buildLookupQuery (placeholders sûrs).
const medalCitationFallbackQuery = `SELECT medal_id, citation_name_display
	 FROM citation_mappings
	 WHERE medal_id IN (%s)
	   AND citation_name_display IS NOT NULL
	   AND citation_name_display <> ''`

// lookupMedalCitationLabels retourne medal_id → libellé issu de citation_mappings
// pour les IDs donnés (fallback quand medal_definitions n'a pas de nom
// exploitable). Map vide si db est nil, ids vide, ou aucune ligne — jamais nil.
//
// Observabilité : émet un log DEBUG (requested vs resolved) à chaque invocation
// du fallback. Comme lookupLabelsByID dégrade silencieusement (erreur SQL =
// map vide, contrat partagé avec les armes), ce compteur est le SEUL moyen de
// diagnostiquer le symptôme « médailles affichées en ID nu » : sous
// LEVELUP_LOG_LEVEL=debug, resolved=0 alors que requested>0 trahit une
// citation_mappings vide/absente. Le PC d'appel étant dans le package duckdb,
// le MultiModuleHandler route ce log vers logs/duckdb.log (cf.
// observability/logging/module.go : detectModuleFromCaller).
func lookupMedalCitationLabels(ctx context.Context, db *DB, ids []int64) map[int64]string {
	if len(ids) == 0 || db == nil {
		return map[int64]string{}
	}
	out := lookupLabelsByID(ctx, db, medalCitationFallbackQuery, ids)
	slog.DebugContext(ctx, "medal_citation_fallback",
		slog.Int("requested", len(ids)),
		slog.Int("resolved", len(out)),
	)
	return out
}
