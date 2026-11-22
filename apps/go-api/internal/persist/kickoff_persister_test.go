//go:build integration

// Package persist — kickoff_persister_test.go : le persisteur du coup d'envoi écrit registre et
// temps de jeu d'un match dans une transaction, sur le schéma migré.
package persist

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func semerMatchCoupDEnvoi(t *testing.T, db *sql.DB, start time.Time) {
	t.Helper()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO match_registry (match_id, start_time, start_time_utc) VALUES ('m1', ?, ?)`, []any{start, start}},
		{`INSERT INTO match_participants (match_id, xuid, time_played_seconds) VALUES ('m1', '1', 600), ('m1', '2', 300)`, nil},
	} {
		if _, err := db.ExecContext(context.Background(), q.sql, q.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func lireCoupDEnvoi(t *testing.T, db *sql.DB) (sql.NullTime, sql.NullString, map[string]int) {
	t.Helper()
	var rst sql.NullTime
	var src sql.NullString
	if err := db.QueryRowContext(context.Background(),
		`SELECT real_start_time, real_start_source FROM match_registry WHERE match_id = 'm1'`).Scan(&rst, &src); err != nil {
		t.Fatalf("registre: %v", err)
	}
	rows, err := db.QueryContext(context.Background(),
		`SELECT xuid, time_played_seconds FROM match_participants WHERE match_id = 'm1'`)
	if err != nil {
		t.Fatalf("participants: %v", err)
	}
	defer func() { _ = rows.Close() }()
	tp := map[string]int{}
	for rows.Next() {
		var x string
		var v int
		if err := rows.Scan(&x, &v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tp[x] = v
	}
	return rst, src, tp
}

func TestKickoffPersister_EcritRegistreEtTempsDeJeu(t *testing.T) {
	db := openSharedTestDB(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semerMatchCoupDEnvoi(t, db, start)
	rs := start.Add(26304 * time.Millisecond)
	p := NewKickoffPersister(db)
	if err := p.WriteMatch(context.Background(), MatchKickoffWrite{
		MatchID:    "m1",
		Kickoff:    &KickoffValue{RealStart: &rs, Source: "film"},
		TimePlayed: []PlayerTimePlayed{{XUID: "1", Seconds: 573}},
	}); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}
	rst, src, tp := lireCoupDEnvoi(t, db)
	if !rst.Valid || !rst.Time.UTC().Equal(rs) || src.String != "film" {
		t.Fatalf("registre = (%v, %v)", rst, src)
	}
	if tp["1"] != 573 || tp["2"] != 300 {
		t.Fatalf("temps de jeu = %v, attendu 1=573 (ecrit) et 2=300 (inchange)", tp)
	}

	// Coup d'envoi devenu inconnu : valeur et source remises à NULL.
	if err := p.WriteMatch(context.Background(), MatchKickoffWrite{MatchID: "m1", Kickoff: &KickoffValue{}}); err != nil {
		t.Fatalf("WriteMatch inconnu: %v", err)
	}
	rst, src, _ = lireCoupDEnvoi(t, db)
	if rst.Valid || src.Valid {
		t.Fatalf("inconnu : registre = (%v, %v), attendu NULL, NULL", rst, src)
	}
}

func TestKickoffPersister_EcritureVideEtRevision(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewKickoffPersister(db)
	if err := p.WriteMatch(context.Background(), MatchKickoffWrite{MatchID: "absent"}); err != nil {
		t.Fatalf("ecriture vide : %v", err)
	}
	if err := p.WriteMatch(context.Background(), MatchKickoffWrite{}); err == nil {
		t.Fatal("matchID vide accepte")
	}
	for _, v := range []string{"rev-a", "rev-b"} {
		if err := p.StoreRevision(context.Background(), "kickoff_revision", v); err != nil {
			t.Fatalf("StoreRevision %s: %v", v, err)
		}
	}
	var got string
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT MAX(value), COUNT(*) FROM sync_meta WHERE key = 'kickoff_revision'`).Scan(&got, &n); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if got != "rev-b" || n != 1 {
		t.Fatalf("sync_meta = %q (%d ligne(s)), attendu rev-b en une ligne", got, n)
	}
}
