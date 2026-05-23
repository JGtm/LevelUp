// Package persist — post_sync_enrichment_persister.go : Phase 4.4 du refactor
// Collect→Persist — helper pour batch-update une colonne unique de
// player_match_enrichment via UPDATE multi-row.
//
// **Problème résolu** : 5 sites post-sync (comeback, engagement, enrichments,
// friends_recompute, performance) faisaient des UPDATE row-by-row sur
// player_match_enrichment. Sous concurrence multi-joueur (pool_size=4) et
// volume (~hundreds de rows par cycle), le pattern stresse l'index ART de
// player_match_enrichment et déclenche le bug DuckDB.
//
// **Stratégie** : regrouper N UPDATE row-by-row en 1 single SQL statement
// via la syntaxe `UPDATE ... FROM (VALUES ...) AS v(...) WHERE ...` (DuckDB
// natif). Une seule opération ART au lieu de N → réduit massivement le
// stress.
//
// **Note** : ce n'est pas INSERT-only strict (l'UPDATE reste un UPDATE) mais
// la consolidation en batch unique élimine le pattern row-by-row qui était
// la vraie cause du bug ART. Refactor plus radical (schema versioning) à
// envisager dans un sprint futur si nécessaire.

package persist

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// EnrichmentColumnUpdate représente l'update d'une colonne unique sur N rows
// de player_match_enrichment. Le `Column` doit être un nom de colonne sûr
// (whitelist par les callers).
type EnrichmentColumnUpdate struct {
	Column string                // colonne cible (ex: "dominance_flag", "performance_score")
	Rows   []EnrichmentColumnRow // 1 row = 1 (match_id, value)
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
// Garde-fou anti SQL injection : les noms sont concaténés dans la query,
// donc la whitelist est obligatoire.
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

// BatchUpdateColumn exécute 1 single UPDATE pour toutes les rows fournies.
// Utilise UPDATE ... FROM (VALUES ...) AS v(match_id, value) WHERE match_id = v.match_id.
//
// Atomique : 1 transaction. Si la query échoue, aucune row modifiée.
// No-op si rows est vide.
//
// Garde-fou : column doit être dans allowedEnrichmentColumns (anti SQL injection).
func (p *PostSyncEnrichmentPersister) BatchUpdateColumn(ctx context.Context, upd EnrichmentColumnUpdate) error {
	if len(upd.Rows) == 0 {
		return nil
	}
	if !allowedEnrichmentColumns[upd.Column] {
		return fmt.Errorf("persist: colonne %q non whitelistée", upd.Column)
	}

	// Construire la clause VALUES (?, ?), (?, ?), ...
	placeholders := make([]string, len(upd.Rows))
	args := make([]any, 0, len(upd.Rows)*2)
	for i, r := range upd.Rows {
		if r.MatchID == "" {
			return errors.New("persist: EnrichmentColumnRow.MatchID vide")
		}
		placeholders[i] = "(?, ?)"
		args = append(args, r.MatchID, r.Value)
	}
	valuesClause := strings.Join(placeholders, ", ")

	// UPDATE en single statement. La clause VALUES est inline (DuckDB natif).
	q := fmt.Sprintf(`
		UPDATE player_match_enrichment AS pme
		SET %s = v.new_value,
		    updated_at = CURRENT_TIMESTAMP
		FROM (VALUES %s) AS v(match_id, new_value)
		WHERE pme.match_id = v.match_id`,
		upd.Column, valuesClause)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx enrichment batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("persist: UPDATE batch %s (n=%d): %w", upd.Column, len(upd.Rows), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit enrichment batch %s: %w", upd.Column, err)
	}
	return nil
}

// EnrichmentMultiColumnUpdate représente l'update de plusieurs colonnes sur
// 1 row identifiée par MatchID. Permet de regrouper plusieurs colonnes en
// 1 single UPDATE statement par cycle (ex: Performance updatent perf_score
// + perf_chain, Engagement update 8 colonnes).
type EnrichmentMultiColumnUpdate struct {
	MatchID string
	Fields  map[string]any // {colonne: valeur} — toutes whitelistées
}

// BatchUpdateMulti exécute 1 single UPDATE multi-row multi-colonnes via
// la syntaxe `UPDATE ... FROM (VALUES ...) AS v(...) WHERE`. Tous les fields
// doivent être dans la whitelist allowedEnrichmentColumns. Toutes les rows
// doivent avoir le même set de fields (sinon erreur — homogénéité requise).
//
// 1 single UPDATE → 1 seule opération ART, indépendamment du nombre de rows
// et de colonnes. Atomic via TX.
func (p *PostSyncEnrichmentPersister) BatchUpdateMulti(ctx context.Context, rows []EnrichmentMultiColumnUpdate) error {
	if len(rows) == 0 {
		return nil
	}

	// Toutes les rows doivent avoir le MÊME set de colonnes (homogénéité).
	// Sinon la syntaxe UPDATE FROM VALUES ne marche pas (cols fixes par row).
	first := rows[0]
	columns := make([]string, 0, len(first.Fields))
	for col := range first.Fields {
		if !allowedEnrichmentColumns[col] {
			return fmt.Errorf("persist: colonne %q non whitelistée", col)
		}
		columns = append(columns, col)
	}
	// Tri pour déterminisme (l'ordre des cols dans VALUES et SET doit matcher).
	// Pas critique pour la correction (Go map est random) mais utile pour
	// debug + tests reproductibles.
	for i := 0; i < len(columns); i++ {
		for j := i + 1; j < len(columns); j++ {
			if columns[i] > columns[j] {
				columns[i], columns[j] = columns[j], columns[i]
			}
		}
	}

	// Vérifier homogénéité + construire args.
	args := make([]any, 0, len(rows)*(1+len(columns)))
	placeholders := make([]string, len(rows))
	for i, r := range rows {
		if r.MatchID == "" {
			return errors.New("persist: EnrichmentMultiColumnUpdate.MatchID vide")
		}
		if len(r.Fields) != len(columns) {
			return fmt.Errorf("persist: row %d a %d fields, attendu %d (homogénéité)", i, len(r.Fields), len(columns))
		}
		rowPlaceholders := make([]string, 1+len(columns))
		rowPlaceholders[0] = "?"
		args = append(args, r.MatchID)
		for j, col := range columns {
			val, ok := r.Fields[col]
			if !ok {
				return fmt.Errorf("persist: row %d manque la colonne %q", i, col)
			}
			rowPlaceholders[1+j] = "?"
			args = append(args, val)
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ", ") + ")"
	}

	// Construire la clause SET dynamiquement.
	setClauses := make([]string, len(columns))
	for i, col := range columns {
		setClauses[i] = fmt.Sprintf("%s = v.%s", col, col)
	}
	setSQL := strings.Join(setClauses, ", ")

	// Construire la clause `AS v(match_id, col1, col2, ...)`.
	vColNames := append([]string{"match_id"}, columns...)
	vAlias := "v(" + strings.Join(vColNames, ", ") + ")"

	valuesClause := strings.Join(placeholders, ", ")

	q := fmt.Sprintf(`
		UPDATE player_match_enrichment AS pme
		SET %s,
		    updated_at = CURRENT_TIMESTAMP
		FROM (VALUES %s) AS %s
		WHERE pme.match_id = v.match_id`,
		setSQL, valuesClause, vAlias)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx multi-col: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("persist: UPDATE batch multi-col (n=%d, cols=%d): %w", len(rows), len(columns), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit multi-col: %w", err)
	}
	return nil
}
