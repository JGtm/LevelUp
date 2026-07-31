package main

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// conn encapsule la connexion shared (RO ou RW) + son releaser. En mode RW on
// garde le *duckdb.DB pour pouvoir construire un PlayerDB pour le repo ; en RO
// rwDB est nil et seul sqlDB (lecture) est utilisé.
type conn struct {
	sqlDB   *sql.DB
	rwDB    *duckdb.DB
	release func()
}

func (c *conn) close() {
	if c.release != nil {
		c.release()
	}
}

// openConn ouvre la DB shared. write=false → lecture seule (OpenReadForQuery,
// safe avec un serveur en cours). write=true → OpenReadWrite (lock exclusif),
// expose aussi le *duckdb.DB pour le repo.
func openConn(path string, write bool) (*conn, error) {
	if !write {
		sqlDB, release, err := duckdb.OpenReadForQuery(path)
		if err != nil {
			return nil, fmt.Errorf("open RO %s: %w", path, err)
		}
		return &conn{sqlDB: sqlDB, release: release}, nil
	}
	rw, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("open RW %s: %w", path, err)
	}
	return &conn{sqlDB: rw.SQLDb(), rwDB: rw, release: func() { _ = rw.Close() }}, nil
}

// ensureObjectiveEventsTables applique la SEULE migration des tables v3
// (shared_objective_events_v1) en direct, sans dérouler tout le chain shared.
// CREATE IF NOT EXISTS → idempotent et sans risque sur une DB existante.
func ensureObjectiveEventsTables(db *sql.DB) error {
	return ensureMigration(db, "shared_objective_events_v1")
}

// ensureMigration applique une migration nommée du registre en direct (sans
// dérouler tout le chain shared). CREATE IF NOT EXISTS → idempotent, sans risque
// sur une DB existante. Sert au -write des tables shadow (objective-events, v3).
func ensureMigration(db *sql.DB, name string) error {
	for _, m := range migration.All() {
		if m.Name == name {
			return m.ApplySchema(db)
		}
	}
	return fmt.Errorf("migration %q introuvable dans le registre", name)
}

// registryRow = sous-ensemble de match_registry utile au diagnostic.
type registryRow struct {
	matchID    string
	variant    string
	team0Score int
	team1Score int
	hasScores  bool
}

// loadRegistry lit la ligne match_registry du match (UUID complet). Renvoie
// (nil,false,nil) si le match est absent du registre.
func loadRegistry(ctx context.Context, db *sql.DB, matchID string) (*registryRow, bool, error) {
	var (
		variant sql.NullString
		t0, t1  sql.NullInt64
	)
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(game_variant_name, ''), team_0_score, team_1_score
		FROM match_registry WHERE match_id = ?`, matchID).Scan(&variant, &t0, &t1)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("loadRegistry(%s): %w", matchID, err)
	}
	return &registryRow{
		matchID:    matchID,
		variant:    variant.String,
		team0Score: int(t0.Int64),
		team1Score: int(t1.Int64),
		hasScores:  t0.Valid && t1.Valid,
	}, true, nil
}

// loadRoster construit le mapping xuid->team_id depuis match_participants.
func loadRoster(ctx context.Context, db *sql.DB, matchID string) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, team_id FROM match_participants
		WHERE match_id = ? AND team_id IS NOT NULL`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadRoster(%s): %w", matchID, err)
	}
	defer rows.Close()

	roster := make(map[string]int)
	for rows.Next() {
		var (
			xuid string
			team int
		)
		if err := rows.Scan(&xuid, &team); err != nil {
			return nil, fmt.Errorf("loadRoster scan: %w", err)
		}
		roster[xuid] = team
	}
	return roster, rows.Err()
}

// resolveFullMatchID résout un argument -match (préfixe 8-char OU UUID complet)
// vers l'UUID complet du registre via LIKE prefix%. Renvoie ("",false,nil) si
// rien ne matche.
func resolveFullMatchID(ctx context.Context, db *sql.DB, arg string) (string, bool, error) {
	var matchID string
	err := db.QueryRowContext(ctx,
		`SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY match_id LIMIT 1`,
		arg+"%").Scan(&matchID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("resolveFullMatchID(%s): %w", arg, err)
	}
	return matchID, true, nil
}
