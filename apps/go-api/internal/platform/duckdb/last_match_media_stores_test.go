//go:build integration

// last_match_media_stores_test.go : LastMatchReader et MediaRecipientsRepo contre de vraies bases migrées, et le
// helper qui les prépare (une base migrée rendue fermée, que le dépôt testé ouvre lui-même).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/port"
)

// migratedFile crée, au chemin donné, une base migrée pour `target` puis y joue `stmts` ; elle est rendue
// fermée (aucun handle en cache) pour que le dépôt testé l'ouvre lui-même.
func migratedFile(t *testing.T, path string, target migration.TargetDB, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	if err := migration.RunForDB(raw, target); err != nil {
		_ = raw.Close()
		t.Fatalf("migrate %s: %v", target, err)
	}
	for _, s := range stmts {
		if _, err := raw.Exec(s); err != nil {
			_ = raw.Close()
			t.Fatalf("exec %.60s: %v", s, err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	EvictAndCloseCached(path)
	t.Cleanup(func() { EvictAndCloseCached(path) })
}

func TestLastMatchReader_UneRequeteGroupee(t *testing.T) {
	root := t.TempDir()
	pr := titlepkg.NewPathResolver(root)
	slug := titlepkg.DefaultSlug
	t1 := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	migratedFile(t, pr.SharedDBPath(slug), migration.TargetShared,
		`INSERT INTO match_registry (match_id, start_time, start_time_utc) VALUES
			('m1', TIMESTAMP '2026-07-10 08:00:00', TIMESTAMP '2026-07-10 08:00:00'),
			('m2', TIMESTAMP '2026-07-15 12:00:00', TIMESTAMP '2026-07-15 12:00:00')`,
		`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES
			('m1', 'xuid-A', 'A'), ('m2', 'xuid-A', 'A'), ('m1', 'xuid-B', 'B')`)
	r := LastMatchReader{Log: slog.Default()}

	out, err := r.LastMatchByXUID(context.Background(), pr.SharedDBPath(slug), slug, []string{"xuid-A", "xuid-B", "xuid-C"})
	if err != nil {
		t.Fatalf("LastMatchByXUID: %v", err)
	}
	a, b := out["xuid-A"], out["xuid-B"]
	if len(out) != 2 || a.Sub(b) != t2.Sub(t1) {
		t.Errorf("résultat = %v : attendu A et B (écart %v), sans C", out, t2.Sub(t1))
	}
	if out, err := r.LastMatchByXUID(context.Background(), "", slug, nil); err != nil || len(out) != 0 {
		t.Errorf("sans joueur : %v, %v (attendu vide, sans ouvrir de base)", out, err)
	}
	if _, err := r.LastMatchByXUID(context.Background(), filepath.Join(root, "absente.duckdb"), slug, []string{"xuid-A"}); !errors.Is(err, ErrLastMatchRead) {
		t.Errorf("base absente : %v, attendu ErrLastMatchRead", err)
	}
}

func TestMediaRecipientsRepo_MatchsAssociesEtParticipants(t *testing.T) {
	root := t.TempDir()
	socialPath := filepath.Join(root, "social.duckdb")
	sharedPath := filepath.Join(root, "shared.duckdb")
	now := time.Now().UTC()
	migratedFile(t, socialPath, migration.TargetSharedSocial,
		`INSERT INTO media_match_associations_history (media_file_id, match_id, associated_at) VALUES
			(1, 'm-recent', now() AT TIME ZONE 'UTC'),
			(2, 'm-ancien', TIMESTAMP '2020-01-01 00:00:00')`)
	migratedFile(t, sharedPath, migration.TargetShared,
		`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES
			('m-recent', 'x1', 'A'), ('m-recent', 'x2', 'B'), ('m-autre', 'x3', 'C')`)
	var src port.MediaRecipientSource = MediaRecipientsRepo{}
	ctx := context.Background()

	ids, err := src.RecentMediaMatchIDs(ctx, socialPath, now.Add(-time.Hour))
	if err != nil || len(ids) != 1 || ids[0] != "m-recent" {
		t.Fatalf("matchs récents = %v, %v", ids, err)
	}
	xuids, err := src.MatchParticipantXUIDs(ctx, sharedPath, ids)
	if err != nil || len(xuids) != 2 {
		t.Fatalf("participants = %v, %v (attendu x1 et x2)", xuids, err)
	}
	if got, err := src.MatchParticipantXUIDs(ctx, sharedPath, nil); err != nil || got != nil {
		t.Errorf("sans match : %v, %v (attendu nil, sans ouvrir de base)", got, err)
	}
}
