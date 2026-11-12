//go:build integration

package main

// cmd_backfill_lead_share_integration_test.go — `levelup backfill-lead-share` : écriture, reprise
// (un match déjà en base est sauté), --force (il est ré-écrit), --dry-run (rien n'est écrit), artefact
// absent, mode à peu de points. La base est migrée par le chemin réel (`migrerSchemaPartage`).

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/replayartifact"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

const (
	lsMatchA = "c1c1c1c1-0000-4000-8000-000000000001"
	lsMatchB = "c2c2c2c2-0000-4000-8000-000000000002"
	lsAucun  = "c3c3c3c3-0000-4000-8000-000000000003"
)

func lsBase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrerSchemaPartage(db, titlePkg.DefaultSlug); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return db
}

func lsRegistre(t *testing.T, db *sql.DB, id, pair string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, backfill_completed, map_name, pair_name, duration_seconds)
		VALUES (?, TIMESTAMP '2026-01-01 10:00:00', TIMESTAMP '2026-01-01 10:00:00', 0, 'Recharge', ?, 600)`, id, pair); err != nil {
		t.Fatalf("insert registre: %v", err)
	}
}

func lsArtefact(t *testing.T, racine, id string) {
	t.Helper()
	o, t0f, c0, c1 := int64(5000), int64(20000), 0, 1
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: id, FrameIntervalMS: 100, FrameCount: 10000,
		OriginMs: &o, T0FilmMs: &t0f,
		Roster: []replay.RosterEntry{{XUID: "a", Team: &c0}, {XUID: "b", Team: &c1}},
		Coverage: &replay.Coverage{Score: &replay.ScoreCoverage{
			TeamIdentity: "a", ModeSupported: true, Rounds: 1}},
		ScoreTimeline: &replay.ScoreTimeline{Teams: []replay.TeamScore{
			{TeamID: &c0, Total: []replay.ScoreTick{{T: 200, V: 1}, {T: 600, V: 3}}},
			{TeamID: &c1, Total: []replay.ScoreTick{{T: 400, V: 2}, {T: 800, V: 3}}},
		}},
	}
	path := titlePkg.NewPathResolver(racine).ReplayArtifactPath(titlePkg.DefaultSlug, id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	blob, _ := json.Marshal(doc)
	if _, _, err := replayartifact.Write(context.Background(), path, blob); err != nil {
		t.Fatal(err)
	}
}

func lsPasses(t *testing.T, db *sql.DB, id string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT decode_pass) FROM match_lead_share WHERE match_id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBackfillLeadShare_EcritReprendForceEtDryRun(t *testing.T) {
	db, racine, ctx := lsBase(t), t.TempDir(), context.Background()
	pr := titlePkg.NewPathResolver(racine)
	lsRegistre(t, db, lsMatchA, "Team Slayer:Arena on Recharge")
	lsRegistre(t, db, lsMatchB, "CTF:Arena on Recharge") // mode à peu de points : non mesuré
	lsRegistre(t, db, lsAucun, "Team Slayer:Arena on Recharge")
	lsArtefact(t, racine, lsMatchA)
	lsArtefact(t, racine, lsMatchB)
	racine0, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	reg, err := mappings.LoadRegulationFromFile(filepath.Join(racine0, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation: %v", err)
	}
	cands := []string{lsMatchA, lsMatchB, lsAucun}
	o := leadShareOptions{titleSlug: titlePkg.DefaultSlug}
	run := func(o leadShareOptions, deja map[string]bool) bilanLeadShareBackfill {
		return passeMainmise{db: db, pr: pr, o: o, reg: reg}.projeterCorpus(ctx, cands, deja)
	}

	// --dry-run : rien n'est écrit.
	od := o
	od.dryRun = true
	if b := run(od, map[string]bool{}); b.ecrits != 0 || b.sansArtefact != 1 || b.mesures != 1 || b.raisons["mode_events"] != 1 {
		t.Fatalf("dry-run : %+v", b)
	}
	if lsPasses(t, db, lsMatchA)+lsPasses(t, db, lsMatchB) != 0 {
		t.Fatal("--dry-run a écrit en base")
	}

	// Première passe réelle : un match mesuré, un non mesuré (écrit avec sa raison), un sans artefact.
	b := run(o, map[string]bool{})
	if b.ecrits != 2 || b.mesures != 1 || b.nonMesures() != 1 || b.sansArtefact != 1 || b.echecs != 0 {
		t.Fatalf("première passe : %+v", b)
	}
	var measured bool
	var reason string
	if err := db.QueryRow(`SELECT measured, unmeasured_reason FROM match_lead_share_latest
		WHERE match_id = ? AND row_kind = 'match'`, lsMatchB).Scan(&measured, &reason); err != nil || measured || reason != "mode_events" {
		t.Errorf("match à peu de points : mesuré=%v raison=%q (err=%v)", measured, reason, err)
	}

	// Reprise : les matchs déjà en base sont sautés.
	deja, err := matchsDejaProjetesMainmise(ctx, db)
	if err != nil || !deja[lsMatchA] || !deja[lsMatchB] {
		t.Fatalf("clé de reprise : %v (err=%v)", deja, err)
	}
	if b = run(o, deja); b.ecrits != 0 || b.dejaEnBase != 2 || lsPasses(t, db, lsMatchA) != 1 {
		t.Fatalf("reprise : %+v, passes=%d", b, lsPasses(t, db, lsMatchA))
	}

	// --force : ré-écrit MÊME ce qui est déjà en base.
	of := o
	of.force = true
	if b = run(of, deja); b.ecrits != 2 || lsPasses(t, db, lsMatchA) != 2 {
		t.Fatalf("force : %+v, passes=%d", b, lsPasses(t, db, lsMatchA))
	}

	// --limit borne les matchs projetés.
	ol := of
	ol.limit = 1
	if b = run(ol, deja); b.mesures+b.nonMesures() != 1 {
		t.Fatalf("limit : %+v", b)
	}
}
