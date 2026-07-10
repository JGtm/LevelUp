// Package duckdb — match_view_history_bulk_test.go : correctness de J3
// (GetHistoryForAvgBulk). La variante multi-xuid doit renvoyer, pour chaque xuid,
// EXACTEMENT le même ensemble de lignes d'historique que GetHistoryForAvg unitaire
// (comparaison en multiset — l'ordre est indifférent, l'historique alimente des
// moyennes). Sans DB réelle, fixture in-memory (root-level, comme SharedReader).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
)

func newHistAvgBulkTestPDB(t *testing.T) *PlayerDB {
	t.Helper()
	playerSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { playerSQL.Close() })
	player := newTestDB(playerSQL, ":memory:")
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			pair_name VARCHAR, is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, kills INTEGER, deaths INTEGER, assists INTEGER,
			headshot_kills INTEGER, max_killing_spree INTEGER)`,
		`CREATE TABLE medals_earned (
			medal_id UBIGINT, medal_name_id UBIGINT, xuid VARCHAR, match_id VARCHAR, count INTEGER)`,
	} {
		if _, err := player.Exec(ctx, q); err != nil {
			t.Fatalf("seed schema: %v\nSQL: %s", err, q)
		}
	}
	return &PlayerDB{
		Player:       player,
		SharedReader: LegacySharedReader(player),
		XUID:         "xa",
		Gamertag:     "gt",
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

func sortHistRows(rows []domain.MatchHistAvgRow) {
	sort.Slice(rows, func(i, j int) bool {
		return histRowKey(rows[i]) < histRowKey(rows[j])
	})
}

func histRowKey(r domain.MatchHistAvgRow) string {
	return fmt.Sprintf("%d|%d|%d|%d|%d|%d|%s|%t|%t|%d",
		r.Kills, r.Deaths, r.Assists, r.HeadshotKills, r.MaxKillingSpree,
		r.PerfectKills, r.PairName, r.IsFirefight, r.IsRanked, r.DurationSeconds)
}

func TestGetHistoryForAvgBulk_EqualsSinglePerXUID(t *testing.T) {
	pdb := newHistAvgBulkTestPDB(t)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO match_registry (match_id, start_time, pair_name, duration_seconds) VALUES
			('m1','2026-01-01 10:00:00','Slayer',300),
			('m2','2026-01-02 10:00:00','Slayer',300),
			('m3','2026-01-03 10:00:00','Oddball',300),
			('m4','2026-01-01 11:00:00','CTF',300),
			('m5','2026-01-02 11:00:00','Strongholds',300)`,
		`INSERT INTO match_participants (match_id, xuid, kills, deaths, assists, headshot_kills, max_killing_spree) VALUES
			('m1','xa',2,3,1,1,2),
			('m2','xa',4,2,0,2,3),
			('m3','xa',1,5,2,0,1),
			('m4','xb',5,1,3,3,4),
			('m5','xb',3,4,1,1,2)`,
		`INSERT INTO medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES
			(1, 1, 'xa', 'm1', 1), (2, 2, 'xb', 'm4', 2)`,
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("seed data: %v\nSQL: %s", err, q)
		}
	}

	repo := NewMatchViewRepo(pdb, "xa")
	bulk, err := repo.GetHistoryForAvgBulk(ctx, []string{"xa", "xb"})
	if err != nil {
		t.Fatalf("GetHistoryForAvgBulk: %v", err)
	}
	if len(bulk) != 2 {
		t.Fatalf("bulk contient %d xuids, attendu 2 (%v)", len(bulk), bulk)
	}
	for _, xuid := range []string{"xa", "xb"} {
		single, err := repo.GetHistoryForAvg(ctx, xuid)
		if err != nil {
			t.Fatalf("GetHistoryForAvg(%s): %v", xuid, err)
		}
		got := bulk[xuid]
		if len(got) != len(single) {
			t.Fatalf("bulk[%s] a %d lignes, single %d", xuid, len(got), len(single))
		}
		sortHistRows(got)
		sortHistRows(single)
		if !reflect.DeepEqual(got, single) {
			t.Errorf("bulk[%s] != single (multiset)\n bulk=%+v\n single=%+v", xuid, got, single)
		}
	}
	if got := len(bulk["xa"]); got != 3 {
		t.Errorf("xa: %d matchs, attendu 3", got)
	}
	if got := len(bulk["xb"]); got != 2 {
		t.Errorf("xb: %d matchs, attendu 2", got)
	}
}

func TestGetHistoryForAvgBulk_EmptyInput(t *testing.T) {
	pdb := newHistAvgBulkTestPDB(t)
	got, err := NewMatchViewRepo(pdb, "xa").GetHistoryForAvgBulk(context.Background(), nil)
	if err != nil {
		t.Fatalf("bulk(nil): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("bulk(nil) = %v, attendu vide", got)
	}
}
