// Package service — relations_segmentation_integration_test.go : test bout-en-bout
// de la segmentation serveur du hub Relations (Phase 2), avec un DuckDB :memory:
// servant à la fois de player DB et de shared (LegacySharedReader). Couvre la
// résolution cross-DB du scope solo/escouade (is_with_friends en player DB) et
// sa propagation jusqu'à l'agrégation shared (Q28 scopée).
//
// Build tag `integration` — exclu du go test ./... par défaut. Lancer avec :
//
//	go test -tags=integration ./internal/service/ -run TestRelationsSegmentation
//
//go:build integration

package service

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// seedRelationsSegmentation peuple un :memory: avec le schéma minimal commun au
// FiltersRepo (v_match_full + player_match_enrichment_latest) et au CareerRepo
// (match_participants + match_registry + killer_victim_pairs + v_gamertag_lookup).
//
// Dataset (joueur = xuidMe) :
//   - m1, m2 : ESCOUADE (is_with_friends=TRUE), me + Buddy alliés (WIN)
//   - m3, m4 : SOLO     (is_with_friends=FALSE), me vs Rival ennemis (LOSS)
func seedRelationsSegmentation(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time_utc TIMESTAMPTZ, start_time TIMESTAMP,
			map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR, pair_id VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN, is_ranked BOOLEAN,
			map_id VARCHAR, playlist_id VARCHAR,
			game_variant_id VARCHAR, game_variant_name VARCHAR)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER, kda DOUBLE)`,
		`CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM xuid_aliases`,
		// v_match_full : la query FiltersRepo lit r.* depuis cette vue.
		`CREATE VIEW v_match_full AS SELECT * FROM match_registry`,
		// player_match_enrichment_latest : is_with_friends + colonnes lues par
		// LoadPlayerMatchEnrichments (session/perf/dominance/engagement).
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR, session_id VARCHAR, session_label VARCHAR,
			is_with_friends BOOLEAN, is_excluded BOOLEAN,
			performance_score DOUBLE, dominance_flag INTEGER,
			had_bot_teammate BOOLEAN,
			engagement_score_brut DOUBLE,
			engagement_pace_player DOUBLE, engagement_pace_lobby DOUBLE)`,
		`CREATE VIEW player_match_enrichment_latest AS SELECT * FROM player_match_enrichment`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("seed DDL: %v\nSQL: %s", err, ddl)
		}
	}

	for _, ins := range []string{
		`INSERT INTO match_registry (match_id, start_time_utc, playlist_name, pair_name, is_firefight, is_ranked) VALUES
			('m1', TIMESTAMPTZ '2026-01-10 14:00:00+00', 'Quick Play', 'Slayer', FALSE, FALSE),
			('m2', TIMESTAMPTZ '2026-02-10 14:00:00+00', 'Quick Play', 'Slayer', FALSE, FALSE),
			('m3', TIMESTAMPTZ '2026-03-10 14:00:00+00', 'Quick Play', 'Slayer', FALSE, FALSE),
			('m4', TIMESTAMPTZ '2026-04-10 14:00:00+00', 'Quick Play', 'Slayer', FALSE, FALSE)`,
		`INSERT INTO match_participants VALUES
			('m1','xuidMe',0,2,1.5), ('m1','xuidBuddy',0,2,2.0),
			('m2','xuidMe',0,2,1.5), ('m2','xuidBuddy',0,2,3.0),
			('m3','xuidMe',0,3,0.8), ('m3','xuidRival',1,2,2.5),
			('m4','xuidMe',0,3,0.8), ('m4','xuidRival',1,2,1.5)`,
		`INSERT INTO killer_victim_pairs VALUES
			('m3','xuidMe','xuidRival',2),
			('m3','xuidRival','xuidMe',6),
			('m4','xuidRival','xuidMe',4)`,
		`INSERT INTO xuid_aliases VALUES
			('xuidMe','MePlayer'), ('xuidBuddy','BuddyPlayer'), ('xuidRival','RivalPlayer')`,
		// m1/m2 = escouade, m3/m4 = solo.
		`INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES
			('m1', TRUE), ('m2', TRUE), ('m3', FALSE), ('m4', FALSE)`,
	} {
		if _, err := db.Exec(ctx, ins); err != nil {
			t.Fatalf("seed INSERT: %v\nSQL: %s", err, ins)
		}
	}
}

