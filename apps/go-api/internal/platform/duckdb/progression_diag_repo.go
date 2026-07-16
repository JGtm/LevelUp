// Package duckdb — progression_diag_repo.go : lectures diagnostic des tables
// progression V2 (Ascension) pour le handler /api/v1/_diag/progression.
//
// Phase 4 plan stabilisation 2026-05-22. Lecture pure (counts), pas de write.
// Source de vérité par table :
//   - streak_latest          : player DB (vue append-only des streaks)
//   - player_records_latest  : shared_social.duckdb (vue append-only des PB)
//   - record_history         : player DB
//   - milestone_earned       : player DB
//   - milestone_catalog      : metadata.duckdb (référentiel)
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// ProgressionDiagRepo lit les counts V2 pour un joueur.
type ProgressionDiagRepo struct {
	pdb *PlayerDB
}

// NewProgressionDiagRepo construit le repo.
func NewProgressionDiagRepo(pdb *PlayerDB) *ProgressionDiagRepo {
	return &ProgressionDiagRepo{pdb: pdb}
}

// GetProgressionDiag retourne un snapshot des counts V2. Satisfait
// handlers.ProgressionDiagProvider par signature (duck typing — assertion
// compile-time côté handler via la factory).
func (r *ProgressionDiagRepo) GetProgressionDiag(ctx context.Context, slug string) (*domain.ProgressionDiag, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, fmt.Errorf("ProgressionDiagRepo: PlayerDB nil")
	}
	out := &domain.ProgressionDiag{PlayerSlug: slug}

	// Counts player DB — best-effort, table peut être absente sur DB legacy.
	// streak_latest = vue append-only (streak n'est plus écrite par le repo).
	out.StreakCount = countTableBestEffort(ctx, r.pdb.ReadDB(), "streak_latest")
	out.RecordHistoryCount = countTableBestEffort(ctx, r.pdb.ReadDB(), "record_history")
	out.MilestoneEarnedCount = countTableBestEffort(ctx, r.pdb.ReadDB(), "milestone_earned")

	// Count PB courants via la vue append-only player_records_latest (et non
	// la table legacy player_records, qui n'est plus écrite par la progression).
	// Filtré par xuid pour scoping joueur.
	if r.pdb.SharedSocial != nil {
		_ = r.pdb.SharedSocial.QueryRow(ctx,
			`SELECT COUNT(*) FROM player_records_latest WHERE xuid = ?`, r.pdb.XUID).Scan(&out.PlayerRecordsCount)
	}

	// Count milestone_catalog dans metadata (référentiel cross-joueurs).
	if r.pdb.Metadata != nil {
		_ = r.pdb.Metadata.QueryRow(ctx,
			`SELECT COUNT(*) FROM milestone_catalog`).Scan(&out.MilestoneCatalogCount)
	}

	// Timestamp dernier post-sync (best-effort).
	if r.pdb.Player != nil {
		var ts sql.NullString
		_ = r.pdb.ReadDB().QueryRow(ctx,
			`SELECT value FROM sync_meta WHERE key = 'last_post_sync_at'`).Scan(&ts)
		if ts.Valid {
			out.PipelineWiredAt = ts.String
		}
	}

	return out, nil
}

// countTableBestEffort compte les rows d'une table ; retourne 0 si la table
// n'existe pas (DB legacy avant migration V2). Idiomatic best-effort.
func countTableBestEffort(ctx context.Context, db interface {
	QueryRowRecovered(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}, table string) int {
	var n int
	rows, err := db.QueryRowRecovered(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table))
	if err != nil {
		return 0 // table absente / Reopen concurrent → n reste 0 (best-effort)
	}
	defer rows.Close()
	_ = rows.Scan(&n) // erreur de scan → n reste 0
	return n
}
