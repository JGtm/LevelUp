//go:build integration

// Package duckdb — citations_cumul_test.go : le cumul d'une citation est arrêté à la date du
// match (tuile Home et vue match), jamais le total d'aujourd'hui.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run CitationCumul -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
)

const cumulTestNorm = "presque_ca_ne_compte_pas"

// newTestPlayerDBForCitationsCumul : topologie post-ADR 0016 (player et shared sur deux
// connexions distinctes). Historique de la citation (paliers 100, 250, 500), par ordre
// chronologique : mA +300 (cumul 300), mB +150 (450), mC +100 (550, franchit le dernier
// palier), mD +80 (630, déjà maîtrisée). L'ordre d'insertion est volontairement mélangé.
func newTestPlayerDBForCitationsCumul(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedPlayerSchema(t, player)
	seedSharedDBSchema(t, shared)
	seedMetaDBSchema(t, meta)
	ctx := context.Background()

	for _, q := range []string{`DELETE FROM match_citations`} {
		if _, err := player.Exec(ctx, q); err != nil {
			t.Fatalf("wipe player: %v", err)
		}
	}
	for _, q := range []string{`DELETE FROM shared.match_participants`, `DELETE FROM shared.match_registry`} {
		if _, err := shared.Exec(ctx, q); err != nil {
			t.Fatalf("wipe shared: %v", err)
		}
	}
	for _, m := range []struct{ id, start string }{
		{"mC", "2025-01-20 12:00:00+00"},
		{"mA", "2025-01-01 12:00:00+00"},
		{"mD", "2025-02-01 12:00:00+00"},
		{"mB", "2025-01-10 12:00:00+00"},
	} {
		if _, err := shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, start_time_utc) VALUES (?, ?, ?)`,
			m.id, m.start, m.start); err != nil {
			t.Fatalf("insert registry %s: %v", m.id, err)
		}
	}
	for _, c := range []struct {
		id    string
		value int
	}{{"mC", 100}, {"mA", 300}, {"mD", 80}, {"mB", 150}} {
		if _, err := player.Exec(ctx,
			`INSERT INTO match_citations (match_id, citation_name_norm, value) VALUES (?,?,?)`,
			c.id, cumulTestNorm, c.value); err != nil {
			t.Fatalf("insert citation %s: %v", c.id, err)
		}
	}
	if _, err := meta.Exec(ctx,
		`INSERT INTO citation_mappings (citation_name_norm, mapping_type, tier_targets, enabled)
		 VALUES (?,?,?,?)`,
		cumulTestNorm, "stat", "100,250,500", true); err != nil {
		t.Fatalf("insert mapping: %v", err)
	}
	insertCitationTexts(t, meta, cumulTestNorm, i18n.Text{i18n.LocaleFR: "Presque, ça ne compte pas"}, nil)
	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		Metadata:     meta,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// Tuile Home : chaque match porte le cumul d'alors. Seul le match qui franchit le dernier
// palier est « nouvellement maîtrisé » ; le suivant est filtré ; les précédents montrent leur
// progression d'alors. (Avec le total d'aujourd'hui, 630, mA et mB passaient pour maîtrisés.)
func TestHomeRepo_LoadMatchCitations_CumulArreteAuMatch(t *testing.T) {
	repo := NewHomeRepo(newTestPlayerDBForCitationsCumul(t))
	got, err := repo.LoadMatchCitations(context.Background(), []string{"mA", "mB", "mC", "mD"})
	if err != nil {
		t.Fatalf("LoadMatchCitations: %v", err)
	}
	want := map[string]int{"mA": 300, "mB": 450, "mC": 550, "mD": 630}
	for id, cum := range want {
		if len(got[id]) != 1 {
			t.Fatalf("%s : %d citations, want 1", id, len(got[id]))
		}
		if got[id][0].Cumulative != cum {
			t.Errorf("%s : cumul = %d, want %d", id, got[id][0].Cumulative, cum)
		}
	}

	var newly []string
	for _, id := range []string{"mA", "mB", "mC", "mD"} {
		snippets := analysis.BuildCitationSnippets(got[id], 3)
		if id == "mD" {
			if len(snippets) != 0 {
				t.Errorf("mD (déjà maîtrisée avant) doit être filtré, obtenu %d snippet(s)", len(snippets))
			}
			continue
		}
		if len(snippets) != 1 {
			t.Fatalf("%s : %d snippets, want 1", id, len(snippets))
		}
		if snippets[0].IsNewlyMastered {
			newly = append(newly, id)
		}
	}
	if len(newly) != 1 || newly[0] != "mC" {
		t.Errorf("nouvellement maîtrisée = %v, want [mC]", newly)
	}
}

// Un lot réduit à un match ancien ne voit pas la progression d'aujourd'hui : le cumul vient
// de tout l'historique, pas du lot demandé.
func TestHomeRepo_LoadMatchCitations_CumulIndependantDuLot(t *testing.T) {
	repo := NewHomeRepo(newTestPlayerDBForCitationsCumul(t))
	got, err := repo.LoadMatchCitations(context.Background(), []string{"mB"})
	if err != nil {
		t.Fatalf("LoadMatchCitations: %v", err)
	}
	if len(got["mB"]) != 1 || got["mB"][0].Cumulative != 450 {
		t.Fatalf("mB seul : %+v, want cumul 450", got["mB"])
	}
}

// Vue match (onglet citations) : même cumul que la tuile Home.
func TestCitationsRepo_LoadMatchCitationsRich_CumulArreteAuMatch(t *testing.T) {
	repo := NewCitationsRepo(newTestPlayerDBForCitationsCumul(t))
	want := map[string]int{"mA": 300, "mB": 450, "mC": 550, "mD": 630}
	for id, cum := range want {
		rows, err := repo.LoadMatchCitationsRich(context.Background(), id)
		if err != nil || len(rows) != 1 {
			t.Fatalf("%s : err=%v n=%d", id, err, len(rows))
		}
		if rows[0].Cumulative != cum {
			t.Errorf("%s : cumul = %d, want %d", id, rows[0].Cumulative, cum)
		}
	}
}

// Départage à heure égale : match_id croissant, et le même résultat à chaque lecture.
func TestHomeRepo_LoadMatchCitations_DepartageAHeureEgale(t *testing.T) {
	pdb := newTestPlayerDBForCitationsCumul(t)
	ctx := context.Background()
	for _, id := range []string{"mE2", "mE1"} {
		if _, err := pdb.Shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time, start_time_utc) VALUES (?,?,?)`,
			id, "2025-03-01 12:00:00+00", "2025-03-01 12:00:00+00"); err != nil {
			t.Fatalf("insert registry %s: %v", id, err)
		}
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO match_citations (match_id, citation_name_norm, value) VALUES (?,?,?)`,
			id, cumulTestNorm, 10); err != nil {
			t.Fatalf("insert citation %s: %v", id, err)
		}
	}
	repo := NewHomeRepo(pdb)
	for i := 0; i < 3; i++ {
		got, err := repo.LoadMatchCitations(ctx, []string{"mE1", "mE2"})
		if err != nil {
			t.Fatalf("LoadMatchCitations: %v", err)
		}
		if got["mE1"][0].Cumulative != 640 || got["mE2"][0].Cumulative != 650 {
			t.Fatalf("passe %d : mE1=%d mE2=%d, want 640 et 650", i, got["mE1"][0].Cumulative, got["mE2"][0].Cumulative)
		}
	}
}

// Sans lecteur partagé, on ne sert pas un cumul faux : aucune citation.
func TestHomeRepo_LoadMatchCitations_SansHeuresPasDeCitationFausse(t *testing.T) {
	pdb := newTestPlayerDBForCitationsCumul(t)
	pdb.SharedReader = nil
	pdb.Shared = nil
	got, err := NewHomeRepo(pdb).LoadMatchCitations(context.Background(), []string{"mA", "mD"})
	if err != nil {
		t.Fatalf("LoadMatchCitations: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("sans heures de match : %d match(s) servis, want 0", len(got))
	}
}
