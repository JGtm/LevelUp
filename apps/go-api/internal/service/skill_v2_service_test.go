//go:build cgo

package service

import (
	"context"
	"database/sql"
	"math"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// openSkillV2TestDB ouvre une DuckDB en mémoire et y crée les tables LUSR v2
// avec leurs vues _latest. Schema dupliqué (et non importé via migration.Run)
// pour rester un test unitaire pur — pas de dépendance au framework de migration.
func openSkillV2TestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb in-memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const ddl = `
		CREATE SEQUENCE player_skill_state_v2_seq START 1;
		CREATE TABLE player_skill_state_v2 (
			id              BIGINT DEFAULT nextval('player_skill_state_v2_seq') PRIMARY KEY,
			xuid            VARCHAR NOT NULL,
			playlist_group  VARCHAR NOT NULL,
			mu              DOUBLE  NOT NULL,
			sigma           DOUBLE  NOT NULL,
			experience      INTEGER NOT NULL DEFAULT 0,
			last_match_id   VARCHAR,
			last_match_at   TIMESTAMP,
			written_at      TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE VIEW player_skill_state_v2_latest AS
		SELECT s.*
		FROM player_skill_state_v2 s
		JOIN (
			SELECT xuid, playlist_group, MAX(written_at) AS max_written_at
			FROM player_skill_state_v2
			GROUP BY xuid, playlist_group
		) m ON s.xuid = m.xuid AND s.playlist_group = m.playlist_group
			AND s.written_at = m.max_written_at;
		CREATE SEQUENCE lusr_hyperparams_v2_seq START 1;
		CREATE TABLE lusr_hyperparams_v2 (
			id              BIGINT DEFAULT nextval('lusr_hyperparams_v2_seq') PRIMARY KEY,
			playlist_group  VARCHAR NOT NULL,
			name            VARCHAR NOT NULL,
			value           DOUBLE  NOT NULL,
			source          VARCHAR NOT NULL,
			written_at      TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE VIEW lusr_hyperparams_v2_latest AS
		SELECT h.*
		FROM lusr_hyperparams_v2 h
		JOIN (
			SELECT playlist_group, name, MAX(written_at) AS max_written_at
			FROM lusr_hyperparams_v2
			GROUP BY playlist_group, name
		) m ON h.playlist_group = m.playlist_group AND h.name = m.name
			AND h.written_at = m.max_written_at;
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("DDL: %v", err)
	}
	return db
}

func TestSkillV2Service_FirstMatch_CreatesStatesFromPriors(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	priors := skillv2.DefaultPriors()
	svc := NewSkillV2Service(repo, priors)

	ctx := context.Background()
	err := svc.UpdateAfterMatch(ctx, MatchInput{
		MatchID:       "m1",
		PlaylistGroup: "arena_slayer",
		StartTime:     time.Now(),
		TeamAXUIDs:    []string{"xA"},
		TeamBXUIDs:    []string{"xB"},
		OutcomeA:      skillv2.TeamWin,
	})
	if err != nil {
		t.Fatalf("UpdateAfterMatch: %v", err)
	}

	a, err := repo.LoadState(ctx, "xA", "arena_slayer")
	if err != nil || a == nil {
		t.Fatalf("LoadState xA: %v / %v", err, a)
	}
	if a.Mu <= priors.Mu0 {
		t.Errorf("winner μ (%v) should > priors.Mu0 (%v)", a.Mu, priors.Mu0)
	}
	if a.Experience != 1 {
		t.Errorf("winner experience = %d, want 1", a.Experience)
	}
	if a.LastMatchID == nil || *a.LastMatchID != "m1" {
		t.Errorf("winner last_match_id = %v, want m1", a.LastMatchID)
	}

	b, _ := repo.LoadState(ctx, "xB", "arena_slayer")
	if b == nil || b.Mu >= priors.Mu0 {
		t.Errorf("loser μ should < priors.Mu0, got %v", b)
	}
}

func TestSkillV2Service_SequentialMatches_StateEvolves(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	priors := skillv2.DefaultPriors()
	svc := NewSkillV2Service(repo, priors)

	ctx := context.Background()
	now := time.Now()
	// xA gagne 5 fois contre xB → mu_xA monte de manière marquée.
	for i := 0; i < 5; i++ {
		if err := svc.UpdateAfterMatch(ctx, MatchInput{
			MatchID:       "m" + string(rune('1'+i)),
			PlaylistGroup: "arena_slayer",
			StartTime:     now.Add(time.Duration(i) * time.Hour),
			TeamAXUIDs:    []string{"xA"},
			TeamBXUIDs:    []string{"xB"},
			OutcomeA:      skillv2.TeamWin,
		}); err != nil {
			t.Fatalf("match %d: %v", i, err)
		}
	}
	a, _ := repo.LoadState(ctx, "xA", "arena_slayer")
	if a.Experience != 5 {
		t.Errorf("xA experience after 5 matchs = %d, want 5", a.Experience)
	}
	// Après 5 wins, μ_A doit être nettement au-dessus du prior. Borne pas tight,
	// juste un sanity check : > 30 (prior 25).
	if a.Mu <= 30 {
		t.Errorf("xA μ after 5 wins = %v, expected > 30 (well above prior)", a.Mu)
	}
}

func TestSkillV2Service_PredictWin_BeforeAnyMatch_IsHalf(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	svc := NewSkillV2Service(repo, skillv2.DefaultPriors())

	p, err := svc.PredictWin(context.Background(), "arena_slayer",
		[]string{"a", "b"}, []string{"c", "d"})
	if err != nil {
		t.Fatalf("PredictWin: %v", err)
	}
	if math.Abs(p-0.5) > 1e-9 {
		t.Errorf("PredictWin sans état = %v, want 0.5", p)
	}
}

func TestSkillV2Service_EmptyTeam_Errors(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	svc := NewSkillV2Service(repo, skillv2.DefaultPriors())

	err := svc.UpdateAfterMatch(context.Background(), MatchInput{
		MatchID:       "m1",
		PlaylistGroup: "arena_slayer",
		TeamAXUIDs:    []string{},
		TeamBXUIDs:    []string{"xB"},
		OutcomeA:      skillv2.TeamWin,
	})
	if err == nil {
		t.Error("expected ErrEmptyTeam, got nil")
	}
}

func TestSkillV2Service_AppendOnly_KeepsHistory(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	svc := NewSkillV2Service(repo, skillv2.DefaultPriors())

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_ = svc.UpdateAfterMatch(ctx, MatchInput{
			MatchID:       "m" + string(rune('1'+i)),
			PlaylistGroup: "btb",
			StartTime:     time.Now().Add(time.Duration(i) * time.Minute),
			TeamAXUIDs:    []string{"x"},
			TeamBXUIDs:    []string{"y"},
			OutcomeA:      skillv2.TeamWin,
		})
	}
	// 3 matchs × 2 joueurs = 6 rows historiques.
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM player_skill_state_v2").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 6 {
		t.Errorf("got %d rows, expected 6 (append-only across 3 matches × 2 players)", n)
	}
	// La vue _latest doit n'en retourner que 2 (1 par joueur).
	if err := db.QueryRow("SELECT COUNT(*) FROM player_skill_state_v2_latest").Scan(&n); err != nil {
		t.Fatalf("count latest: %v", err)
	}
	if n != 2 {
		t.Errorf("latest view = %d rows, expected 2", n)
	}
}

func TestSkillV2Service_HyperparamRoundtrip(t *testing.T) {
	db := openSkillV2TestDB(t)
	repo := duckdb.NewSkillV2Repo(db)

	ctx := context.Background()
	type kv struct {
		name  string
		value float64
	}
	for _, h := range []kv{
		{"Mu0", 25.0}, {"Sigma0", 25.0 / 3.0}, {"Beta", 25.0 / 6.0}, {"Tau", 25.0 / 300.0},
	} {
		if err := repo.UpsertHyperparam(ctx, domain.SkillV2Hyperparam{
			PlaylistGroup: "arena_slayer", Name: h.name, Value: h.value, Source: "halo5_default",
		}); err != nil {
			t.Fatalf("UpsertHyperparam %s: %v", h.name, err)
		}
	}

	loaded, err := repo.LoadHyperparams(ctx, "arena_slayer")
	if err != nil {
		t.Fatalf("LoadHyperparams: %v", err)
	}
	if len(loaded) != 4 {
		t.Errorf("got %d params, want 4", len(loaded))
	}
	if math.Abs(loaded["Mu0"]-25.0) > 1e-12 {
		t.Errorf("Mu0 = %v, want 25.0", loaded["Mu0"])
	}
}
