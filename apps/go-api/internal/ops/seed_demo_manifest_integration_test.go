//go:build integration

// Package ops — seed_demo_manifest_integration_test.go : le manifeste figé rend le
// corpus démo reproductible malgré la dérive des données source.
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// TestSeedDemo_FrozenCorpusIsByteStable : avec un manifeste figé sur {m2, m3}, le
// corpus reste {m2, m3} — même quand m1 est plus récent ET même après insertion d'un
// match m0 ENCORE plus récent dans la source. La sélection dynamique aurait dérivé.
func TestSeedDemo_FrozenCorpusIsByteStable(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	const sourceXUID = "1111111111111111"
	repoRoot := filepath.Join(tmpDir, "repo")

	// Manifeste figé : corpus = {m2, m3} (PAS m1, le plus récent).
	manPath := titlePkg.NewPathResolver(repoRoot).DemoManifestPath("JGtm", titlePkg.DefaultSlug)
	man := &DemoManifest{
		Version:   demoManifestVersion,
		TitleSlug: titlePkg.DefaultSlug,
		Corpus:    DemoManifestCorpus{SoloMatchIDs: []string{"m2", "m3"}},
	}
	if err := writeDemoManifest(manPath, man); err != nil {
		t.Fatal(err)
	}

	opts := SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     sourceXUID,
		OutDir:         filepath.Join(tmpDir, "out"),
		MaxMatches:     2, // ignoré quand le manifeste pilote le corpus
		SourceLabel:    "JGtm",
		ServiceTag:     "SPTA",
		IncludeMedia:   false,
		RepoRoot:       repoRoot,
	}
	res, err := SeedDemo(context.Background(), opts)
	if err != nil {
		t.Fatalf("SeedDemo (run A): %v", err)
	}
	if !res.Frozen {
		t.Error("res.Frozen attendu true (manifeste chargé)")
	}
	assertSameSet(t, "run A", res.MatchIDs, []string{"m2", "m3"})
	assertNoRealGamertagLeak(t, filepath.Join(opts.OutDir, "warehouse", "shared_matches_v2.duckdb"))

	// ── Drift : insérer un match m0 ENCORE plus récent que m1 pour le source.
	sharedDB, err := sql.Open("duckdb", srcShared)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, sharedDB, `INSERT INTO match_registry (match_id, start_time, map_name, playlist_name, pair_name) VALUES ('m0', TIMESTAMP '2026-05-23 18:00:00', 'Recharge', 'Open Crossplay', 'Open:Slayer')`)
	mustExec(t, sharedDB, `INSERT INTO match_participants (match_id, xuid, gamertag, kills, deaths) VALUES ('m0', '`+sourceXUID+`', 'JGtm', 5, 5)`)
	_ = sharedDB.Close()

	// ── Re-seed avec le MÊME manifeste → corpus INCHANGÉ (m0 absent).
	opts.OutDir = filepath.Join(tmpDir, "out2")
	res2, err := SeedDemo(context.Background(), opts)
	if err != nil {
		t.Fatalf("SeedDemo (run B): %v", err)
	}
	assertSameSet(t, "run B (après drift)", res2.MatchIDs, []string{"m2", "m3"})
}

// TestSeedDemo_NoManifest_StaysDynamic : sans manifeste, le comportement historique
// (sélection « N récents ») est préservé — res.Frozen reste false.
func TestSeedDemo_NoManifest_StaysDynamic(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	opts := SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     "1111111111111111",
		OutDir:         filepath.Join(tmpDir, "out"),
		MaxMatches:     2,
		SourceLabel:    "JGtm",
		RepoRoot:       filepath.Join(tmpDir, "repo_without_manifest"),
	}
	res, err := SeedDemo(context.Background(), opts)
	if err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}
	if res.Frozen {
		t.Error("res.Frozen attendu false sans manifeste")
	}
	assertSameSet(t, "dynamique", res.MatchIDs, []string{"m1", "m2"}) // 2 plus récents
}

func assertSameSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	gs := append([]string(nil), got...)
	ws := append([]string(nil), want...)
	sort.Strings(gs)
	sort.Strings(ws)
	if len(gs) != len(ws) {
		t.Fatalf("%s: corpus = %v, want %v", label, got, want)
	}
	for i := range gs {
		if gs[i] != ws[i] {
			t.Fatalf("%s: corpus = %v, want %v", label, got, want)
		}
	}
}

// assertNoRealGamertagLeak vérifie qu'aucun vrai gamertag (JGtm/Other) ne subsiste
// dans le shared démo après anonymisation.
func assertNoRealGamertagLeak(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := sql.Open("duckdb", sharedPath+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, table := range []string{"xuid_aliases", "match_participants"} {
		rows, err := db.Query(`SELECT DISTINCT gamertag FROM ` + table + ` WHERE gamertag IS NOT NULL`)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		for rows.Next() {
			var gt string
			if err := rows.Scan(&gt); err != nil {
				t.Fatal(err)
			}
			if gt == "JGtm" || gt == "Other" {
				rows.Close()
				t.Fatalf("fuite gamertag réel %q dans %s (anonymisation incomplète)", gt, table)
			}
		}
		rows.Close()
	}
}
