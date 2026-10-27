//go:build integration

// campaign_exclusion_e2_behavior_test.go — lecteurs de la Campagne rattrapés à l'étape E2
// du plan .ai/archive/V7.5/PLAN_BACKLOG_2026-10-15.md (DB-25, DB-26 du backlog 2026-09-26) : duel entre
// deux joueurs (kill-feed) et totaux de médailles. Même fixture que les cas B4 : un match
// d'arène (arena1) et un match de Campagne (camp1) pour A et B ; Halo 5 masque la Campagne,
// Halo Infinite (aucune variante masquée) reste neutre.

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/migration"
)

// newE2PlayerDB : fixture B4 + kill-feed (frags A/B) et médailles de A par match.
// arena1 : A tue B 2 fois, B tue A 1 fois. camp1 : A tue B 10 fois, B tue A 7 fois.
// Médailles de A : 100 (arena1 x1, camp1 x4), 200 (camp1 seulement, x3).
func newE2PlayerDB(t *testing.T, titleSlug string) *PlayerDB {
	t.Helper()
	pdb := newB4PlayerDB(t, titleSlug)
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)`,
	}
	for _, s := range stmts {
		if _, err := pdb.Shared.Exec(ctx, s); err != nil {
			t.Fatalf("fixture E2 : %v\n%s", err, s)
		}
	}
	// Le journal et ses résumés (ADR 0039) : schéma réel, une passe par match.
	if err := migration.EnsureMatchKillEvents(pdb.Shared.SQLDb()); err != nil {
		t.Fatalf("fixture E2 journal : %v", err)
	}
	var morts []ligneJournal
	insertN := func(match, killer, victim string, n int) {
		for i := 0; i < n; i++ {
			morts = append(morts, ligneJournal{match: match, publiable: true, tueur: killer, victime: victim,
				timeMS: len(morts)})
		}
	}
	insertN("arena1", b4XUIDA, b4XUIDB, 2)
	insertN("arena1", b4XUIDB, b4XUIDA, 1)
	insertN("camp1", b4XUIDA, b4XUIDB, 10)
	insertN("camp1", b4XUIDB, b4XUIDA, 7)
	poserJournal(t, pdb.Shared.SQLDb(), morts...)
	for _, m := range []struct {
		match string
		medal int64
		n     int
	}{{"arena1", 100, 1}, {"camp1", 100, 4}, {"camp1", 200, 3}} {
		if _, err := pdb.Shared.Exec(ctx, `INSERT INTO medals_earned VALUES (?, ?, ?, ?)`, m.match, b4XUIDA, m.medal, m.n); err != nil {
			t.Fatalf("fixture E2 médailles : %v", err)
		}
	}
	return pdb
}

// E2.1 — CompareRepo.GetEncounterStats : frags du duel A/B (QKillsBetweenPlayers).
func TestCampaignExclusion_EncounterKills(t *testing.T) {
	for _, tc := range b4Titres {
		enc, err := NewCompareRepo(newE2PlayerDB(t, tc.slug)).GetEncounterStats(context.Background(), b4XUIDA, b4XUIDB)
		if err != nil || enc == nil {
			t.Fatalf("%s : GetEncounterStats = %v, %v", tc.slug, enc, err)
		}
		if want := b4Attendu(tc.masquee, 2, 12); enc.KillsDealt != want {
			t.Errorf("%s : %d frags infligés, attendu %d (Campagne masquée : %v)", tc.slug, enc.KillsDealt, want, tc.masquee)
		}
		if want := b4Attendu(tc.masquee, 1, 8); enc.DeathsSuffered != want {
			t.Errorf("%s : %d morts subies, attendu %d (Campagne masquée : %v)", tc.slug, enc.DeathsSuffered, want, tc.masquee)
		}
	}
}

// E2.1 — ExplorerRepo.GetKillerVictimBetween : même duel, forme bornée par la liste de matchs.
func TestCampaignExclusion_ExplorerKillerVictim(t *testing.T) {
	for _, tc := range b4Titres {
		agg, err := NewExplorerRepo(newE2PlayerDB(t, tc.slug), b4XUIDA).GetKillerVictimBetween(context.Background(), b4XUIDA, b4XUIDB)
		if err != nil {
			t.Fatalf("%s : GetKillerVictimBetween : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, 2, 12); agg.KillsDealt != want {
			t.Errorf("%s : %d frags infligés, attendu %d", tc.slug, agg.KillsDealt, want)
		}
		if want := b4Attendu(tc.masquee, 1, 8); agg.DeathsSuffered != want {
			t.Errorf("%s : %d morts subies, attendu %d", tc.slug, agg.DeathsSuffered, want)
		}
	}
}

// E2.2 — Q36a : totaux de médailles, lus par CitationsRepo et MedalsRepo.
func TestCampaignExclusion_MedalTotals(t *testing.T) {
	type lecteur struct {
		nom  string
		lire func(pdb *PlayerDB) (map[int64]int64, error)
	}
	lecteurs := []lecteur{
		{"CitationsRepo", func(pdb *PlayerDB) (map[int64]int64, error) {
			rows, err := NewCitationsRepo(pdb).LoadMedalTotals(context.Background(), b4XUIDA)
			out := map[int64]int64{}
			for _, r := range rows {
				out[int64(r.MedalID)] = int64(r.TotalCount)
			}
			return out, err
		}},
		{"MedalsRepo", func(pdb *PlayerDB) (map[int64]int64, error) {
			rows, err := NewMedalsRepo(pdb).LoadMedalTotals(context.Background(), b4XUIDA)
			out := map[int64]int64{}
			for _, r := range rows {
				out[int64(r.MedalID)] = int64(r.TotalCount)
			}
			return out, err
		}},
	}
	for _, tc := range b4Titres {
		for _, l := range lecteurs {
			got, err := l.lire(newE2PlayerDB(t, tc.slug))
			if err != nil {
				t.Fatalf("%s %s : %v", tc.slug, l.nom, err)
			}
			want := b4Attendu(tc.masquee, map[int64]int64{100: 1}, map[int64]int64{100: 5, 200: 3})
			if len(got) != len(want) {
				t.Errorf("%s %s : %v, attendu %v", tc.slug, l.nom, got, want)
				continue
			}
			for id, n := range want {
				if got[id] != n {
					t.Errorf("%s %s : %v, attendu %v", tc.slug, l.nom, got, want)
					break
				}
			}
		}
	}
}
