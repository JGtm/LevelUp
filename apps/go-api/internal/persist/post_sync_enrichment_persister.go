// Package persist — post_sync_enrichment_persister.go : helper pour
// batch-update des colonnes de player_match_enrichment.
//
// Stratégie anti-ART (cf. ADR 0019) : N UPDATE row-by-row dans 1 seule
// transaction. Chaque statement ne touche qu'1 entrée ART → évite le bug
// "Failed to delete all rows from index" que déclenchait la syntaxe
// UPDATE FROM (VALUES ...) multi-row (bulk ART delete sur N entrées).
//
// Les callers de BatchUpdateMulti / BatchUpdateColumn doivent pré-filtrer
// via delta pour minimiser le nombre de rows réellement écrites
// (cf. deltaSessionAssignments dans sessions_postsync_persist.go).

package persist

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// EnrichmentColumnUpdate représente l'update d'une colonne unique sur N rows
// de player_match_enrichment. Le Column doit être dans allowedEnrichmentColumns.
type EnrichmentColumnUpdate struct {
	Column string
	Rows   []EnrichmentColumnRow
}

// EnrichmentColumnRow — 1 ligne à updater. Value peut être nil → NULL en SQL.
type EnrichmentColumnRow struct {
	MatchID string
	Value   any
}

// PostSyncEnrichmentPersister exécute des UPDATE batch sur
// player_match_enrichment. Une instance par playerDB (single writer assumed
// via dblease).
type PostSyncEnrichmentPersister struct {
	db txBeginner
}

// NewPostSyncEnrichmentPersister construit un persister.
func NewPostSyncEnrichmentPersister(db txBeginner) *PostSyncEnrichmentPersister {
	return &PostSyncEnrichmentPersister{db: db}
}

// allowedEnrichmentColumns liste les colonnes que ce persister peut updater.
// Garde-fou anti SQL injection : les noms sont concaténés dans la query.
var allowedEnrichmentColumns = map[string]bool{
	"dominance_flag":              true,
	"performance_score":           true,
	"performance_chain":           true,
	"session_id":                  true,
	"session_label":               true,
	"is_with_friends":             true,
	"had_bot_teammate":            true,
	"teammates_signature":         true,
	"engagement_score":            true,
	"engagement_score_brut":       true,
	"engagement_score_confidence": true,
	"engagement_pace_player":      true,
	"engagement_pace_team":        true,
	"engagement_pace_lobby":       true,
	"engagement_player_activity":  true,
	"mode_category":               true,
}

// BatchUpdateColumn exécute N UPDATE row-by-row dans 1 transaction.
// Chaque UPDATE ne touche qu'1 entrée ART → évite le bug DuckDB ART
// "Failed to delete all rows from index" (cf. ADR 0019).
// Atomique : 1 TX. Si une row échoue, rollback total.
// No-op si rows est vide.
func (p *PostSyncEnrichmentPersister) BatchUpdateColumn(ctx context.Context, upd EnrichmentColumnUpdate) error {
	if len(upd.Rows) == 0 {
		return nil
	}
	if !allowedEnrichmentColumns[upd.Column] {
		return fmt.Errorf("persist: colonne %q non whitelistée", upd.Column)
	}

	q := fmt.Sprintf(`
		UPDATE player_match_enrichment
		SET %s = ?, updated_at = CURRENT_TIMESTAMP
		WHERE match_id = ?`, upd.Column)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx enrichment batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range upd.Rows {
		if r.MatchID == "" {
			return errors.New("persist: EnrichmentColumnRow.MatchID vide")
		}
		if _, err := tx.ExecContext(ctx, q, r.Value, r.MatchID); err != nil {
			return fmt.Errorf("persist: UPDATE %s (match_id=%s): %w", upd.Column, r.MatchID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit enrichment batch %s: %w", upd.Column, err)
	}
	return nil
}

// EnrichmentMultiColumnUpdate représente l'update de plusieurs colonnes sur
// 1 row identifiée par MatchID.
type EnrichmentMultiColumnUpdate struct {
	MatchID string
	Fields  map[string]any // {colonne: valeur} — toutes whitelistées
}

// BatchUpdateMulti exécute N UPDATE row-by-row dans 1 transaction.
// Chaque UPDATE ne touche qu'1 entrée ART → évite le bug DuckDB ART
// "Failed to delete all rows from index" (cf. ADR 0019).
// Toutes les rows doivent avoir le même set de fields (homogénéité).
// Atomique : 1 TX. No-op si rows est vide.
func (p *PostSyncEnrichmentPersister) BatchUpdateMulti(ctx context.Context, rows []EnrichmentMultiColumnUpdate) error {
	if len(rows) == 0 {
		return nil
	}

	first := rows[0]
	columns := make([]string, 0, len(first.Fields))
	for col := range first.Fields {
		if !allowedEnrichmentColumns[col] {
			return fmt.Errorf("persist: colonne %q non whitelistée", col)
		}
		columns = append(columns, col)
	}
	sort.Strings(columns) // déterminisme pour debug + tests

	for i, r := range rows {
		if r.MatchID == "" {
			return errors.New("persist: EnrichmentMultiColumnUpdate.MatchID vide")
		}
		if len(r.Fields) != len(columns) {
			return fmt.Errorf("persist: row %d a %d fields, attendu %d (homogénéité)", i, len(r.Fields), len(columns))
		}
		for _, col := range columns {
			if _, ok := r.Fields[col]; !ok {
				return fmt.Errorf("persist: row %d manque la colonne %q", i, col)
			}
		}
	}

	setClauses := make([]string, len(columns))
	for i, col := range columns {
		setClauses[i] = col + " = ?"
	}
	q := fmt.Sprintf(`
		UPDATE player_match_enrichment
		SET %s, updated_at = CURRENT_TIMESTAMP
		WHERE match_id = ?`, strings.Join(setClauses, ", "))

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx multi-col: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range rows {
		args := make([]any, 0, len(columns)+1)
		for _, col := range columns {
			args = append(args, r.Fields[col])
		}
		args = append(args, r.MatchID)
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return fmt.Errorf("persist: UPDATE multi-col %s (match_id=%s): %w",
				strings.Join(columns, "+"), r.MatchID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit multi-col: %w", err)
	}
	return nil
}
