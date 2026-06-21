// Package persist — post_sync_enrichment_persister.go : helper pour
// persister des colonnes de player_match_enrichment.
//
// APPEND-ONLY (#23046, 2026-06-21) : player_match_enrichment est append-only.
// BatchUpdateColumn/BatchUpdateMulti n'UPDATENT plus — ils INSÈRENT une row
// partielle taguée du `stage` propriétaire de la/des colonne(s). La lecture
// courante passe par la vue player_match_enrichment_latest (merge-on-read
// par-groupe). Plus aucun UPDATE/ON CONFLICT → vecteur ART éliminé.
//
// Les callers DOIVENT pré-filtrer via delta (sur player_match_enrichment_latest)
// pour ne pas ré-INSÉRER des rows inchangées (croissance non bornée) —
// cf. deltaSessionAssignments dans sessions_postsync_persist.go.

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

// enrichmentColumnStage mappe chaque colonne persistable vers son `stage`
// propriétaire (append-only #23046). Sert AUSSI de whitelist anti SQL injection
// (les noms sont concaténés dans la query). Doit rester aligné avec pmeColumnStage
// de la migration (internal/migration/steps_player_append_only_match_enrichment.go).
var enrichmentColumnStage = map[string]string{
	"dominance_flag":              "dominance",
	"performance_score":           "perf",
	"performance_chain":           "perf",
	"session_id":                  "session",
	"session_label":               "session",
	"is_with_friends":             "friends",
	"had_bot_teammate":            "bot",
	"teammates_signature":         "teammates",
	"engagement_score":            "engagement",
	"engagement_score_brut":       "engagement",
	"engagement_score_confidence": "engagement",
	"engagement_pace_player":      "engagement",
	"engagement_pace_team":        "engagement",
	"engagement_pace_lobby":       "engagement",
	"engagement_player_activity":  "engagement",
	"mode_category":               "engagement",
}

// deriveEnrichmentStage résout le `stage` commun d'un ensemble de colonnes.
// Erreur si une colonne est inconnue (non whitelistée) ou si les colonnes
// appartiennent à des stages différents (un INSERT partiel = un seul stage).
func deriveEnrichmentStage(columns []string) (string, error) {
	stage := ""
	for _, col := range columns {
		s, ok := enrichmentColumnStage[col]
		if !ok {
			return "", fmt.Errorf("persist: colonne %q non whitelistée", col)
		}
		if stage == "" {
			stage = s
		} else if stage != s {
			return "", fmt.Errorf("persist: colonnes de stages mixtes (%s vs %s) dans un même INSERT partiel", stage, s)
		}
	}
	if stage == "" {
		return "", errors.New("persist: aucune colonne à persister")
	}
	return stage, nil
}

// BatchUpdateColumn INSÈRE N rows partielles (1 colonne) taguées du `stage`
// propriétaire de la colonne, dans 1 transaction (append-only #23046).
// La lecture courante passe par player_match_enrichment_latest.
// Atomique : 1 TX. Si une row échoue, rollback total. No-op si rows est vide.
//
// Pour les colonnes booléennes (is_with_friends/had_bot_teammate/is_excluded),
// le caller DOIT fournir la valeur EXPLICITE (TRUE/FALSE), jamais NULL.
func (p *PostSyncEnrichmentPersister) BatchUpdateColumn(ctx context.Context, upd EnrichmentColumnUpdate) error {
	if len(upd.Rows) == 0 {
		return nil
	}
	stage, ok := enrichmentColumnStage[upd.Column]
	if !ok {
		return fmt.Errorf("persist: colonne %q non whitelistée", upd.Column)
	}

	q := fmt.Sprintf(
		`INSERT INTO player_match_enrichment (match_id, %s, stage) VALUES (?, ?, ?)`,
		upd.Column)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx enrichment batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range upd.Rows {
		if r.MatchID == "" {
			return errors.New("persist: EnrichmentColumnRow.MatchID vide")
		}
		if _, err := tx.ExecContext(ctx, q, r.MatchID, r.Value, stage); err != nil {
			return fmt.Errorf("persist: INSERT %s (match_id=%s): %w", upd.Column, r.MatchID, err)
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

// BatchUpdateMulti INSÈRE N rows partielles (plusieurs colonnes du MÊME stage)
// dans 1 transaction (append-only #23046). Toutes les rows doivent avoir le même
// set de fields (homogénéité) ET ces colonnes doivent appartenir à un seul stage.
// Atomique : 1 TX. No-op si rows est vide.
//
// Retourne len(rows) (chaque INSERT crée 1 row). Contrairement à l'ancien UPDATE,
// il n'y a plus de no-op silencieux : un match sans row pré-existante reçoit
// désormais sa row partielle (c'est le comportement voulu en append-only).
func (p *PostSyncEnrichmentPersister) BatchUpdateMulti(ctx context.Context, rows []EnrichmentMultiColumnUpdate) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	first := rows[0]
	columns := make([]string, 0, len(first.Fields))
	for col := range first.Fields {
		columns = append(columns, col)
	}
	sort.Strings(columns) // déterminisme pour debug + tests

	stage, err := deriveEnrichmentStage(columns)
	if err != nil {
		return 0, err
	}

	for i, r := range rows {
		if r.MatchID == "" {
			return 0, errors.New("persist: EnrichmentMultiColumnUpdate.MatchID vide")
		}
		if len(r.Fields) != len(columns) {
			return 0, fmt.Errorf("persist: row %d a %d fields, attendu %d (homogénéité)", i, len(r.Fields), len(columns))
		}
		for _, col := range columns {
			if _, ok := r.Fields[col]; !ok {
				return 0, fmt.Errorf("persist: row %d manque la colonne %q", i, col)
			}
		}
	}

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}
	q := fmt.Sprintf(
		`INSERT INTO player_match_enrichment (match_id, %s, stage) VALUES (?, %s, ?)`,
		strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("persist: BeginTx multi-col: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var totalInserted int64
	for _, r := range rows {
		args := make([]any, 0, len(columns)+2)
		args = append(args, r.MatchID)
		for _, col := range columns {
			args = append(args, r.Fields[col])
		}
		args = append(args, stage)
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return 0, fmt.Errorf("persist: INSERT multi-col %s (match_id=%s): %w",
				strings.Join(columns, "+"), r.MatchID, err)
		}
		totalInserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("persist: Commit multi-col: %w", err)
	}
	return totalInserted, nil
}
