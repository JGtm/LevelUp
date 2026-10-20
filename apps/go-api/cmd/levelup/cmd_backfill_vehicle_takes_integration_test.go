//go:build integration

package main

// cmd_backfill_vehicle_takes_integration_test.go — `levelup backfill-vehicle-takes` : ecriture,
// reprise (un match deja en base est saute), --force (il est re-ecrit), --dry-run (rien n'est
// ecrit), artefact absent. La base est migree par le chemin reel (`migrerSchemaPartage`).

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	titlePkg "levelup/go-api/internal/domain/title"
	halo "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/weapons"
)

func vtTagEngin(t *testing.T) uint32 {
	t.Helper()
	classes := weapons.ClassesByKey()
	for _, tag := range killicon.ResolvedTags() {
		if ic, _ := killicon.Lookup(tag); ic.WeaponKey != "" && domain.IsEngineFragClass(classes[ic.WeaponKey]) {
			return tag
		}
	}
	t.Fatal("aucune source d'engin dans la table killicon")
	return 0
}

func vtBase(t *testing.T) *sql.DB {
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

func vtArtefact(t *testing.T, racine, matchID string, schema int) {
	t.Helper()
	o, t0 := int64(5000), 0
	doc := replay.ReplayDocument{
		SchemaVersion: schema, MatchID: matchID, FrameIntervalMS: 100, OriginMs: &o,
		Coverage: &replay.Coverage{Vehicles: &replay.VehicleCoverage{Scanned: true}},
		Roster:   []replay.RosterEntry{{XUID: "a1", Team: &t0}},
		Vehicles: []replay.VehicleTrack{{Slot: 1, Gen: 1, Family: "warthog", Rides: []replay.VehicleRide{
			{XUID: "a1", T0: 10, T1: 20, Src: "film"}}}},
	}
	path := titlePkg.NewPathResolver(racine).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	blob, _ := json.Marshal(doc)
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
}

func vtPasses(t *testing.T, db *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT decode_pass) FROM match_vehicle_takes WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBackfillVehicleTakes_EcritReprendForceEtDryRun(t *testing.T) {
	db, racine, ctx := vtBase(t), t.TempDir(), context.Background()
	pr := titlePkg.NewPathResolver(racine)
	vtArtefact(t, racine, "b1b1b1b1-0000-4000-8000-000000000001", replay.SchemaVersion)
	vtArtefact(t, racine, "b2b2b2b2-0000-4000-8000-000000000002", 61) // sans occupation lue : « non mesure »
	if _, err := db.Exec(`INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms, victim_gamertag, feed_killer_xuid,
		 feed_present, assist_known, source_tag, read_path, read_origin)
		VALUES ('b1b1b1b1-0000-4000-8000-000000000001', 'p', 'r', TRUE, 6500, 'v', 'a1', TRUE, FALSE, ?, ?, ?)`,
		vtTagEngin(t), killscope.ReadPathFilmWalk, killscope.OriginCreditOnly); err != nil {
		t.Fatalf("insert mort: %v", err)
	}
	cl := halo.NewKillSourceRegistry()
	cands := []string{"b1b1b1b1-0000-4000-8000-000000000001", "b2b2b2b2-0000-4000-8000-000000000002", "b3b3b3b3-0000-4000-8000-000000000003"} // b3b3b3b3-0000-4000-8000-000000000003 : aucun artefact
	o := vehicleTakesOptions{titleSlug: titlePkg.DefaultSlug}

	// --dry-run : rien n'est ecrit.
	od := o
	od.dryRun = true
	if b := projeterCorpusVehicules(ctx, db, pr, od, cl, cands, map[string]bool{}); b.ecrits != 0 || b.sansArtefact != 1 {
		t.Fatalf("dry-run : %+v", b)
	}
	if vtPasses(t, db, "b1b1b1b1-0000-4000-8000-000000000001")+vtPasses(t, db, "b2b2b2b2-0000-4000-8000-000000000002") != 0 {
		t.Fatal("--dry-run a ecrit en base")
	}

	// Premiere passe reelle.
	b := projeterCorpusVehicules(ctx, db, pr, o, cl, cands, map[string]bool{})
	if b.ecrits != 2 || b.nonMesures != 1 || b.sansArtefact != 1 || b.echecs != 0 || b.fragsTotal != 1 || b.fragsNonApp != 0 {
		t.Fatalf("premiere passe : %+v", b)
	}
	var frags int
	if err := db.QueryRow(`SELECT frags FROM match_vehicle_takes_latest WHERE match_id='b1b1b1b1-0000-4000-8000-000000000001' AND row_kind='take'`).Scan(&frags); err != nil || frags != 1 {
		t.Errorf("frag apparie en base : %d (err=%v)", frags, err)
	}

	// Reprise : les matchs deja en base sont sautes, aucune passe de plus.
	deja, err := matchsDejaProjetesVehicules(ctx, db)
	if err != nil || !deja["b1b1b1b1-0000-4000-8000-000000000001"] || !deja["b2b2b2b2-0000-4000-8000-000000000002"] {
		t.Fatalf("cle de reprise : %v (err=%v)", deja, err)
	}
	b = projeterCorpusVehicules(ctx, db, pr, o, cl, cands, deja)
	if b.ecrits != 0 || b.dejaEnBase != 2 || vtPasses(t, db, "b1b1b1b1-0000-4000-8000-000000000001") != 1 {
		t.Fatalf("reprise : %+v, passes b1b1b1b1-0000-4000-8000-000000000001=%d", b, vtPasses(t, db, "b1b1b1b1-0000-4000-8000-000000000001"))
	}

	// --force : re-ecrit MEME ce qui est deja en base (la cle de reprise est fournie, et ignoree).
	of := o
	of.force = true
	b = projeterCorpusVehicules(ctx, db, pr, of, cl, cands, deja)
	if b.ecrits != 2 || vtPasses(t, db, "b1b1b1b1-0000-4000-8000-000000000001") != 2 {
		t.Fatalf("force : %+v, passes b1b1b1b1-0000-4000-8000-000000000001=%d", b, vtPasses(t, db, "b1b1b1b1-0000-4000-8000-000000000001"))
	}
}