func newSegmentationPlayerDB(t *testing.T) *duckdb.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// buildSegmentationService câble RelationsService.WithFilters(real FiltersService)
// sur un PlayerDB où Player == Shared == db (SharedReader legacy).
func buildSegmentationService(t *testing.T, db *duckdb.DB) *RelationsService {
	t.Helper()
	pdb := &duckdb.PlayerDB{
		Player:       db,
		Shared:       db,
		SharedReader: duckdb.LegacySharedReader(db),
		XUID:         "xuidMe",
		Gamertag:     "MePlayer",
	}
	repo := duckdb.NewCareerRepo(pdb)
	filtersSvc := NewFiltersService(duckdb.NewFiltersRepo(pdb))
	return NewRelationsService(repo).WithFilters(filtersSvc)
}

// TestRelationsSegmentation_SoloVsSquad_CrossDB : la vue solo ne garde que les
// matchs solo (m3/m4 → Rival), la vue escouade que les matchs en groupe (m1/m2
// → Buddy). Valide le filtre is_with_friends résolu en player DB puis propagé à
// l'agrégation shared.
func TestRelationsSegmentation_SoloVsSquad_CrossDB(t *testing.T) {
	db := newSegmentationPlayerDB(t)
	seedRelationsSegmentation(t, db)
	svc := buildSegmentationService(t, db)
	ctx := context.Background()

	// Vue ESCOUADE : seul Buddy (m1/m2), Rival hors périmètre.
	squad, err := svc.GetRelationsPage(ctx, domain.FilterContextInput{MatchContext: domain.MatchContextSquad})
	if err != nil {
		t.Fatalf("squad page: %v", err)
	}
	if got := relationGamertags(squad); len(got) != 1 || got[0] != "BuddyPlayer" {
		t.Fatalf("squad relations=%v want [BuddyPlayer]", got)
	}

	// Vue SOLO : seul Rival (m3/m4), Buddy hors périmètre.
	solo, err := svc.GetRelationsPage(ctx, domain.FilterContextInput{MatchContext: domain.MatchContextSolo})
	if err != nil {
		t.Fatalf("solo page: %v", err)
	}
	if got := relationGamertags(solo); len(got) != 1 || got[0] != "RivalPlayer" {
		t.Fatalf("solo relations=%v want [RivalPlayer]", got)
	}

	// Vue TOUT (input trivial) : les deux relations émergent (scope nil).
	all, err := svc.GetRelationsPage(ctx, domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("all page: %v", err)
	}
	if got := relationGamertags(all); len(got) != 2 {
		t.Fatalf("all relations=%v want 2 (Buddy + Rival)", got)
	}
}

// TestRelationsSegmentation_PlaylistFilter_CrossDB : un filtre playlist absent du
// dataset vide le périmètre → aucune relation (page vide), sans crash.
func TestRelationsSegmentation_PlaylistFilter_CrossDB(t *testing.T) {
	db := newSegmentationPlayerDB(t)
	seedRelationsSegmentation(t, db)
	svc := buildSegmentationService(t, db)

	in := domain.FilterContextInput{Cascade: domain.CascadeFilter{Playlists: []string{"Ranked Arena"}}}
	page, err := svc.GetRelationsPage(context.Background(), in)
	if err != nil {
		t.Fatalf("playlist page: %v", err)
	}
	if len(page.Relations) != 0 {
		t.Fatalf("unknown playlist must yield 0 relations, got %d", len(page.Relations))
	}

	// Playlist présente → les 2 relations reviennent.
	in2 := domain.FilterContextInput{Cascade: domain.CascadeFilter{Playlists: []string{"Quick Play"}}}
	page2, err := svc.GetRelationsPage(context.Background(), in2)
	if err != nil {
		t.Fatalf("playlist page2: %v", err)
	}
	if len(page2.Relations) != 2 {
		t.Fatalf("Quick Play must yield 2 relations, got %d", len(page2.Relations))
	}
}

func relationGamertags(p domain.RelationsPageResponse) []string {
	out := make([]string, 0, len(p.Relations))
	for _, r := range p.Relations {
		out = append(out, r.Gamertag)
	}
	return out
}
