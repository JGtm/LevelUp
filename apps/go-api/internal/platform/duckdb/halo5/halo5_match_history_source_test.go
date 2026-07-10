//go:build integration

// halo5_match_history_source_test.go — test in-memory (DuckDB :memory:) de la
// lecture de l'historique h5 depuis le shared LOCAL. NE TOUCHE JAMAIS la vraie DB
// h5 (mono-process lock) : tout est créé/peuplé in-memory ici.
//
// Lancer : go test -tags=integration ./internal/platform/duckdb/ -run Halo5MatchHistory

package halo5

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/games/canonical"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedH5SharedHistory crée les 2 tables minimales (match_registry + match_participants)
// et insère 3 matchs h5 pour le gamertag "JGtm" + 1 match d'un autre joueur (bruit).
func seedH5SharedHistory(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			map_id VARCHAR, map_name VARCHAR, map_name_fr VARCHAR,
			playlist_id VARCHAR, playlist_name VARCHAR, playlist_name_fr VARCHAR,
			game_variant_id VARCHAR, game_variant_name VARCHAR,
			pair_id VARCHAR, pair_name VARCHAR, pair_name_fr VARCHAR,
			is_ranked BOOLEAN, is_firefight BOOLEAN,
			duration_seconds INTEGER, team_0_score INTEGER, team_1_score INTEGER
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			team_id INTEGER, outcome INTEGER,
			PRIMARY KEY (match_id, xuid)
		)`,
	}
	for _, q := range ddl {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("ddl: %v\n%s", err, q)
		}
	}
	// 3 matchs JGtm (m3 le plus récent), ordre DESC attendu = m3, m2, m1.
	reg := []string{
		`INSERT INTO match_registry VALUES ('m1','2023-05-01 20:00:00','2023-05-01 20:00:00+00',
			'map-truth','Truth',NULL,'hop-1','Ranked Arena',NULL,'gv-slayer','Slayer',
			'pair-1','Slayer',NULL,TRUE,FALSE,540,50,38)`,
		`INSERT INTO match_registry VALUES ('m2','2023-05-02 21:00:00','2023-05-02 21:00:00+00',
			'map-plaza','Plaza',NULL,'hop-2','Social',NULL,'gv-ctf','CTF',
			NULL,NULL,NULL,FALSE,FALSE,600,2,3)`,
		`INSERT INTO match_registry VALUES ('m3','2023-05-03 22:00:00','2023-05-03 22:00:00+00',
			'map-eden','Eden',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,FALSE,FALSE,NULL,NULL,NULL)`,
		`INSERT INTO match_registry VALUES ('mx','2023-05-04 23:00:00','2023-05-04 23:00:00+00',
			'map-x','X',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,FALSE,FALSE,300,1,2)`,
	}
	part := []string{
		// JGtm : xuid résolu réel stocké, gamertag aussi (filtre se fait sur gamertag).
		`INSERT INTO match_participants VALUES ('m1','xuid-jgtm','JGtm',0,2)`,
		`INSERT INTO match_participants VALUES ('m2','xuid-jgtm','JGtm',1,3)`,
		`INSERT INTO match_participants VALUES ('m3','xuid-jgtm','JGtm',0,1)`,
		// Autre joueur sur mx : ne doit jamais ressortir pour JGtm.
		`INSERT INTO match_participants VALUES ('mx','xuid-other','OtherGuy',0,2)`,
	}
	for _, q := range append(reg, part...) {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
}

func newH5HistorySource(t *testing.T, gamertag string) *Halo5MatchHistorySource {
	t.Helper()
	mem := openMemSQL(t)
	seedH5SharedHistory(t, mem)
	return NewHalo5MatchHistorySource(&memSharedReader{mem}, gamertag)
}

// TestHalo5MatchHistory_LatestN : matchIDs nil → tous les matchs du joueur, DESC.
func TestHalo5MatchHistory_LatestN(t *testing.T) {
	src := newH5HistorySource(t, "JGtm")
	got, err := src.GetMatchSummaries(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMatchSummaries: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (mx d'un autre joueur exclu)", len(got))
	}
	// Ordre DESC par start_time : m3, m2, m1.
	wantOrder := []string{"m3", "m2", "m1"}
	for i, id := range wantOrder {
		if got[i].MatchID != id {
			t.Errorf("ordre #%d = %q, want %q", i, got[i].MatchID, id)
		}
	}
	// Mapping m1 : ranked + outcome win + map/playlist/variant/pair + team scores.
	var m1 *canonical.MatchSummary
	for i := range got {
		if got[i].MatchID == "m1" {
			m1 = &got[i]
		}
	}
	if m1 == nil {
		t.Fatal("m1 absent")
	}
	if m1.IsRanked == nil || !*m1.IsRanked {
		t.Errorf("m1 IsRanked = %v, want &true", m1.IsRanked)
	}
	if m1.MatchType != canonical.MatchTypeRanked {
		t.Errorf("m1 MatchType = %q, want ranked", m1.MatchType)
	}
	if m1.Outcome != canonical.OutcomeWin {
		t.Errorf("m1 Outcome = %q, want win", m1.Outcome)
	}
	if m1.Map == nil || m1.Map.ID != "map-truth" || m1.Map.DefaultLabel != "Truth" {
		t.Errorf("m1 Map KO: %+v", m1.Map)
	}
	if m1.Playlist == nil || m1.Playlist.ID != "hop-1" {
		t.Errorf("m1 Playlist KO: %+v", m1.Playlist)
	}
	if m1.GameVariant == nil || m1.GameVariant.ID != "gv-slayer" {
		t.Errorf("m1 GameVariant KO: %+v", m1.GameVariant)
	}
	if m1.DurationSeconds == nil || *m1.DurationSeconds != 540 {
		t.Errorf("m1 DurationSeconds = %v, want 540", m1.DurationSeconds)
	}
	if len(m1.Teams) != 2 {
		t.Errorf("m1 Teams = %d, want 2", len(m1.Teams))
	}
	if m1.StartedAtUTC.IsZero() {
		t.Error("m1 StartedAtUTC ne doit pas être zéro")
	}
	// m3 : pas de scores → Teams vide ; pas de durée → nil ; social.
	var m3 *canonical.MatchSummary
	for i := range got {
		if got[i].MatchID == "m3" {
			m3 = &got[i]
		}
	}
	if m3 == nil {
		t.Fatal("m3 absent")
	}
	if len(m3.Teams) != 0 {
		t.Errorf("m3 Teams = %d, want 0 (scores NULL)", len(m3.Teams))
	}
	if m3.DurationSeconds != nil {
		t.Errorf("m3 DurationSeconds = %v, want nil", m3.DurationSeconds)
	}
	if m3.Outcome != canonical.OutcomeTie {
		t.Errorf("m3 Outcome = %q, want tie", m3.Outcome)
	}
}

// TestHalo5MatchHistory_FilterByIDs_OrderPreserved : matchIDs filtre + préserve
// l'ordre d'entrée (≠ ordre DESC), un ID inconnu est omis.
func TestHalo5MatchHistory_FilterByIDs_OrderPreserved(t *testing.T) {
	src := newH5HistorySource(t, "JGtm")
	got, err := src.GetMatchSummaries(context.Background(), []string{"m2", "m1", "absent", "m3"})
	if err != nil {
		t.Fatalf("GetMatchSummaries: %v", err)
	}
	// "absent" omis → 3 résultats, ordre d'entrée préservé : m2, m1, m3.
	want := []string{"m2", "m1", "m3"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].MatchID != want[i] {
			t.Errorf("ordre #%d = %q, want %q", i, got[i].MatchID, want[i])
		}
	}
}

// TestHalo5MatchHistory_CaseInsensitiveGamertag : le filtre gamertag est insensible
// à la casse (h5 gamertag-keyé).
func TestHalo5MatchHistory_CaseInsensitiveGamertag(t *testing.T) {
	src := newH5HistorySource(t, "jgtm")
	got, err := src.GetMatchSummaries(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMatchSummaries: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3 (casse ignorée)", len(got))
	}
}

// TestHalo5MatchHistory_UnknownPlayer_Empty : un gamertag sans match → liste vide,
// pas d'erreur.
func TestHalo5MatchHistory_UnknownPlayer_Empty(t *testing.T) {
	src := newH5HistorySource(t, "Nobody")
	got, err := src.GetMatchSummaries(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMatchSummaries: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

// TestHalo5MatchHistory_EmptyGamertag_NeutralEmpty : source sans gamertag → vide
// neutre (pas de query, pas d'erreur).
func TestHalo5MatchHistory_EmptyGamertag_NeutralEmpty(t *testing.T) {
	mem := openMemSQL(t)
	seedH5SharedHistory(t, mem)
	src := NewHalo5MatchHistorySource(&memSharedReader{mem}, "   ")
	got, err := src.GetMatchSummaries(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMatchSummaries: %v", err)
	}
	if got != nil {
		t.Errorf("gamertag vide → nil, got %+v", got)
	}
}
