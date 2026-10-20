//go:build integration

// Package prestige — campaign_exclusion_behavior_test.go : les lecteurs Prestige des
// matchs d'un joueur (baseline, cumul, escouade) excluent la Campagne (backlog
// 2026-09-26, lot B4, D-5 ; règle produit du 2026-07-18).
//
// Fixture : un match d'arène et un match de Campagne PLUS RÉCENT, joués par A et B dans
// la même équipe. Halo 5 : la Campagne disparaît. Halo Infinite (aucune variante masquée) :
// le résolveur est neutre, les deux matchs restent.
package prestige

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
)

const (
	b4XUIDA = "2533274800000001"
	b4XUIDB = "2533274800000002"
)

var b4Titres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}}

func b4Attendu[T any](masquee bool, sans, avec T) T {
	if masquee {
		return sans
	}
	return avec
}

// newB4SharedReader : A a 5 frags en arène (playlist « pl-arene »), 50 en Campagne
// (playlist « pl-campagne », pour rendre la fuite visible dans l'indice d'escouade).
func newB4SharedReader(t *testing.T) *squadTestSharedReader {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	stmts := []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			game_variant_id VARCHAR, playlist_id VARCHAR, playlist_name VARCHAR, playlist_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`,
		`INSERT INTO match_registry VALUES
			('arena1', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', 'aaaaaaaa-0000-0000-0000-000000000001',
			 'pl-arene', 'Arene', '', 'Slayer', ''),
			('camp1',  '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', '` + camp + `',
			 'pl-campagne', 'Campagne', '', 'Campaign', '')`,
		`INSERT INTO match_participants VALUES
			('arena1', '` + b4XUIDA + `', 5), ('arena1', '` + b4XUIDB + `', 3),
			('camp1',  '` + b4XUIDA + `', 50), ('camp1', '` + b4XUIDB + `', 40)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture B4 : %v\n%s", err, s)
		}
	}
	return &squadTestSharedReader{db: db}
}

// B4.2.5 — HaloBaselineProvider.RecentMatches.
func TestCampaignExclusion_BaselineRecentMatches(t *testing.T) {
	for _, tc := range b4Titres {
		got, err := NewHaloBaselineProvider(newB4SharedReader(t), b4XUIDA).
			RecentMatches(context.Background(), tc.slug, "kills", 20)
		if err != nil {
			t.Fatalf("%s : RecentMatches : %v", tc.slug, err)
		}
		ids := make([]string, 0, len(got))
		for _, m := range got {
			ids = append(ids, m.MatchID)
		}
		if want := b4Attendu(tc.masquee, "[arena1]", "[camp1 arena1]"); fmt.Sprint(ids) != want {
			t.Errorf("%s : matchs %v, attendu %s", tc.slug, ids, want)
		}
	}
}

// B4.2.5 — HaloBaselineProvider.CumulativeSince.
func TestCampaignExclusion_BaselineCumulativeSince(t *testing.T) {
	for _, tc := range b4Titres {
		total, n, err := NewHaloBaselineProvider(newB4SharedReader(t), b4XUIDA).
			CumulativeSince(context.Background(), tc.slug, "kills", time.Time{})
		if err != nil {
			t.Fatalf("%s : CumulativeSince : %v", tc.slug, err)
		}
		wantTotal, wantN := b4Attendu(tc.masquee, 5.0, 55.0), b4Attendu(tc.masquee, 1, 2)
		if total != wantTotal || n != wantN {
			t.Errorf("%s : cumul %v sur %d match(s), attendu %v sur %d", tc.slug, total, n, wantTotal, wantN)
		}
	}
}

// B4.2.8 — PrestigeSquadMatchProvider.candidateMatches (via SquadMatchMetrics, `xuid IN`).
func TestCampaignExclusion_SquadMatchMetrics(t *testing.T) {
	roster := []string{b4XUIDA, b4XUIDB}
	for _, tc := range b4Titres {
		got, err := NewPrestigeSquadMatchProvider(newB4SharedReader(t)).
			SquadMatchMetrics(context.Background(), roster, tc.slug, "kills", 50, time.Time{})
		if err != nil {
			t.Fatalf("%s : SquadMatchMetrics : %v", tc.slug, err)
		}
		ids := make([]string, 0, len(got))
		for _, m := range got {
			ids = append(ids, m.MatchID)
		}
		if want := b4Attendu(tc.masquee, 1, 2); len(ids) != want {
			t.Errorf("%s : matchs d'escouade %v, attendu %d", tc.slug, ids, want)
		}
	}
}

// B4.2.8 — PrestigeSquadMatchProvider.SquadUsualContexts (`xuid IN`, roster).
func TestCampaignExclusion_SquadUsualContexts(t *testing.T) {
	roster := []string{b4XUIDA, b4XUIDB}
	for _, tc := range b4Titres {
		playlists, _, err := NewPrestigeSquadMatchProvider(newB4SharedReader(t)).
			SquadUsualContexts(context.Background(), roster, tc.slug, 60)
		if err != nil {
			t.Fatalf("%s : SquadUsualContexts : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, 1, 2); len(playlists) != want {
			t.Errorf("%s : playlists usuelles %v, attendu %d", tc.slug, playlists, want)
		}
	}
}
