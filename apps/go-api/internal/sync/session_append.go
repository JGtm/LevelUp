// Package sync — session_append.go : algorithme incrémental de sessions.
//
// Principes de la session incrémentale :
//   - Une session fermée (session K < max) est immuable — jamais relue, jamais réécrite.
//   - Seule la session courante (la dernière, encore potentiellement ouverte) peut
//     recevoir de nouveaux matchs.
//   - L'ancre = dernier match assigné (par heure) : sert de point de départ pour
//     calculer si les nouveaux matchs continuent la session courante ou en démarrent une.
//   - Seuls les nouveaux matchs (session_id IS NULL) et les labels mis à jour (si la
//     session courante est étendue) sont écrits.
//
// Complexité : O(new_matches) en écriture au lieu de O(all_matches). Typiquement
// 5-20 writes au lieu de 700 pour un sync de 5 nouveaux matchs.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// assignedEntry représente l'état actuel d'un match dans player_match_enrichment.
type assignedEntry struct {
	sessionID    int
	sessionLabel string
}

// appendSessionsInline calcule et persiste les sessions de façon incrémentale.
// Fallback vers fullSessionCompute si aucun match n'est encore assigné (premier sync).
func appendSessionsInline(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	opts domain.SessionComputeOptions,
) (int, error) {
	// 1. Charger les assignments existants depuis playerDB
	existing, maxSessionID, err := loadAssignedSessionsMap(ctx, playerDB)
	if err != nil {
		return 0, fmt.Errorf("appendSessionsInline: load existing: %w", err)
	}

	// 2. Charger tous les matchs depuis shared (nécessaire pour l'ancre et les labels)
	allRows, err := loadSessionMatchRowsDirect(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("appendSessionsInline: load rows: %w", err)
	}
	if len(allRows) == 0 {
		return 0, nil
	}

	// 3. Séparer l'ancre (dernier match assigné par heure) et les nouveaux matchs.
	// allRows est trié ASC par start_time — le dernier match assigné est le plus récent.
	var anchor *domain.SessionMatchRow
	var newRows []domain.SessionMatchRow
	for i := range allRows {
		r := &allRows[i]
		if _, ok := existing[r.MatchID]; ok {
			anchor = r // mise à jour : reste le plus récent (dernier itéré = dernier par temps)
		} else {
			newRows = append(newRows, *r)
		}
	}

	// 4. Pas d'ancre → premier sync, calcul complet
	if anchor == nil {
		return fullSessionCompute(ctx, playerDB, allRows, opts)
	}

	// 5. Aucun nouveau match
	if len(newRows) == 0 {
		return 0, nil
	}

	// 6. Détection hors-ordre : si un nouveau match est antérieur à l'ancre, il
	// appartient à une session passée fermée — l'algorithme incrémental ne peut pas
	// lui attribuer le bon session_id. Fallback vers calcul complet.
	for _, r := range newRows {
		if r.StartTime.Before(anchor.StartTime) {
			slog.WarnContext(ctx, "appendSessionsInline: match antérieur à l'ancre — fallback full compute",
				"match_id", r.MatchID, "match_time", r.StartTime, "anchor_time", anchor.StartTime)
			return fullSessionCompute(ctx, playerDB, allRows, opts)
		}
	}

	// 7. Calculer les sessions sur [ancre] + nouveaux matchs
	combined := make([]domain.SessionMatchRow, 0, 1+len(newRows))
	combined = append(combined, *anchor)
	combined = append(combined, newRows...)
	rawAssignments := analysis.ComputeSessionsWithContext(combined, opts)
	// rawAssignments[0] = ancre (session calculée = 0, à ignorer)
	// rawAssignments[1:] = nouveaux matchs (session 0 = continuation, 1+ = nouvelle)

	// 8. Appliquer l'offset : session_id_réel = session_id_calculé + maxSessionID
	newAssignments := make([]domain.SessionAssignment, 0, len(newRows))
	for _, a := range rawAssignments[1:] {
		newAssignments = append(newAssignments, domain.SessionAssignment{
			MatchID:   a.MatchID,
			SessionID: a.SessionID + maxSessionID,
		})
	}

	// 9. Construire les groupes pour les labels.
	// Si des nouveaux matchs rejoignent la session courante (maxSessionID), inclure
	// les matchs existants dans ce groupe pour que le label reflète le total correct.
	currentExtended := false
	for _, a := range newAssignments {
		if a.SessionID == maxSessionID {
			currentExtended = true
			break
		}
	}

	affectedRows := make([]domain.SessionMatchRow, 0, len(allRows))
	affectedAssignments := make([]domain.SessionAssignment, 0, len(allRows))

	if currentExtended {
		for _, r := range allRows {
			e, ok := existing[r.MatchID]
			if ok && e.sessionID == maxSessionID {
				affectedRows = append(affectedRows, r)
				affectedAssignments = append(affectedAssignments, domain.SessionAssignment{
					MatchID: r.MatchID, SessionID: maxSessionID,
				})
			}
		}
	}

	newRowByID := make(map[string]domain.SessionMatchRow, len(newRows))
	for _, r := range newRows {
		newRowByID[r.MatchID] = r
	}
	for _, a := range newAssignments {
		if r, ok := newRowByID[a.MatchID]; ok {
			affectedRows = append(affectedRows, r)
			affectedAssignments = append(affectedAssignments, a)
		}
	}

	// 10. Labels pour les sessions affectées
	groups := analysis.BuildSessionGroups(affectedRows, affectedAssignments)
	withLabels := analysis.MergeSessionLabels(affectedAssignments, groups)

	// 11. Persister — le delta filter dans writeSessionAssignmentsBatch ne garde
	// que les lignes réellement changées (nouveaux matchs + labels mis à jour si étendus)
	return writeSessionAssignmentsBatch(ctx, playerDB, withLabels)
}

// loadAssignedSessionsMap charge (match_id → {session_id, session_label}) depuis
// player_match_enrichment pour les matchs déjà assignés. Retourne aussi le max session_id.
func loadAssignedSessionsMap(ctx context.Context, db *sql.DB) (map[string]assignedEntry, int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT match_id,
		       CAST(session_id AS INT),
		       COALESCE(session_label, '')
		FROM player_match_enrichment
		WHERE session_id IS NOT NULL
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("loadAssignedSessionsMap: %w", err)
	}
	defer rows.Close()

	result := make(map[string]assignedEntry)
	maxID := 0
	for rows.Next() {
		var matchID string
		var sid int
		var label string
		if err := rows.Scan(&matchID, &sid, &label); err != nil {
			return nil, 0, fmt.Errorf("loadAssignedSessionsMap scan: %w", err)
		}
		result[matchID] = assignedEntry{sessionID: sid, sessionLabel: label}
		if sid > maxID {
			maxID = sid
		}
	}
	return result, maxID, rows.Err()
}

// fullSessionCompute est le fallback premier-sync : calcule et persiste toutes les sessions.
// Utilisé quand aucune ancre n'existe (premier sync joueur) ou lors d'une découverte
// de match hors-ordre (match antérieur à l'ancre).
func fullSessionCompute(ctx context.Context, playerDB *sql.DB, allRows []domain.SessionMatchRow, opts domain.SessionComputeOptions) (int, error) {
	assignments := analysis.ComputeSessionsWithContext(allRows, opts)
	groups := analysis.BuildSessionGroups(allRows, assignments)
	assignments = analysis.MergeSessionLabels(assignments, groups)
	return writeSessionAssignmentsBatch(ctx, playerDB, assignments)
}
