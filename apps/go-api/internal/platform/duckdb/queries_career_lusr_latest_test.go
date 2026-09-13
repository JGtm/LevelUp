package duckdb

// queries_career_lusr_latest_test.go — garde C.3 du lot finitions LUSR (2026-09-13).
//
// Q8LUSRHistoryPlayer (graphe « Évolution LUSR / CSR » de la page Carrière) lisait la
// table BRUTE match_skill_rank. Sur une table append-only, le brut sert AUSSI les
// lignes des passes précédentes : une chaîne fausse écrite une fois y restait tracée
// À VIE, même après un replay correct — c'est ce qui maintenait la série « Arène »
// fantôme des 4 joueurs Infinite après la réparation d'août
// (.ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §6.2).
//
// Fixture sur FICHIER (t.TempDir), pas :memory: — on exerce la vue et les index posés
// par les MIGRATIONS RÉELLES, jamais une DDL recopiée.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// newPlayerFixtureDB ouvre une player DB de test sur fichier, schéma par migrations.
func newPlayerFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open player fixture: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	return db
}

func TestQ8LUSRHistoryPlayer_ServesOnlyLatestRowPerMatch(t *testing.T) {
	db := newPlayerFixtureDB(t)
	ctx := context.Background()

	start := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	// Même match, même rating_type : la ligne PÉRIMÉE porte la chaîne étrangère
	// h5_arena (passe fautive du 2026-06-26), la ligne RÉPARÉE porte arena_slayer et
	// un written_at postérieur. Un replay append-only n'efface jamais la première.
	for _, r := range []struct {
		group     string
		value     float64
		writtenAt time.Time
	}{
		{"h5_arena", 1100, time.Date(2026, 6, 26, 11, 23, 0, 0, time.UTC)},
		{"arena_slayer", 1350, time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
			VALUES ('m_replayed', 'LUSR', ?, ?, ?, ?)`,
			r.value, r.group, start, r.writtenAt); err != nil {
			t.Fatalf("insert %s: %v", r.group, err)
		}
	}
	// Un second match, une seule ligne : il doit rester servi (pas d'effet de bord).
	if _, err := db.ExecContext(ctx, `INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES ('m_simple', 'LUSR', 1200, 'btb', ?, ?)`,
		start.Add(time.Hour), time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert m_simple: %v", err)
	}

	rows, err := db.QueryContext(ctx, Q8LUSRHistoryPlayer)
	if err != nil {
		t.Fatalf("Q8LUSRHistoryPlayer: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := map[string]string{}
	for rows.Next() {
		var matchID, ratingType string
		var ratingValue float64
		var tierLabel, playlistGroup, tier sql.NullString
		var subTier sql.NullInt16
		if err := rows.Scan(&matchID, &ratingType, &ratingValue, &tierLabel,
			&playlistGroup, &tier, &subTier); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, dup := got[matchID]; dup {
			t.Fatalf("match %q servi DEUX fois : le graphe d'évolution reçoit une ligne périmée", matchID)
		}
		got[matchID] = playlistGroup.String
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("matchs servis = %d (%v), want 2", len(got), got)
	}
	if got["m_replayed"] != "arena_slayer" {
		t.Errorf("m_replayed → playlist_group = %q, want arena_slayer "+
			"(la ligne h5_arena du 2026-06-26 est périmée et ne doit plus être servie)", got["m_replayed"])
	}
	if got["m_simple"] != "btb" {
		t.Errorf("m_simple → playlist_group = %q, want btb", got["m_simple"])
	}
}
