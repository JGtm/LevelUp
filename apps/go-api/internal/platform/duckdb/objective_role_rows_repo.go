// Package duckdb — objective_role_rows_repo.go : lignes (match, joueur) de
// `match_objective_stats_latest` projetées par RÔLE (prendre / défendre / tenir)
// pour le bloc objectifs de l'agrégat de session (chantier session-usage S2).
//
// Les sommes par rôle sont GÉNÉRÉES depuis narrative.ObjectiveRoleColumns
// (source unique de la classification, objective_roles.go — même doctrine que
// objectiveIndexSelectColumns) : aucune liste de colonnes locale. La famille de
// la ligne vient d'objectiveFamilyCaseSQL, le même discriminant que l'index de
// participation. LES DEUX CAMPS sont chargés (aucun filtre xuid) : les parts
// joueur/camp/lobby se calculent côté analysis (sessionusage.ComputeObjectives).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
)

// objectiveRoleSumSQL — l'expression SUM d'un rôle : COALESCE(col,0) additionnées
// (les colonnes des autres familles sont NULL sur une ligne — 0 par COALESCE).
func objectiveRoleSumSQL(role narrative.ObjectiveRole) string {
	cols := narrative.ObjectiveRoleColumns(role)
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, "COALESCE(o."+c+", 0)")
	}
	return "(" + strings.Join(parts, " + ") + ")::DOUBLE"
}

// LoadObjectiveRoleRows retourne, sur un scope fermé de matchs, les lignes
// (match, joueur, famille) projetées par rôle — les deux camps. Lecture par la
// vue `_latest` UNIQUEMENT (ADR 0026). Best-effort : une erreur de requête (vue
// absente sur une DB non migrée) dégrade en nil + warn — le bloc objectifs de la
// page est alors omis, jamais un échec dur (même politique que
// LoadObjectiveIndexInputs).
func (r *ObjectiveStatsRepo) LoadObjectiveRoleRows(
	ctx context.Context, matchIDs []string,
) ([]sessionusage.ObjectiveRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()

	q := "SELECT o.match_id, o.xuid, " + objectiveFamilyCaseSQL() + ` AS family,
		` + objectiveRoleSumSQL(narrative.ObjectiveRoleTake) + `,
		` + objectiveRoleSumSQL(narrative.ObjectiveRoleDefend) + `,
		` + objectiveRoleSumSQL(narrative.ObjectiveRoleHold) + `
		FROM match_objective_stats_latest o
		WHERE o.match_id IN (` + Placeholders(len(matchIDs)) + `)`

	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "ObjectiveStatsRepo: role rows query failed (best-effort)",
			"match_count", len(matchIDs), "err", err)
		return nil, nil
	}
	defer rows.Close()

	var out []sessionusage.ObjectiveRow
	for rows.Next() {
		var row sessionusage.ObjectiveRow
		var family *string
		if err := rows.Scan(&row.MatchID, &row.XUID, &family,
			&row.Take, &row.Defend, &row.HoldSeconds); err != nil {
			return nil, fmt.Errorf("ObjectiveStatsRepo: role rows scan: %w", err)
		}
		// Famille NULL = ligne sans aucun bloc objectif — hors périmètre.
		if family == nil {
			continue
		}
		row.Family = narrative.ObjectiveFamily(*family)
		out = append(out, row)
	}
	return out, rows.Err()
}
